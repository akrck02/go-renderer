package opengl

import (
	"time"
	"unsafe"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/sceneplan"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// SceneRenderer draws a scene.Scene. Meshes and instances are uploaded once and kept on the
// GPU; they are uploaded again only when marked Dirty (for simulations).
// It must be created and used on the thread that owns the OpenGL context.
type SceneRenderer struct {
	Shadows    ShadowSettings
	Profiling  bool            // measure CPU and GPU times into Statistics (waits for the GPU every frame)
	Statistics FrameStatistics // the last frame drawn
	// OverlayScale is how many framebuffer pixels make one pixel of the overlay (labels, header,
	// notices): 2 on a high-density screen, so they keep their size; 0 = 1.
	OverlayScale float32
	profiler     *passProfiler

	planner        *sceneplan.Planner
	sceneProgram   *shaderProgram
	sky            *skyRenderer
	overlay        *overlayRenderer
	shadow         *shadowMap
	levelVariation *levelVariationTexture
	seabed         *seabedMap
	frameLights    []sceneplan.FrameLight                     // the frame's point lights, which each draw takes a few of
	lightValues    [2][4 * sceneplan.MaxLightsPerDraw]float32 // the draw's lights as uniforms: positions, colours
}

// ShadowSettings controls the sun shadows (see sceneplan.ShadowSettings).
type ShadowSettings = sceneplan.ShadowSettings

// DefaultShadowSettings returns enabled shadows with automatic sizes.
func DefaultShadowSettings() ShadowSettings { return sceneplan.DefaultShadowSettings() }

// FrameStatistics describes the last frame drawn (see sceneplan.FrameStatistics).
type FrameStatistics = sceneplan.FrameStatistics

// PassTimes are GPU durations of the passes of one frame.
type PassTimes = sceneplan.PassTimes

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
		planner: sceneplan.NewPlanner(instanceBatches{})}
	if renderer.shadow, err = newShadowMap(renderer.Shadows.EffectiveResolution()); err != nil {
		return nil, err
	}
	if renderer.seabed, err = newSeabedMap(seabedResolution); err != nil {
		return nil, err
	}
	renderer.levelVariation = newLevelVariationTexture()
	if renderer.overlay, err = newOverlayRenderer(); err != nil {
		return nil, err
	}
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

