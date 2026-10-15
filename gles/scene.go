package gles

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/sceneplan"
)

// SceneRenderer draws a scene.Scene in its simplest form: lit meshes and instances, the light of
// the time of day, fog, translucent water and, over them, the overlay (labels, header, notices and
// the program's draw list). What a frame draws (culling, levels of detail, order) is planned by
// sceneplan, like on the desktop.
//
// It is a first step, not the desktop renderer: there are no shadows, sky dome, procedural
// patterns, waves, wind or snow. OpenGL ES 2 cannot draw instances, so each batch of instances is
// made into one mesh on the processor, again only when it changes.
type SceneRenderer struct {
	// OverlayScale is framebuffer pixels per layout pixel, so labels and interfaces keep their
	// size on a high-density screen (0 = 1).
	OverlayScale float32

	functions    Functions
	program      *program
	quads        *quadPass
	planner      *sceneplan.Planner
	attributesOn bool // the vertex attributes were switched on in the context
}

// NewSceneRenderer compiles the shaders and prepares the overlay's atlas.
func NewSceneRenderer(functions Functions) (*SceneRenderer, error) {
	shaders, err := newProgram(functions, sceneVertexShader, sceneFragmentShader, sceneAttributes)
	if err != nil {
		return nil, err
	}
	quads, err := newQuadPass(functions)
	if err != nil {
		return nil, err
	}
	return &SceneRenderer{functions: functions, program: shaders, quads: quads,
		planner: sceneplan.NewPlanner(instanceStore{})}, nil
}

// Draw draws a frame of the scene seen by the camera, in a framebuffer of width × height pixels.
func (renderer *SceneRenderer) Draw(world *scene.Scene, camera models.Camera, width, height int) {
	frame := renderer.planner.Plan(world, camera, width, height, sceneplan.ShadowSettings{})
	renderer.clear(frame.Light, width, height)
	renderer.setFrameUniforms(world.Environment, frame.Light, camera, frame.Matrices)
	renderer.drawCommands(frame.Opaque, frame.View)
	renderer.drawCommands(frame.Transparent, frame.View)
	scale := renderer.OverlayScale
	if scale <= 0 {
		scale = 1
	}
	renderer.quads.paintScene(world, frame.Matrices.ViewProjection, width, height, scale)
}

// clear fills the frame with the colour of the horizon, which stands in for the sky; the fog
// fades into it, so it takes the same gamma as the shaded surfaces.
func (renderer *SceneRenderer) clear(light scene.Daylight, width, height int) {
	functions := renderer.functions
	functions.Viewport(0, 0, width, height)
	horizon := linearToScreen(light.Horizon)
	functions.ClearColor(horizon[0], horizon[1], horizon[2], 1)
	functions.Clear(ColorBufferBit | DepthBufferBit)
}

func (renderer *SceneRenderer) setFrameUniforms(environment scene.Environment, light scene.Daylight, camera models.Camera, matrices sceneplan.CameraMatrices) {
	program := renderer.program
	program.use()
	program.setMatrix("viewProjection", matrices.ViewProjection)
	program.setFloat("verticalScale", environment.EffectiveVerticalScale())
	program.setVector3("cameraPosition", camera.Position)
	program.setVector3("sunDirection", light.LightDirection)
	program.setVector3("sunColor", light.LightColor)
	program.setVector3("skyColor", light.SkyLight)
	program.setVector3("groundColor", light.GroundLight)
	program.setFloat("ambient", environment.Ambient)
	program.setFloat("brightness", light.Brightness)
	program.setVector3("fogColor", light.Fog)
	program.setVector2("fogRange", environment.FogNear, environment.FogFar)
}

// drawCommands draws commands in order, blending the transparent ones over what is behind them.
func (renderer *SceneRenderer) drawCommands(commands []sceneplan.Command, view sceneplan.Viewpoint) {
	if len(commands) == 0 {
		return
	}
	functions := renderer.functions
	renderer.program.use()
	functions.Enable(DepthTest)
	functions.DepthFunc(LessOrEqual)
	transparent := commands[0].Transparent
	if transparent {
		functions.Enable(Blend)
		functions.BlendFuncSeparate(SourceAlpha, OneMinusSrcAlpha, One, OneMinusSrcAlpha)
		functions.DepthMask(false) // translucent surfaces do not hide what is drawn after them
	}
	for _, command := range commands {
		renderer.drawCommand(command, view)
	}
	if transparent {
		functions.DepthMask(true)
		functions.Disable(Blend)
	}
}

func (renderer *SceneRenderer) drawCommand(command sceneplan.Command, view sceneplan.Viewpoint) {
	program := renderer.program
	program.setVector4("baseColor", sceneplan.BaseColorOf(command.Material))
	program.setInteger("kind", int32(command.Material.Kind))
	if command.Node.Instances != nil {
		renderer.drawInstances(command, view)
		return
	}
	mesh, visible := renderer.planner.MeshToDraw(command, view)
	if !visible {
		return
	}
	renderer.setModel(command.World)
	program.setVector4("tint", graphics.Vec4{1, 1, 1, 1})
	renderer.drawMesh(mesh)
	renderer.planner.Count(sceneplan.MainPass, mesh, 0)
}

