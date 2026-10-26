package gles

import (
	"math"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
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
	// Target is the framebuffer Draw draws into: 0, the screen, unless the caller made one to
	// draw a picture offscreen.
	Target Framebuffer
	// LinearOutput makes the shaders and the clear write linear colour instead of turning it to
	// the screen's gamma: for a target that encodes sRGB itself (a headset's swapchain), which
	// would otherwise apply the gamma twice and wash the picture out.
	LinearOutput bool

	functions    Functions
	program      *program
	quads        *quadPass
	planner      *sceneplan.Planner
	attributesOn bool                   // the vertex attributes were switched on in the context
	frameLights  []sceneplan.FrameLight // the frame's point lights, which each draw takes a few of
	plans        int                    // frames planned, so tests can tell DrawViews plans once
}

// NewSceneRenderer compiles the shaders and prepares the overlay's atlas; the overlay's text is
// drawn in fonts (overlay.Fonts{} for the default ones).
func NewSceneRenderer(functions Functions, fonts overlay.Fonts) (*SceneRenderer, error) {
	shaders, err := newProgram(functions, sceneVertexShader, sceneFragmentShader, sceneAttributes)
	if err != nil {
		return nil, err
	}
	quads, err := newQuadPass(functions, fonts)
	if err != nil {
		return nil, err
	}
	return &SceneRenderer{functions: functions, program: shaders, quads: quads,
		planner: sceneplan.NewPlanner(sceneplan.MergingStore{})}, nil
}

// Draw draws a frame of the scene seen by the camera, in the target framebuffer of width × height
// pixels.
func (renderer *SceneRenderer) Draw(world *scene.Scene, camera camera.Lens, width, height int) {
	renderer.functions.BindFramebuffer(FramebufferTarget, renderer.Target)
	renderer.quads.target = renderer.Target
	frame := renderer.plan(world, camera, width, height)
	renderer.functions.SetViewport(0, 0, width, height)
	renderer.clear(frame.Light)
	renderer.setFrameUniforms(world.Environment, frame.Light, camera.Position, frame.Matrices.ViewProjection)
	renderer.drawFrame(frame)
	scale := renderer.OverlayScale
	if scale <= 0 {
		scale = 1
	}
	renderer.quads.paintScene(world, frame.Matrices.ViewProjection, width, height, scale)
}

// View is one picture DrawViews draws: an eye of a headset.
type View struct {
	// Target is the framebuffer it is drawn into, with a depth buffer. Views may share one, each
	// in its own viewport.
	Target Framebuffer
	// X, Y, Width and Height are its viewport in the target, in pixels from the bottom-left.
	X, Y, Width, Height int
	// View and Projection are the eye's matrices (see camera.Rig and
	// graphics.PerspectiveFromFieldOfView).
	View, Projection graphics.Matrix
	// Eye is the eye in the world: lighting, fog and the water's reflection are worked out from it.
	Eye graphics.Vector
}

// planHeight is the height of the picture DrawViews plans for: the planner only reads the aspect
// from the size, so any height will do.
const planHeight = 1000

// DrawViews plans the frame once, from covering (see camera.CoveringLens) and its aspect (width
// over height), and draws it in every view with the view's own matrices and eye. There is no
// overlay (labels, header, notices, draw list): a headset has no screen to pin it to.
func (renderer *SceneRenderer) DrawViews(world *scene.Scene, covering camera.Lens, aspect float32, views []View) {
	if len(views) == 0 {
		return
	}
	if aspect <= 0 {
		aspect = 1
	}
	frame := renderer.plan(world, covering, int(math.Round(float64(aspect)*planHeight)), planHeight)
	for index, view := range views {
		renderer.functions.BindFramebuffer(FramebufferTarget, view.Target)
		renderer.functions.SetViewport(view.X, view.Y, view.Width, view.Height)
		if !targetDrawnBefore(views[:index], view.Target) {
			renderer.clear(frame.Light) // a clear fills the whole target, so only before its first view
		}
		renderer.setFrameUniforms(world.Environment, frame.Light, view.Eye, view.Projection.Multiply(view.View))
		renderer.drawFrame(frame)
	}
}

// targetDrawnBefore tells whether one of the views already drew in target.
func targetDrawnBefore(views []View, target Framebuffer) bool {
	for _, view := range views {
		if view.Target == target {
			return true
		}
	}
	return false
}

// plan plans a frame seen by the camera, for a picture of width × height pixels.
func (renderer *SceneRenderer) plan(world *scene.Scene, camera camera.Lens, width, height int) sceneplan.Frame {
	renderer.plans++
	frame := renderer.planner.Plan(world, camera, width, height, sceneplan.ShadowSettings{})
	renderer.frameLights = frame.Lights
	return frame
}

// drawFrame draws the planned surfaces, opaque first, with the frame's uniforms already set.
func (renderer *SceneRenderer) drawFrame(frame sceneplan.Frame) {
	renderer.drawCommands(frame.Opaque, frame.View)
	renderer.drawCommands(frame.Transparent, frame.View)
}

// clear fills the bound framebuffer with the colour of the horizon, which stands in for the sky;
// the fog fades into it, so it takes the same gamma as the shaded surfaces.
func (renderer *SceneRenderer) clear(light scene.Daylight) {
	functions := renderer.functions
	horizon := light.Horizon
	if !renderer.LinearOutput {
		horizon = linearToScreen(horizon)
	}
	functions.SetClearColor(horizon[0], horizon[1], horizon[2], 1)
	functions.Clear(ColorBufferBit | DepthBufferBit)
}