// Draw renders the scene from the camera into the current framebuffer of the given size, then its
// overlay on top. seconds is the elapsed time, which animates water and waterfalls.
func (renderer *SceneRenderer) Draw(world *scene.Scene, camera models.Camera, width, height int, seconds float64) {
	started := time.Now()
	frame := renderer.planner.Plan(world, camera, width, height, renderer.Shadows)
	renderer.measurePass("seabed", func() { renderer.seabed.update(frame.Seabed) })
	var err error
	renderer.measurePass("shadows", func() { err = renderer.renderShadows(world, frame, seconds) })
	shadowsReady := err == nil && frame.Shadow.Enabled
	clearFrame(frame.Light, width, height)
	renderer.setFrameUniforms(world.Environment, frame.Light, camera, frame.Matrices, seconds)
	renderer.shadow.bindForLighting(renderer.sceneProgram, frame.Shadow.Region, shadowsReady)
	renderer.seabed.bindForWater(renderer.sceneProgram, world.Environment.Water)
	renderer.levelVariation.bind(renderer.sceneProgram, world.Environment.Water)
	renderer.frameLights = frame.Lights
	renderer.measurePass("scene", func() { renderer.executeDrawCommands(frame.Opaque, frame.View) })
	renderer.measurePass("sky", func() { renderer.sky.draw(world.Environment, frame.Light, camera, frame.Matrices, seconds) })
	renderer.measurePass("transparent", func() { renderer.executeDrawCommands(frame.Transparent, frame.View) })
	renderer.measurePass("overlay", func() { renderer.overlay.draw(world, frame.Matrices, width, height, renderer.OverlayScale) })
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

func (renderer *SceneRenderer) recordStatistics(processorTime time.Duration) {
	statistics := renderer.planner.Statistics(processorTime)
	if renderer.Profiling {
		statistics.GPU = renderer.profiler.collect()
	}
	renderer.Statistics = statistics
}

// renderShadows draws the shadow map for this frame when shadows are enabled.
func (renderer *SceneRenderer) renderShadows(world *scene.Scene, frame sceneplan.Frame, seconds float64) error {
	if !frame.Shadow.Enabled {
		return nil
	}
	if err := renderer.matchShadowResolution(); err != nil {
		return err
	}
	drawCaster := func(program *shaderProgram, command sceneplan.Command) {
		renderer.drawGeometry(program, command, frame.Shadow.View, sceneplan.ShadowPass)
	}
	renderer.shadow.renderShadowPass(frame.Commands, frame.Shadow.Region, world.Environment.EffectiveVerticalScale(), world.Environment.Wind, seconds, drawCaster)
	return nil
}

// matchShadowResolution recreates the shadow map when the requested resolution changes.
func (renderer *SceneRenderer) matchShadowResolution() error {
	wanted := renderer.Shadows.EffectiveResolution()
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

func clearFrame(light scene.Daylight, width, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
	horizon := light.Horizon
	gl.ClearColor(horizon[0], horizon[1], horizon[2], 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.Disable(gl.CULL_FACE)
}

// setFrameUniforms sets what every draw of the frame shares; the light is the sun or the moon.
func (renderer *SceneRenderer) setFrameUniforms(environment scene.Environment, light scene.Daylight, camera models.Camera, matrices sceneplan.CameraMatrices, seconds float64) {
	program := renderer.sceneProgram
	gl.UseProgram(program.handle)
	program.setMatrix("projection", matrices.Projection)
	program.setMatrix("view", matrices.View)
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
	program.setFloat("waveLength", sceneplan.WaveLengthFor(environment.Water))
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

func (renderer *SceneRenderer) executeDrawCommands(commands []sceneplan.Command, view sceneplan.Viewpoint) {
	gl.UseProgram(renderer.sceneProgram.handle)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
	blending := false
	for _, command := range commands {
		if command.Transparent && !blending {
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

func (renderer *SceneRenderer) drawNode(command sceneplan.Command, view sceneplan.Viewpoint) {
	program := renderer.sceneProgram
	material := command.Material
	program.setVector4("baseColor", sceneplan.BaseColorOf(material))
	program.setInteger("kind", int32(material.Kind))
	program.setInteger("pattern", int32(material.Pattern))
	program.setInteger("foliage", boolToInteger(material.Foliage))
	program.setFloat("patternScale", material.PatternScale)
	program.setInteger("levelVariationEnabled", boolToInteger(command.SeaSurface && renderer.levelVariation.active))
	program.setInteger("seaSurface", boolToInteger(command.SeaSurface))
	renderer.setDrawLights(command.Lights)
	renderer.drawGeometry(program, command, view, sceneplan.MainPass)
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
	gl.BindVertexArray(buffers.vertexArray)
	fillVertexBuffer(buffers.vertexBuffer, sceneplan.InterleaveVertices(mesh))
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

func fillVertexBuffer(buffer uint32, data []float32) {
	gl.BindBuffer(gl.ARRAY_BUFFER, buffer)
	if len(data) > 0 {
		gl.BufferData(gl.ARRAY_BUFFER, len(data)*bytesPerFloat, gl.Ptr(data), gl.STATIC_DRAW)
	}
}

func describeVertexLayout() {
	stride := int32(sceneplan.FloatsPerVertex * bytesPerFloat)
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
	fillInstanceBuffer(buffers.instanceBuffer, sceneplan.PackInstances(instances))
	describeInstanceLayout()
	gl.BindVertexArray(0)
	instances.GPU, instances.Dirty = buffers, false
	return buffers
}

func fillInstanceBuffer(buffer uint32, data []float32) {
	gl.BindBuffer(gl.ARRAY_BUFFER, buffer)
	if len(data) > 0 {
		gl.BufferData(gl.ARRAY_BUFFER, len(data)*bytesPerFloat, gl.Ptr(data), gl.DYNAMIC_DRAW)
	}
}

// describeInstanceLayout binds attributes 3-6 (matrix columns) and 7 (color), one per instance.
func describeInstanceLayout() {
	stride := int32(sceneplan.FloatsPerInstance * bytesPerFloat)
	for column := uint32(0); column < 5; column++ {
		attribute := 3 + column
		gl.EnableVertexAttribArray(attribute)
		gl.VertexAttribPointerWithOffset(attribute, 4, gl.FLOAT, false, stride, uintptr(column*4*bytesPerFloat))
		gl.VertexAttribDivisor(attribute, 1)
	}
}