// drawInstances draws the visible instances of a node at the level of detail the planner chose
// for their cell: each batch in one call, as one mesh holding all its copies.
func (renderer *SceneRenderer) drawInstances(command sceneplan.Command, view sceneplan.Viewpoint) {
	for _, batch := range renderer.planner.InstanceBatches(command, view, sceneplan.MainPass) {
		instances, _ := batch.Handle.(*instanceBatch)
		if instances == nil || batch.Count == 0 {
			continue
		}
		renderer.setModel(command.World)
		renderer.program.setVector4("tint", graphics.Vec4{1, 1, 1, 1})
		renderer.drawMesh(instances.mergedFor(batch.Mesh, batch.Count))
		renderer.planner.Count(sceneplan.MainPass, batch.Mesh, batch.Count)
	}
}

// setModel sets the model matrix and the normal matrix that goes with it.
func (renderer *SceneRenderer) setModel(model graphics.Mat4) {
	renderer.program.setMatrix("model", model)
	renderer.program.setMatrix3("normalMatrix", normalMatrix(model))
}

// normalMatrix is the inverse transpose of a model matrix's rotation and scale: what turns normals
// so they stay perpendicular to surfaces scaled differently along each axis.
func normalMatrix(model graphics.Mat4) [9]float32 {
	inverse, invertible := model.Inverse()
	if !invertible {
		inverse = graphics.Identity()
	}
	// the transpose of the inverse's upper 3 × 3, column by column
	return [9]float32{
		inverse[0], inverse[4], inverse[8],
		inverse[1], inverse[5], inverse[9],
		inverse[2], inverse[6], inverse[10],
	}
}

func (renderer *SceneRenderer) drawMesh(mesh *scene.Mesh) {
	buffers := renderer.uploadMesh(mesh)
	functions := renderer.functions
	if !renderer.attributesOn {
		switchAttributesOn(functions)
		renderer.attributesOn = true
	}
	functions.BindBuffer(ArrayBuffer, buffers.vertexBuffer)
	describeSceneVertexLayout(functions)
	if buffers.indexed {
		functions.BindBuffer(ElementArrayBuffer, buffers.indexBuffer)
		functions.DrawElements(Triangles, buffers.count, UnsignedInt, 0)
		return
	}
	functions.DrawArrays(Triangles, 0, buffers.count)
}

// meshBuffers is a mesh on the GPU, in the graphics context of the renderer that uploaded it.
type meshBuffers struct {
	// owner is the renderer that made the buffers. A phone destroys the graphics context when the
	// application goes to the background, and the window makes a new renderer when it comes back:
	// buffers of another renderer name nothing in its context, and are made again.
	owner        *SceneRenderer
	vertexBuffer Buffer
	indexBuffer  Buffer
	count        int
	indexed      bool
}

// uploadMesh sends a mesh to the GPU the first time this renderer draws it, and again when it is
// Dirty.
func (renderer *SceneRenderer) uploadMesh(mesh *scene.Mesh) *meshBuffers {
	buffers, _ := mesh.GPU.(*meshBuffers)
	if buffers != nil && buffers.owner == renderer && !mesh.Dirty {
		return buffers
	}
	functions := renderer.functions
	if buffers == nil || buffers.owner != renderer {
		buffers = &meshBuffers{owner: renderer, vertexBuffer: functions.CreateBuffer(), indexBuffer: functions.CreateBuffer()}
	}
	functions.BindBuffer(ArrayBuffer, buffers.vertexBuffer)
	functions.BufferData(ArrayBuffer, floatBytes(sceneplan.InterleaveVertices(mesh)), StaticDraw)
	buffers.indexed = len(mesh.Indices) > 0
	buffers.count = sceneplan.ElementCount(mesh)
	if buffers.indexed {
		functions.BindBuffer(ElementArrayBuffer, buffers.indexBuffer)
		functions.BufferData(ElementArrayBuffer, indexBytes(mesh.Indices), StaticDraw)
	}
	mesh.GPU, mesh.Dirty = buffers, false
	return buffers
}

// describeSceneVertexLayout tells where position, normal and colour are in sceneplan's interleaved
// vertices, in the bound vertex buffer; with no vertex array objects, before every draw.
func describeSceneVertexLayout(functions Functions) {
	stride := sceneplan.FloatsPerVertex * 4
	functions.VertexAttribPointer(0, 3, Float, false, stride, 0)
	functions.VertexAttribPointer(1, 3, Float, false, stride, 3*4)
	functions.VertexAttribPointer(2, 4, Float, false, stride, 6*4)
}

