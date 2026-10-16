//go:build nintendo3ds

package ctr

/*
#cgo CFLAGS: -Wno-ignored-attributes
#include <3ds.h>
#include <citro3d.h>

static void setVector(int location, float x, float y, float z, float w) {
	C3D_FVUnifSet(GPU_VERTEX_SHADER, location, x, y, z, w);
}

// prepareScene sets what every scene frame shares: no face culling, since glTF meshes are not
// all wound the same way (gles culls none either). Depth tests keep the greater value: citro3d
// maps the near plane to 1 and the far one to 0, and clears to 0.
static void prepareScene(void) {
	C3D_CullFace(GPU_CULL_NONE);
}

// setBlending switches between opaque surfaces, which write depth, and translucent ones, blended
// over what is behind them without hiding what is drawn after them.
static void setBlending(int translucent) {
	if (translucent) {
		C3D_AlphaBlend(GPU_BLEND_ADD, GPU_BLEND_ADD, GPU_SRC_ALPHA, GPU_ONE_MINUS_SRC_ALPHA, GPU_ONE, GPU_ONE_MINUS_SRC_ALPHA);
		C3D_DepthTest(true, GPU_GEQUAL, GPU_WRITE_COLOR);
		return;
	}
	C3D_AlphaBlend(GPU_BLEND_ADD, GPU_BLEND_ADD, GPU_ONE, GPU_ZERO, GPU_ONE, GPU_ZERO);
	C3D_DepthTest(true, GPU_GEQUAL, GPU_WRITE_ALL);
}

// drawSceneElements draws a piece of a mesh: interleaved position (3), normal (3) and colour (4)
// floats, and 16-bit indices.
static void drawSceneElements(void* vertices, void* indices, int count) {
	C3D_AttrInfo* attributeInfo = C3D_GetAttrInfo();
	AttrInfo_Init(attributeInfo);
	AttrInfo_AddLoader(attributeInfo, 0, GPU_FLOAT, 3);
	AttrInfo_AddLoader(attributeInfo, 1, GPU_FLOAT, 3);
	AttrInfo_AddLoader(attributeInfo, 2, GPU_FLOAT, 4);
	C3D_BufInfo* bufferInfo = C3D_GetBufInfo();
	BufInfo_Init(bufferInfo);
	BufInfo_Add(bufferInfo, vertices, 10 * sizeof(float), 3, 0x210);
	C3D_DrawElements(GPU_TRIANGLES, count, C3D_UNSIGNED_SHORT, indices);
}
*/
import "C"

import (
	_ "embed"
	"time"
	"unsafe"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/sceneplan"
)

// The top screen's size, which frames are planned for.
const (
	ScreenWidth  = 400
	ScreenHeight = 240
)

//go:embed shaders/scene.shbin
var sceneShaderBinary []byte

// SceneRenderer draws a scene.Scene on the top screen in gles's simple form: lit meshes and
// instances, the light of the time of day, fog and translucent water, shaded per vertex. What a
// frame draws is planned by sceneplan, as on every other platform. The PICA200 cannot draw
// instances either, so each batch is merged into one mesh on the processor (sceneplan's
// MergedBatch), again only when it changes.
type SceneRenderer struct {
	screen    *Screen
	shader    *Shader
	locations sceneLocations
	planner   *sceneplan.Planner
}

// sceneLocations are where scene.v.pica reads each uniform.
type sceneLocations struct {
	viewProjection, camera, sunDirection, sunColour, ambientGround, ambientSpan int
	waterLight, skyReflection, fogColour, settings, fog                         int
	model, normalMatrix, baseColour, weights, minimumAlpha                      int
}

// NewSceneRenderer loads the scene shader for drawing on the screen.
func NewSceneRenderer(screen *Screen) (*SceneRenderer, error) {
	shader, err := LoadShader(sceneShaderBinary)
	if err != nil {
		return nil, err
	}
	renderer := &SceneRenderer{screen: screen, shader: shader, planner: sceneplan.NewPlanner(sceneplan.MergingStore{})}
	renderer.locations = sceneLocations{
		viewProjection: shader.Uniform("viewProjection"), camera: shader.Uniform("camera"),
		sunDirection: shader.Uniform("sunDirection"), sunColour: shader.Uniform("sunColour"),
		ambientGround: shader.Uniform("ambientGround"), ambientSpan: shader.Uniform("ambientSpan"),
		waterLight: shader.Uniform("waterLight"), skyReflection: shader.Uniform("skyReflection"),
		fogColour: shader.Uniform("fogColour"), settings: shader.Uniform("settings"), fog: shader.Uniform("fog"),
		model: shader.Uniform("model"), normalMatrix: shader.Uniform("normalMatrix"),
		baseColour: shader.Uniform("baseColour"), weights: shader.Uniform("weights"),
		minimumAlpha: shader.Uniform("minimumAlpha"),
	}
	return renderer, nil
}

// Statistics are what the last frame drew; processorTime is how long the program took for it.
func (renderer *SceneRenderer) Statistics(processorTime time.Duration) sceneplan.FrameStatistics {
	return renderer.planner.Statistics(processorTime)
}

// Draw plans and draws a frame of the scene seen by the camera, between BeginFrame and EndFrame
// of the screen: Draw calls them itself.
func (renderer *SceneRenderer) Draw(world *scene.Scene, camera models.Camera) {
	frame := renderer.planner.Plan(world, camera, ScreenWidth, ScreenHeight, sceneplan.ShadowSettings{})
	renderer.screen.BeginFrame(screenColour(frame.Light.Horizon))
	renderer.shader.Use()
	C.prepareScene()
	renderer.setFrameUniforms(world.Environment, frame, camera)
	renderer.drawCommands(frame.Opaque, frame.View, frame.Light.Brightness)
	renderer.drawCommands(frame.Transparent, frame.View, frame.Light.Brightness)
	renderer.screen.EndFrame()
}

