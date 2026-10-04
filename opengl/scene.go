package opengl

import (
	"math"
	"sort"
	"time"
	"unsafe"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// SceneRenderer draws a scene.Scene. Meshes and instances are uploaded once and kept on the
// GPU; they are uploaded again only when marked Dirty (for simulations).
// It must be created and used on the thread that owns the OpenGL context.
type SceneRenderer struct {
	Shadows    ShadowSettings
	Profiling  bool            // measure CPU and GPU times into Statistics (waits for the GPU every frame)
	Statistics FrameStatistics // the last frame drawn
	profiler   *passProfiler

	sceneProgram     *shaderProgram
	sky              *skyRenderer
	waterPlane       *scene.Mesh
	waterFloor       *scene.Mesh
	shadow           *shadowMap
	levelVariation   *levelVariationTexture
	seabed           *seabedMap
	measured         *scene.Scene
	meshMeasures     map[*scene.Mesh]*meshMeasure
	instancedDetails map[*scene.Instances]*instancedDetail
	verticalScale    float32 // of the frame being drawn
	counters         [passCount]drawCounter
	sceneSize        float32 // horizontal size of the measured scene
	sceneHeight      float32 // vertical span of the measured scene (before vertical scaling)
}

// shaderProgram is a linked program with its uniform locations cached by name.
type shaderProgram struct {
	handle    uint32
	locations map[string]int32
}

type meshBuffers struct {
	vertexArray, vertexBuffer, indexBuffer uint32
	elementCount                           int32
	indexed                                bool
}

type instanceBuffers struct {
	vertexArray, instanceBuffer uint32
	instanceCount               int32
	mesh                        *meshBuffers
}

// drawCommand is one node to draw this frame.
type drawCommand struct {
	node                  *scene.Node
	world                 graphics.Mat4
	material              *scene.Material
	squaredCameraDistance float32
	transparent           bool
	seaSurface            bool // the environment's water plane: the only surface the level variation moves
}

// cameraMatrices are the matrices of one frame.
type cameraMatrices struct {
	projection, view, viewProjection graphics.Mat4
}

const floatsPerVertex = 10   // position 3 + normal 3 + color 4
const floatsPerInstance = 20 // matrix 16 + color 4
const bytesPerFloat = 4

// NewSceneRenderer compiles the scene shaders. Call it after the OpenGL context exists.
func NewSceneRenderer() (*SceneRenderer, error) {
	sceneProgram, err := newShaderProgram(graphics.SceneVertexShader, graphics.SceneFragmentShader)
	if err != nil {
		return nil, err
	}
	sky, err := newSkyRenderer()
	if err != nil {
		return nil, err
	}
	renderer := &SceneRenderer{sceneProgram: sceneProgram, sky: sky, Shadows: DefaultShadowSettings(), profiler: newPassProfiler(),
		meshMeasures: map[*scene.Mesh]*meshMeasure{}, instancedDetails: map[*scene.Instances]*instancedDetail{}}
	if renderer.shadow, err = newShadowMap(renderer.Shadows.resolution()); err != nil {
		return nil, err
	}
	if renderer.seabed, err = newSeabedMap(seabedResolution); err != nil {
		return nil, err
	}
	renderer.levelVariation = newLevelVariationTexture()
	return renderer, nil
}

func newShaderProgram(vertexSource, fragmentSource string) (*shaderProgram, error) {
	vertexShader, err := CompileShader(vertexSource, gl.VERTEX_SHADER)
	if err != nil {
		return nil, err
	}
	fragmentShader, err := CompileShader(fragmentSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return nil, err
	}
	handle, err := LinkProgram(vertexShader, fragmentShader)
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
	if err != nil {
		return nil, err
	}
	return &shaderProgram{handle: handle, locations: map[string]int32{}}, nil
}

func (program *shaderProgram) location(name string) int32 {
	if location, cached := program.locations[name]; cached {
		return location
	}
	location := gl.GetUniformLocation(program.handle, gl.Str(name+"\x00"))
	program.locations[name] = location
	return location
}

func (program *shaderProgram) setVector3(name string, value graphics.Vec4) {
	gl.Uniform3f(program.location(name), value[0], value[1], value[2])
}

func (program *shaderProgram) setVector4(name string, value graphics.Vec4) {
	gl.Uniform4fv(program.location(name), 1, &value[0])
}

func (program *shaderProgram) setMatrix(name string, value graphics.Mat4) {
	gl.UniformMatrix4fv(program.location(name), 1, false, &value[0])
}

func (program *shaderProgram) setFloat(name string, value float32) {
	gl.Uniform1f(program.location(name), value)
}

func (program *shaderProgram) setInteger(name string, value int32) {
	gl.Uniform1i(program.location(name), value)
}

// Draw renders the scene from the camera into the current framebuffer of the given size.
// seconds is the elapsed time, which animates water and waterfalls.
func (renderer *SceneRenderer) Draw(world *scene.Scene, camera models.Camera, width, height int, seconds float64) {
	started := time.Now()
	matrices := computeCameraMatrices(camera, width, height)
	light := world.Environment.Daylight()
	renderer.startFrame(world)
	fieldOfView, _, _ := camera.Projection()
	mainView := newViewpoint(matrices.viewProjection, camera.Position, fieldOfView)
	commands := renderer.collectDrawCommands(world, camera)
	sortOpaqueBeforeTransparent(commands)
	renderer.measurePass("seabed", func() { renderer.seabed.update(world, commands) })
	var region shadowRegion
	var err error
	renderer.measurePass("shadows", func() {
		region, err = renderer.renderShadows(world, camera, commands, light.LightDirection, seconds, mainView)
	})
	shadowsReady := err == nil && renderer.Shadows.Enabled
	clearFrame(light, width, height)
	renderer.setFrameUniforms(world.Environment, light, camera, matrices, seconds)
	renderer.shadow.bindForLighting(renderer.sceneProgram, region, shadowsReady)
	renderer.seabed.bindForWater(renderer.sceneProgram, world.Environment.Water)
	renderer.levelVariation.bind(renderer.sceneProgram, world.Environment.Water)
	opaque, transparent := splitOpaqueAndTransparent(commands)
	renderer.measurePass("scene", func() { renderer.executeDrawCommands(opaque, mainView) })
	renderer.measurePass("sky", func() { renderer.sky.draw(world.Environment, light, camera, matrices, seconds) })
	renderer.measurePass("transparent", func() { renderer.executeDrawCommands(transparent, mainView) })
	renderer.recordStatistics(time.Since(started))
}

// measurePass runs a pass, timing it on the GPU when profiling.
func (renderer *SceneRenderer) measurePass(pass string, run func()) {
	if !renderer.Profiling {
		run()
		return
	}
	renderer.profiler.begin(pass)
	run()
	renderer.profiler.end()
}

// startFrame resets the counters and remembers what every draw of the frame shares.
func (renderer *SceneRenderer) startFrame(world *scene.Scene) {
	renderer.verticalScale = world.Environment.EffectiveVerticalScale()
	renderer.counters = [passCount]drawCounter{}
}

func (renderer *SceneRenderer) recordStatistics(processorTime time.Duration) {
	main, shadow := renderer.counters[mainPass], renderer.counters[shadowPass]
	statistics := FrameStatistics{CPU: processorTime,
		DrawCalls: main.drawCalls, Triangles: main.triangles, Instances: main.instances,
		ShadowDrawCalls: shadow.drawCalls, ShadowTriangles: shadow.triangles}
	if renderer.Profiling {
		statistics.GPU = renderer.profiler.collect()
	}
	renderer.Statistics = statistics
}

// shadowLevelBias makes casters one level of detail coarser than what the camera sees: a shadow
// shows an outline, and the shadow pass covers more than the view (what is behind the camera too).
const shadowLevelBias = 1

// renderShadows draws the shadow map for this frame when shadows are enabled.
func (renderer *SceneRenderer) renderShadows(world *scene.Scene, camera models.Camera, commands []drawCommand, lightDirection graphics.Vec4, seconds float64, mainView viewpoint) (shadowRegion, error) {
	if !renderer.Shadows.Enabled {
		return shadowRegion{}, nil
	}
	if err := renderer.matchShadowResolution(); err != nil {
		return shadowRegion{}, err
	}
	renderer.measureScene(world)
	verticalScale := world.Environment.EffectiveVerticalScale()
	extent := renderer.Shadows.extent(camera, renderer.sceneSize)
	depthRange := extent*2 + renderer.sceneHeight*verticalScale*2
	region := fitShadowRegion(lightDirection, shadowCenter(camera, extent), extent, depthRange, renderer.shadow.resolution)
	shadowView := mainView
	shadowView.volume = frustumFromMatrix(region.lightViewProjection())
	shadowView.levelBias += shadowLevelBias
	drawCaster := func(program *shaderProgram, command drawCommand) {
		renderer.drawGeometry(program, command, shadowView, shadowPass, &renderer.counters[shadowPass])
	}
	renderer.shadow.renderShadowPass(commands, region, verticalScale, world.Environment.Wind, seconds, drawCaster)
	return region, nil
}

// matchShadowResolution recreates the shadow map when the requested resolution changes.
func (renderer *SceneRenderer) matchShadowResolution() error {
	wanted := renderer.Shadows.resolution()
	if renderer.shadow != nil && renderer.shadow.resolution == wanted {
		return nil
	}
	shadow, err := newShadowMap(wanted)
	if err != nil {
		return err
	}
	renderer.shadow = shadow
	return nil
}

// measureScene caches the size of the scene, used to size the shadow region.
func (renderer *SceneRenderer) measureScene(world *scene.Scene) {
	if renderer.measured == world {
		return
	}
	minimum, maximum := world.Bounds()
	renderer.sceneSize = float32(math.Max(float64(maximum[0]-minimum[0]), float64(maximum[2]-minimum[2])))
	renderer.sceneHeight = maximum[1] - minimum[1]
	if renderer.sceneSize <= 0 || math.IsInf(float64(renderer.sceneSize), 0) {
		renderer.sceneSize, renderer.sceneHeight = 1, 1
	}
	renderer.measured = world
}

func computeCameraMatrices(camera models.Camera, width, height int) cameraMatrices {
	fieldOfView, near, far := camera.Projection()
	aspect := float64(width) / math.Max(float64(height), 1)
	projection := graphics.PerspectiveOpenGL(fieldOfView, aspect, float64(near), float64(far))
	view := graphics.LookAt(camera.Position, camera.Target, camera.Up)
	return cameraMatrices{projection: projection, view: view, viewProjection: projection.Multiply(view)}
}

func clearFrame(light scene.Daylight, width, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
	horizon := light.Horizon
	gl.ClearColor(horizon[0], horizon[1], horizon[2], 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.Disable(gl.CULL_FACE)
}

// setFrameUniforms sets what every draw of the frame shares; the light is the sun or the moon.
func (renderer *SceneRenderer) setFrameUniforms(environment scene.Environment, light scene.Daylight, camera models.Camera, matrices cameraMatrices, seconds float64) {
	program := renderer.sceneProgram
	gl.UseProgram(program.handle)
	program.setMatrix("projection", matrices.projection)
	program.setMatrix("view", matrices.view)
	program.setFloat("verticalScale", environment.EffectiveVerticalScale())
	program.setVector3("sunDirection", light.LightDirection)
	program.setVector3("sunColor", light.LightColor)
	program.setVector3("skyColor", light.SkyLight)
	program.setVector3("horizonColor", light.Horizon)
	program.setVector3("groundColor", light.GroundLight)
	program.setFloat("ambient", environment.Ambient)
	program.setVector3("fogColor", light.Fog)
	program.setFloat("brightness", light.Brightness)
	setWind(program, environment.Wind)
	gl.Uniform2f(program.location("fogRange"), environment.FogNear, environment.FogFar)
	setSeason(program, environment)
	program.setVector3("cameraPosition", camera.Position)
	program.setFloat("time", float32(seconds))
	program.setFloat("waveLength", waveLengthFor(environment.Water))
}

// setSeason sets the snow and the colour of the foliage.
func setSeason(program *shaderProgram, environment scene.Environment) {
	program.setVector4("foliageTint", environment.FoliageTint)
	if environment.Snow == nil {
		program.setInteger("snowEnabled", 0)
		return
	}
	program.setInteger("snowEnabled", 1)
	program.setFloat("snowLevel", environment.Snow.Level)
	program.setFloat("snowBlend", max(environment.Snow.Blend, 1e-6))
	program.setVector3("snowColor", environment.Snow.Color)
}

func waveLengthFor(water *scene.Water) float32 {
	if water == nil || water.EffectiveWaveLength() <= 0 {
		return 1
	}
	return water.EffectiveWaveLength()
}

func isTransparent(material *scene.Material) bool {
	return material.Transparent || material.Kind == scene.KindWater || material.Kind == scene.KindWaterfall
}

func (renderer *SceneRenderer) collectDrawCommands(world *scene.Scene, camera models.Camera) []drawCommand {
	var commands []drawCommand
	world.Walk(func(node *scene.Node, transform graphics.Mat4) {
		if node.Mesh == nil {
			return
		}
		material := materialOrDefault(node.Mesh.Material)
		commands = append(commands, drawCommand{
			node: node, world: transform, material: material, transparent: isTransparent(material),
			squaredCameraDistance: squaredHorizontalDistance(transform.Apply(graphics.Vec4{0, 0, 0, 1}), camera.Position),
		})
	})
	if world.Environment.Water != nil {
		commands = append(commands, renderer.waterFloorCommand(world.Environment.Water, camera))
		commands = append(commands, renderer.waterCommand(world.Environment.Water, camera))
	}
	return commands
}

func materialOrDefault(material *scene.Material) *scene.Material {
	if material == nil {
		return &scene.Material{BaseColor: graphics.Vec4{1, 1, 1, 1}}
	}
	return material
}

func squaredHorizontalDistance(first, second graphics.Vec4) float32 {
	deltaX, deltaZ := first[0]-second[0], first[2]-second[2]
	return deltaX*deltaX + deltaZ*deltaZ
}

// sortOpaqueBeforeTransparent keeps opaque commands in order and draws transparent ones last,
// farthest first, so blending composes correctly.
func sortOpaqueBeforeTransparent(commands []drawCommand) {
	sort.SliceStable(commands, func(first, second int) bool {
		if commands[first].transparent != commands[second].transparent {
			return !commands[first].transparent
		}
		return commands[first].transparent && commands[first].squaredCameraDistance > commands[second].squaredCameraDistance
	})
}

// splitOpaqueAndTransparent cuts the sorted commands where the transparent ones start.
func splitOpaqueAndTransparent(commands []drawCommand) (opaque, transparent []drawCommand) {
	for index, command := range commands {
		if command.transparent {
			return commands[:index], commands[index:]
		}
	}
	return commands, nil
}

func (renderer *SceneRenderer) executeDrawCommands(commands []drawCommand, view viewpoint) {
	gl.UseProgram(renderer.sceneProgram.handle)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
	blending := false
	for _, command := range commands {
		if command.transparent && !blending {
			enableBlending()
			blending = true
		}
		renderer.drawNode(command, view)
	}
	if blending {
		disableBlending()
	}
	gl.BindVertexArray(0)
}

func enableBlending() {
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.DepthMask(false)
}

func disableBlending() {
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}

func (renderer *SceneRenderer) drawNode(command drawCommand, view viewpoint) {
	program := renderer.sceneProgram
	baseColor := command.material.BaseColor
	if baseColor == (graphics.Vec4{}) {
		baseColor = graphics.Vec4{1, 1, 1, 1}
	}
	program.setVector4("baseColor", baseColor)
	program.setInteger("kind", int32(command.material.Kind))
	program.setInteger("pattern", int32(command.material.Pattern))
	program.setInteger("foliage", boolToInteger(command.material.Foliage))
	program.setFloat("patternScale", command.material.PatternScale)
	program.setInteger("levelVariationEnabled", boolToInteger(command.seaSurface && renderer.levelVariation.active))
	program.setInteger("seaSurface", boolToInteger(command.seaSurface))
	renderer.drawGeometry(program, command, view, mainPass, &renderer.counters[mainPass])
}

func drawSingle(program *shaderProgram, buffers *meshBuffers) {
	program.setInteger("instanced", 0)
	gl.BindVertexArray(buffers.vertexArray)
	if buffers.indexed {
		gl.DrawElementsWithOffset(gl.TRIANGLES, buffers.elementCount, gl.UNSIGNED_INT, 0)
	} else {
		gl.DrawArrays(gl.TRIANGLES, 0, buffers.elementCount)
	}
}

func drawInstanced(program *shaderProgram, buffers *meshBuffers, instances *instanceBuffers) {
	program.setInteger("instanced", 1)
	gl.BindVertexArray(instances.vertexArray)
	if buffers.indexed {
		gl.DrawElementsInstanced(gl.TRIANGLES, buffers.elementCount, gl.UNSIGNED_INT, unsafe.Pointer(nil), instances.instanceCount)
	} else {
		gl.DrawArraysInstanced(gl.TRIANGLES, 0, buffers.elementCount, instances.instanceCount)
	}
}

// uploadMesh returns the GPU buffers of a mesh, creating or refreshing them when needed.
func uploadMesh(mesh *scene.Mesh) *meshBuffers {
	buffers, _ := mesh.GPU.(*meshBuffers)
	if buffers != nil && !mesh.Dirty {
		return buffers
	}
	if buffers == nil {
		buffers = newMeshBuffers()
	}
	if len(mesh.Normals) != len(mesh.Positions) {
		mesh.ComputeNormals()
	}
	gl.BindVertexArray(buffers.vertexArray)
	fillVertexBuffer(buffers.vertexBuffer, interleaveVertices(mesh))
	describeVertexLayout()
	fillIndexBuffer(buffers, mesh)
	gl.BindVertexArray(0)
	mesh.GPU, mesh.Dirty = buffers, false
	return buffers
}

func newMeshBuffers() *meshBuffers {
	buffers := &meshBuffers{}
	gl.GenVertexArrays(1, &buffers.vertexArray)
	gl.GenBuffers(1, &buffers.vertexBuffer)
	gl.GenBuffers(1, &buffers.indexBuffer)
	return buffers
}

// interleaveVertices packs position, normal and color per vertex (white when there are no colors).
func interleaveVertices(mesh *scene.Mesh) []float32 {
	vertexCount := mesh.VertexCount()
	data := make([]float32, vertexCount*floatsPerVertex)
	for vertex := 0; vertex < vertexCount; vertex++ {
		offset := vertex * floatsPerVertex
		copy(data[offset:offset+3], mesh.Positions[3*vertex:3*vertex+3])
		copy(data[offset+3:offset+6], mesh.Normals[3*vertex:3*vertex+3])
		if len(mesh.Colors) >= 4*(vertex+1) {
			copy(data[offset+6:offset+10], mesh.Colors[4*vertex:4*vertex+4])
		} else {
			copy(data[offset+6:offset+10], []float32{1, 1, 1, 1})
		}
	}
	return data
}

func fillVertexBuffer(buffer uint32, data []float32) {
	gl.BindBuffer(gl.ARRAY_BUFFER, buffer)
	if len(data) > 0 {
		gl.BufferData(gl.ARRAY_BUFFER, len(data)*bytesPerFloat, gl.Ptr(data), gl.STATIC_DRAW)
	}
}

func describeVertexLayout() {
	stride := int32(floatsPerVertex * bytesPerFloat)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 3, gl.FLOAT, false, stride, 3*bytesPerFloat)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 4, gl.FLOAT, false, stride, 6*bytesPerFloat)
}

