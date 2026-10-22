package gles

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/akrck02/go-renderer/models"
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

	functions    Functions
	program      *program
	quads        *quadPass
	planner      *sceneplan.Planner
	attributesOn bool                   // the vertex attributes were switched on in the context
	frameLights  []sceneplan.FrameLight // the frame's point lights, which each draw takes a few of
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
		planner: sceneplan.NewPlanner(sceneplan.MergingStore{})}, nil
}

// Draw draws a frame of the scene seen by the camera, in the target framebuffer of width × height
// pixels.
func (renderer *SceneRenderer) Draw(world *scene.Scene, camera models.Camera, width, height int) {
	renderer.functions.BindFramebuffer(FramebufferTarget, renderer.Target)
	renderer.quads.target = renderer.Target
	frame := renderer.planner.Plan(world, camera, width, height, sceneplan.ShadowSettings{})
	renderer.clear(frame.Light, width, height)
	renderer.setFrameUniforms(world.Environment, frame.Light, camera, frame.Matrices)
	renderer.frameLights = frame.Lights
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
		functions.DepthMask(false) // translucent surfaces do not hide what is drawn after them
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
		functions.DepthMask(true)
		functions.Disable(Blend)
	}
}

// blendFor mixes what is drawn next over what is behind it by alpha, or adds its light.
func (renderer *SceneRenderer) blendFor(blending sceneplan.Blending) {
	if blending == sceneplan.BlendAdditive {
		renderer.functions.BlendFuncSeparate(SourceAlpha, One, One, OneMinusSrcAlpha)
		return
	}
	renderer.functions.BlendFuncSeparate(SourceAlpha, OneMinusSrcAlpha, One, OneMinusSrcAlpha)
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

// linearToScreen turns a linear colour into the screen's gamma, as the scene shader does.
func linearToScreen(color graphics.Vector) graphics.Vector {
	return graphics.Vector{gammaOf(color[0]), gammaOf(color[1]), gammaOf(color[2]), color[3]}
}

func gammaOf(channel float32) float32 {
	if channel <= 0 {
		return 0
	}
	return float32(math.Pow(float64(channel), 1/2.2))
}