// outputExponent is the power the scene shader raises its final colour to: the screen's gamma,
// or 1 to leave it linear.
func (renderer *SceneRenderer) outputExponent() float32 {
	if renderer.LinearOutput {
		return 1
	}
	return 1 / screenGamma
}

func (renderer *SceneRenderer) setFrameUniforms(environment scene.Environment, light scene.Daylight, eye graphics.Vector, viewProjection graphics.Matrix) {
	program := renderer.program
	program.use()
	program.setMatrix("viewProjection", viewProjection)
	program.setFloat("verticalScale", environment.EffectiveVerticalScale())
	program.setFloat("outputExponent", renderer.outputExponent())
	program.setVector3("cameraPosition", eye)
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
	functions.SetDepthFunction(LessOrEqual)
	transparent := commands[0].Transparent
	if transparent {
		functions.Enable(Blend)
		functions.SetDepthWrite(false) // translucent surfaces do not hide what is drawn after them
	}
	blending := sceneplan.BlendNone
	for _, command := range commands {
		if wanted := sceneplan.BlendingOf(command); transparent && wanted != blending {
			renderer.blendFor(wanted)
			blending = wanted
		}
		renderer.drawCommand(command, view)
	}
	if transparent {
		functions.SetDepthWrite(true)
		functions.Disable(Blend)
	}
}

// blendFor mixes what is drawn next over what is behind it by alpha, or adds its light.
func (renderer *SceneRenderer) blendFor(blending sceneplan.Blending) {
	if blending == sceneplan.BlendAdditive {
		renderer.functions.SetBlendFactors(SourceAlpha, One, One, OneMinusSourceAlpha)
		return
	}
	renderer.functions.SetBlendFactors(SourceAlpha, OneMinusSourceAlpha, One, OneMinusSourceAlpha)
}

func (renderer *SceneRenderer) drawCommand(command sceneplan.Command, view sceneplan.Viewpoint) {
	program := renderer.program
	program.setVector4("baseColor", sceneplan.BaseColorOf(command.Material))
	program.setInteger("kind", int32(command.Material.Kind))
	renderer.setDrawLights(command.Lights)
	if command.Node.Instances != nil {
		renderer.drawInstances(command, view)
		return
	}
	mesh, visible := renderer.planner.MeshToDraw(command, view)
	if !visible {
		return
	}
	renderer.setModel(command.World)
	program.setVector4("tint", graphics.Vector{1, 1, 1, 1})
	renderer.drawMesh(mesh)
	renderer.planner.Count(sceneplan.MainPass, mesh, 0)
}

// drawInstances draws the visible instances of a node at the level of detail the planner chose
// for their cell: each batch in one call, as one mesh holding all its copies.
func (renderer *SceneRenderer) drawInstances(command sceneplan.Command, view sceneplan.Viewpoint) {
	for _, batch := range renderer.planner.InstanceBatches(command, view, sceneplan.MainPass) {
		instances, _ := batch.Handle.(*sceneplan.MergedBatch)
		if instances == nil || batch.Count == 0 {
			continue
		}
		renderer.setModel(command.World)
		renderer.program.setVector4("tint", graphics.Vector{1, 1, 1, 1})
		renderer.drawMesh(instances.MergedFor(batch.Mesh, batch.Count))
		renderer.planner.Count(sceneplan.MainPass, batch.Mesh, batch.Count)
	}
}

// setModel sets the model matrix and the normal matrix that goes with it.
func (renderer *SceneRenderer) setModel(model graphics.Matrix) {
	renderer.program.setMatrix("model", model)
	renderer.program.setMatrix3("normalMatrix", sceneplan.NormalMatrix(model))
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
	functions.SetBufferData(ArrayBuffer, floatBytes(sceneplan.InterleaveVertices(mesh)), StaticDraw)
	buffers.indexed = len(mesh.Indices) > 0
	buffers.count = sceneplan.ElementCount(mesh)
	if buffers.indexed {
		functions.BindBuffer(ElementArrayBuffer, buffers.indexBuffer)
		functions.SetBufferData(ElementArrayBuffer, indexBytes(mesh.Indices), StaticDraw)
	}
	mesh.GPU, mesh.Dirty = buffers, false
	return buffers
}

// describeSceneVertexLayout tells where position, normal and colour are in sceneplan's interleaved
// vertices, in the bound vertex buffer; with no vertex array objects, before every draw.
func describeSceneVertexLayout(functions Functions) {
	stride := sceneplan.FloatsPerVertex * 4
	functions.DescribeVertexAttribute(0, 3, Float, false, stride, 0)
	functions.DescribeVertexAttribute(1, 3, Float, false, stride, 3*4)
	functions.DescribeVertexAttribute(2, 4, Float, false, stride, 6*4)
}

// screenGamma is the gamma of a screen that does not encode sRGB itself.
const screenGamma = 2.2

// linearToScreen turns a linear colour into the screen's gamma, as the scene shader does.
func linearToScreen(color graphics.Vector) graphics.Vector {
	return graphics.Vector{gammaOf(color[0]), gammaOf(color[1]), gammaOf(color[2]), color[3]}
}

func gammaOf(channel float32) float32 {
	if channel <= 0 {
		return 0
	}
	return float32(math.Pow(float64(channel), 1/screenGamma))
}