func fillIndexBuffer(buffers *meshBuffers, mesh *scene.Mesh) {
	buffers.indexed = len(mesh.Indices) > 0
	if !buffers.indexed {
		buffers.elementCount = int32(mesh.VertexCount())
		return
	}
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, buffers.indexBuffer)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(mesh.Indices)*4, gl.Ptr(mesh.Indices), gl.STATIC_DRAW)
	buffers.elementCount = int32(len(mesh.Indices))
}

// uploadInstances returns the GPU buffers of an instance set; each set has its own vertex array
// that reuses the mesh buffers, so one mesh can be instanced by several nodes.
func uploadInstances(instances *scene.Instances, mesh *meshBuffers) *instanceBuffers {
	buffers, _ := instances.GPU.(*instanceBuffers)
	if buffers != nil && !instances.Dirty && buffers.mesh == mesh {
		return buffers
	}
	if buffers == nil {
		buffers = &instanceBuffers{}
		gl.GenVertexArrays(1, &buffers.vertexArray)
		gl.GenBuffers(1, &buffers.instanceBuffer)
	}
	buffers.mesh, buffers.instanceCount = mesh, int32(len(instances.Transforms))
	gl.BindVertexArray(buffers.vertexArray)
	gl.BindBuffer(gl.ARRAY_BUFFER, mesh.vertexBuffer)
	describeVertexLayout()
	if mesh.indexed {
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, mesh.indexBuffer)
	}
	fillInstanceBuffer(buffers.instanceBuffer, packInstances(instances))
	describeInstanceLayout()
	gl.BindVertexArray(0)
	instances.GPU, instances.Dirty = buffers, false
	return buffers
}