// instanceBatch keeps a batch's packed instances, and the batch as one mesh: OpenGL ES 2 cannot
// draw instances, and a draw call per copy is far too slow (a forest is thousands of calls, each
// a trip through x/mobile's or the browser's bindings). The mesh is made again only when the
// batch changes.
type instanceBatch struct {
	packed []float32
	merged *scene.Mesh // every copy placed and tinted; nil or stale when the batch changed
	stale  bool
	// mesh and count are what merged was made from
	mesh  *scene.Mesh
	count int
}

// mergedFor returns the batch's first count instances of mesh as one mesh, making it when needed.
func (batch *instanceBatch) mergedFor(mesh *scene.Mesh, count int) *scene.Mesh {
	if batch.merged != nil && !batch.stale && batch.mesh == mesh && batch.count == count {
		return batch.merged
	}
	if batch.merged == nil {
		batch.merged = &scene.Mesh{Name: mesh.Name + " (instances)"}
	}
	mergeInstances(batch.merged, mesh, batch.packed, count)
	batch.merged.Dirty = true // upload it again, into the same buffers
	batch.stale, batch.mesh, batch.count = false, mesh, count
	return batch.merged
}

// mergeInstances fills into with count copies of mesh, each placed by its instance's matrix,
// its normals turned with it and its colours tinted by the instance's colour.
func mergeInstances(into, mesh *scene.Mesh, packed []float32, count int) {
	if len(mesh.Normals) != len(mesh.Positions) {
		mesh.ComputeNormals()
	}
	vertices := mesh.VertexCount()
	into.Positions, into.Normals = into.Positions[:0], into.Normals[:0]
	into.Colors, into.Indices = into.Colors[:0], into.Indices[:0]
	for instance := range count {
		values := packed[instance*sceneplan.FloatsPerInstance:]
		placement, tint := graphics.Mat4(values[0:16]), graphics.Vec4(values[16:20])
		turn := normalMatrix(placement)
		for vertex := range vertices {
			into.Positions = appendPoint(into.Positions, placement.Apply(mesh.Vertex(uint32(vertex))))
			into.Normals = appendNormal(into.Normals, turn, mesh.Normals[3*vertex:3*vertex+3])
			into.Colors = appendTinted(into.Colors, mesh.Colors, vertex, tint)
		}
		first := uint32(instance * vertices)
		for _, index := range mesh.Indices {
			into.Indices = append(into.Indices, first+index)
		}
	}
}

func appendPoint(values []float32, point graphics.Vec4) []float32 {
	return append(values, point[0], point[1], point[2])
}

// appendNormal turns a normal by a normal matrix (column by column) and keeps it of unit length.
func appendNormal(values []float32, turn [9]float32, normal []float32) []float32 {
	x := turn[0]*normal[0] + turn[3]*normal[1] + turn[6]*normal[2]
	y := turn[1]*normal[0] + turn[4]*normal[1] + turn[7]*normal[2]
	z := turn[2]*normal[0] + turn[5]*normal[1] + turn[8]*normal[2]
	length := float32(math.Sqrt(float64(x*x + y*y + z*z)))
	if length == 0 {
		return append(values, 0, 1, 0)
	}
	return append(values, x/length, y/length, z/length)
}

// appendTinted is a vertex's colour (white when the mesh has none) times the instance's tint.
func appendTinted(values, colors []float32, vertex int, tint graphics.Vec4) []float32 {
	color := [4]float32{1, 1, 1, 1}
	if len(colors) >= 4*(vertex+1) {
		color = [4]float32(colors[4*vertex : 4*vertex+4])
	}
	return append(values, color[0]*tint[0], color[1]*tint[1], color[2]*tint[2], color[3]*tint[3])
}

// instanceStore keeps the planner's instance batches for SceneRenderer.
type instanceStore struct{}

func (instanceStore) FillBatch(handle any, mesh *scene.Mesh, packed []float32) any {
	batch, _ := handle.(*instanceBatch)
	if batch == nil {
		batch = &instanceBatch{}
	}
	batch.packed = append(batch.packed[:0], packed...)
	batch.stale = true
	return batch
}

func (instanceStore) RewriteInstances(handle any, position int, packed []float32) {
	batch, _ := handle.(*instanceBatch)
	if batch == nil {
		return
	}
	copy(batch.packed[position*sceneplan.FloatsPerInstance:], packed)
	batch.stale = true
}

// linearToScreen turns a linear colour into the screen's gamma, as the scene shader does.
func linearToScreen(color graphics.Vec4) graphics.Vec4 {
	return graphics.Vec4{gammaOf(color[0]), gammaOf(color[1]), gammaOf(color[2]), color[3]}
}

func gammaOf(channel float32) float32 {
	if channel <= 0 {
		return 0
	}
	return float32(math.Pow(float64(channel), 1/2.2))
}