func (renderer *SceneRenderer) setFrameUniforms(environment scene.Environment, frame sceneplan.Frame, camera models.Camera) {
	shading := shadingFor(environment, frame.Light, camera)
	locations := renderer.locations
	SetMatrix(locations.viewProjection, PICAViewProjection(frame.Matrices.ViewProjection))
	setVector(locations.camera, shading.camera)
	setVector(locations.sunDirection, shading.sunDirection)
	setVector(locations.sunColour, shading.sunColour)
	setVector(locations.ambientGround, shading.ambientGround)
	setVector(locations.ambientSpan, shading.ambientSpan)
	setVector(locations.waterLight, shading.waterLight)
	setVector(locations.skyReflection, shading.skyReflection)
	setVector(locations.fogColour, shading.fogColour)
	setVector(locations.settings, shading.settings)
	setVector(locations.fog, shading.fog)
}

func (renderer *SceneRenderer) drawCommands(commands []sceneplan.Command, view sceneplan.Viewpoint, brightness float32) {
	if len(commands) == 0 {
		return
	}
	translucent := 0
	if commands[0].Transparent {
		translucent = 1
	}
	C.setBlending(C.int(translucent))
	for _, command := range commands {
		renderer.drawCommand(command, view, brightness)
	}
}

func (renderer *SceneRenderer) drawCommand(command sceneplan.Command, view sceneplan.Viewpoint, brightness float32) {
	locations := renderer.locations
	setVector(locations.baseColour, sceneplan.BaseColorOf(command.Material))
	setVector(locations.weights, materialWeights(command.Material.Kind, brightness))
	setVector(locations.minimumAlpha, graphics.Vec4{minimumAlpha(command.Material.Kind), 0, 0, 0})
	renderer.setModel(command.World)
	if command.Node.Instances != nil {
		renderer.drawInstances(command, view)
		return
	}
	mesh, visible := renderer.planner.MeshToDraw(command, view)
	if !visible {
		return
	}
	drawMesh(mesh)
	renderer.planner.Count(sceneplan.MainPass, mesh, 0)
}

// drawInstances draws each batch of visible instances as one mesh holding all its copies.
func (renderer *SceneRenderer) drawInstances(command sceneplan.Command, view sceneplan.Viewpoint) {
	for _, batch := range renderer.planner.InstanceBatches(command, view, sceneplan.MainPass) {
		instances, _ := batch.Handle.(*sceneplan.MergedBatch)
		if instances == nil || batch.Count == 0 {
			continue
		}
		drawMesh(instances.MergedFor(batch.Mesh, batch.Count))
		renderer.planner.Count(sceneplan.MainPass, batch.Mesh, batch.Count)
	}
}

// setModel sets the model matrix and the rows of its normal matrix.
func (renderer *SceneRenderer) setModel(model graphics.Mat4) {
	SetMatrix(renderer.locations.model, model)
	normal := sceneplan.NormalMatrix(model) // column by column
	for row := range 3 {
		setVector(renderer.locations.normalMatrix+row, graphics.Vec4{normal[row], normal[3+row], normal[6+row], 0})
	}
}

// meshOnGPU is a mesh in linear memory, in pieces of 16-bit indices.
type meshOnGPU struct {
	pieces []pieceOnGPU
}

type pieceOnGPU struct {
	vertices, indices unsafe.Pointer
	count             int
}

func drawMesh(mesh *scene.Mesh) {
	for _, piece := range upload(mesh).pieces {
		C.drawSceneElements(piece.vertices, piece.indices, C.int(piece.count))
	}
}

// upload copies a mesh to linear memory the first time it is drawn, and again when it is Dirty.
func upload(mesh *scene.Mesh) *meshOnGPU {
	onGPU, _ := mesh.GPU.(*meshOnGPU)
	if onGPU != nil && !mesh.Dirty {
		return onGPU
	}
	if onGPU != nil {
		onGPU.free()
	}
	onGPU = &meshOnGPU{}
	for _, piece := range splitForShortIndices(sceneplan.InterleaveVertices(mesh), mesh.Indices, sceneplan.FloatsPerVertex) {
		vertices := copyToLinearMemory(unsafe.Pointer(&piece.vertices[0]), len(piece.vertices)*4)
		indices := copyToLinearMemory(unsafe.Pointer(&piece.indices[0]), len(piece.indices)*2)
		if vertices == nil || indices == nil {
			freeLinearMemory(vertices)
			freeLinearMemory(indices)
			break // out of linear memory: draw what fits
		}
		onGPU.pieces = append(onGPU.pieces, pieceOnGPU{vertices: vertices, indices: indices, count: len(piece.indices)})
	}
	mesh.GPU, mesh.Dirty = onGPU, false
	return onGPU
}

func (onGPU *meshOnGPU) free() {
	for _, piece := range onGPU.pieces {
		freeLinearMemory(piece.vertices)
		freeLinearMemory(piece.indices)
	}
	onGPU.pieces = nil
}

func setVector(location int, value graphics.Vec4) {
	if location < 0 {
		return
	}
	C.setVector(C.int(location), C.float(value[0]), C.float(value[1]), C.float(value[2]), C.float(value[3]))
}