// packInstances writes the matrix and the color of every instance (white when missing).
func packInstances(instances *scene.Instances) []float32 {
	data := make([]float32, len(instances.Transforms)*floatsPerInstance)
	for instance := range instances.Transforms {
		packInstance(instances, instance, data[instance*floatsPerInstance:])
	}
	return data
}

func fillInstanceBuffer(buffer uint32, data []float32) {
	gl.BindBuffer(gl.ARRAY_BUFFER, buffer)
	if len(data) > 0 {
		gl.BufferData(gl.ARRAY_BUFFER, len(data)*bytesPerFloat, gl.Ptr(data), gl.DYNAMIC_DRAW)
	}
}

// describeInstanceLayout binds attributes 3-6 (matrix columns) and 7 (color), one per instance.
func describeInstanceLayout() {
	stride := int32(floatsPerInstance * bytesPerFloat)
	for column := uint32(0); column < 5; column++ {
		attribute := 3 + column
		gl.EnableVertexAttribArray(attribute)
		gl.VertexAttribPointerWithOffset(attribute, 4, gl.FLOAT, false, stride, uintptr(column*4*bytesPerFloat))
		gl.VertexAttribDivisor(attribute, 1)
	}
}

// waterCommand returns a large water plane that follows the camera horizontally.
func (renderer *SceneRenderer) waterCommand(water *scene.Water, camera models.Camera) drawCommand {
	if renderer.waterPlane == nil {
		renderer.waterPlane = newWaterGrid(water.Size)
	}
	deepColor := water.Deep
	if deepColor == (graphics.Vec4{}) {
		deepColor = graphics.Vec4{0.05, 0.22, 0.28, 0.85}
	}
	renderer.waterPlane.Material.BaseColor = deepColor
	world := graphics.Translate(graphics.Vec4{camera.Position[0], water.Level, camera.Position[2], 0})
	node := &scene.Node{Name: "water", Mesh: renderer.waterPlane}
	return drawCommand{node: node, world: world, material: renderer.waterPlane.Material, squaredCameraDistance: -1, transparent: true, seaSurface: true}
}

// waterFloorCommand returns an opaque dark floor under the water, so that nothing behind the
// scene shows through the translucent surface where the geometry ends.
func (renderer *SceneRenderer) waterFloorCommand(water *scene.Water, camera models.Camera) drawCommand {
	if renderer.waterFloor == nil {
		renderer.waterFloor = newWaterPlane(water.Size)
		renderer.waterFloor.Material = &scene.Material{Name: "water floor", Kind: scene.KindUnlit, DoubleSided: true}
	}
	deep := water.Deep
	renderer.waterFloor.Material.BaseColor = graphics.Vec4{deep[0] * 0.6, deep[1] * 0.6, deep[2] * 0.6, 1}
	depth := water.Level - water.EffectiveFloorDepth()
	world := graphics.Translate(graphics.Vec4{camera.Position[0], depth, camera.Position[2], 0})
	node := &scene.Node{Name: "water floor", Mesh: renderer.waterFloor}
	return drawCommand{node: node, world: world, material: renderer.waterFloor.Material}
}

// waterGridDivisions is the number of cells per side of the water surface.
const waterGridDivisions = 256

// newWaterGrid returns the water surface as a grid whose cells are small near its centre (the
// camera) and large far away, so a varying surface (tides) meets the coast accurately nearby.
func newWaterGrid(size float32) *scene.Mesh {
	if size <= 0 {
		size = 1000
	}
	half := size / 2
	vertices := waterGridDivisions + 1
	mesh := &scene.Mesh{Name: "water",
		Material: &scene.Material{Name: "water", Kind: scene.KindWater, Transparent: true, DoubleSided: true}}
	for row := 0; row < vertices; row++ {
		for column := 0; column < vertices; column++ {
			mesh.Positions = append(mesh.Positions, half*denseNearCentre(column, vertices), 0, half*denseNearCentre(row, vertices))
			mesh.Normals = append(mesh.Normals, 0, 1, 0)
		}
	}
	for row := 0; row < waterGridDivisions; row++ {
		for column := 0; column < waterGridDivisions; column++ {
			corner := uint32(row*vertices + column)
			next := corner + uint32(vertices)
			mesh.Indices = append(mesh.Indices, corner, next, corner+1, corner+1, next, next+1)
		}
	}
	return mesh
}

// denseNearCentre maps a grid index to -1..1 with a quadratic spacing: fine in the middle.
func denseNearCentre(index, count int) float32 {
	linear := 2*float32(index)/float32(count-1) - 1
	if linear < 0 {
		return -linear * linear
	}
	return linear * linear
}

func newWaterPlane(size float32) *scene.Mesh {
	if size <= 0 {
		size = 1000
	}
	half := size / 2
	return &scene.Mesh{
		Name:      "water",
		Positions: []float32{-half, 0, -half, half, 0, -half, -half, 0, half, half, 0, half},
		Normals:   []float32{0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0},
		Indices:   []uint32{0, 2, 1, 1, 2, 3},
		Material:  &scene.Material{Name: "water", Kind: scene.KindWater, Transparent: true, DoubleSided: true},
	}
}
