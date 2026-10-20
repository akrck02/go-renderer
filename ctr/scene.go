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
	// The first colour stage passes the vertex colour through, and the second passes it on (the
	// overlay changes both).
	C3D_TexEnv* environment = C3D_GetTexEnv(0);
	C3D_TexEnvInit(environment);
	C3D_TexEnvSrc(environment, C3D_Both, GPU_PRIMARY_COLOR, 0, 0);
	C3D_TexEnvFunc(environment, C3D_Both, GPU_REPLACE);
	C3D_TexEnvInit(C3D_GetTexEnv(1));
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

// drawBakedElements draws a piece of a baked mesh: interleaved position (3 floats) and colour (4
// bytes), and 16-bit indices.
static void drawBakedElements(void* vertices, void* indices, int count) {
	C3D_AttrInfo* attributeInfo = C3D_GetAttrInfo();
	AttrInfo_Init(attributeInfo);
	AttrInfo_AddLoader(attributeInfo, 0, GPU_FLOAT, 3);
	AttrInfo_AddLoader(attributeInfo, 1, GPU_UNSIGNED_BYTE, 4);
	C3D_BufInfo* bufferInfo = C3D_GetBufInfo();
	BufInfo_Init(bufferInfo);
	BufInfo_Add(bufferInfo, vertices, 3 * sizeof(float) + 4, 2, 0x10);
	C3D_DrawElements(GPU_TRIANGLES, count, C3D_UNSIGNED_SHORT, indices);
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

// waterGridDivisions is how many cells per side the water surface has here.
const waterGridDivisions = 32

//go:embed shaders/scene.shbin
var sceneShaderBinary []byte

//go:embed shaders/baked.shbin
var bakedShaderBinary []byte

// SceneRenderer draws a scene.Scene on the top screen in gles's simple form: lit meshes and
// instances, the light of the time of day, fog and translucent water. What a frame draws is
// planned by sceneplan, as on every other platform. The PICA200 cannot draw instances either, so
// each batch is merged into one mesh on the processor (sceneplan's MergedBatch), again only when
// it changes.
//
// Lighting every vertex on the GPU was too slow (30 frames a second for the sample island on a
// New 3DS XL), so meshes are drawn with their light worked out on the processor (bake.go) through
// baked.v.pica. Water, whose reflection depends on the camera, and moving nodes (scene.Node's
// Moving), whose light changes as they turn, are shaded on the GPU by scene.v.pica: baking a ship
// again each time it turned took most of a frame with a few dozen of them.
type SceneRenderer struct {
	screen      *Screen
	water       *Shader
	baked       *Shader
	waterPlaces waterLocations
	bakedPlaces bakedLocations
	planner     *sceneplan.Planner
	bakedMeshes map[*scene.Mesh]*bakedMesh
	shaded      map[*scene.Mesh]*shadedMesh
	forests     map[*scene.Instances]*mergedForest
	frames      int // frames drawn, to let go of meshes no longer drawn (forget)
	// vertexScratch and indexScratch are reused by every bake and upload: their contents only
	// matter until they are copied to linear memory.
	vertexScratch []float32
	indexScratch  []uint16

	// this frame's shading, and the shader its uniforms were last set for
	shading        frameShading
	light          lightKey
	bakes          int           // meshes baked since the renderer was made (Bakes)
	planTime       time.Duration // how long sceneplan took for the last frame (PlanTime)
	bound          *Shader
	frame          sceneplan.Frame
	viewProjection graphics.Mat4 // of the eye being drawn, in the PICA200's clip space
}

// waterLocations are where scene.v.pica reads each uniform.
type waterLocations struct {
	viewProjection, camera, sunDirection, sunColour, ambientGround, ambientSpan int
	waterLight, skyReflection, fogColour, settings, fog                         int
	model, normalMatrix, baseColour, weights, minimumAlpha                      int
}

// bakedLocations are where baked.v.pica reads each uniform.
type bakedLocations struct {
	viewProjection, camera, settings, fog, fogColour, model int
}

// NewSceneRenderer loads the scene shaders for drawing on the screen.
func NewSceneRenderer(screen *Screen) (*SceneRenderer, error) {
	water, err := LoadShader(sceneShaderBinary)
	if err != nil {
		return nil, err
	}
	baked, err := LoadShader(bakedShaderBinary)
	if err != nil {
		water.Free()
		return nil, err
	}
	planner := sceneplan.NewPlanner(sceneplan.MergingStore{})
	// 256 × 256 cells of water were 131 072 of the sample island's 160 756 triangles, through the
	// slow full shader; without waves 32 × 32 is enough, fine near the camera for the fog.
	planner.WaterGridDivisions = waterGridDivisions
	renderer := &SceneRenderer{screen: screen, water: water, baked: baked,
		planner: planner, bakedMeshes: map[*scene.Mesh]*bakedMesh{}, shaded: map[*scene.Mesh]*shadedMesh{},
		forests: map[*scene.Instances]*mergedForest{}}
	renderer.waterPlaces = waterLocations{
		viewProjection: water.Uniform("viewProjection"), camera: water.Uniform("camera"),
		sunDirection: water.Uniform("sunDirection"), sunColour: water.Uniform("sunColour"),
		ambientGround: water.Uniform("ambientGround"), ambientSpan: water.Uniform("ambientSpan"),
		waterLight: water.Uniform("waterLight"), skyReflection: water.Uniform("skyReflection"),
		fogColour: water.Uniform("fogColour"), settings: water.Uniform("settings"), fog: water.Uniform("fog"),
		model: water.Uniform("model"), normalMatrix: water.Uniform("normalMatrix"),
		baseColour: water.Uniform("baseColour"), weights: water.Uniform("weights"),
		minimumAlpha: water.Uniform("minimumAlpha"),
	}
	renderer.bakedPlaces = bakedLocations{
		viewProjection: baked.Uniform("viewProjection"), camera: baked.Uniform("camera"),
		settings: baked.Uniform("settings"), fog: baked.Uniform("fog"),
		fogColour: baked.Uniform("fogColour"), model: baked.Uniform("model"),
	}
	return renderer, nil
}

// PlanTime is how long planning the last frame took on the processor, before citro3d's own
// processing time (ProcessingTime) starts.
func (renderer *SceneRenderer) PlanTime() time.Duration {
	return renderer.planTime
}

// Bakes is how many times a mesh's light has been worked out on the processor since the renderer
// was made: when meshes are new, change, or the light changes.
func (renderer *SceneRenderer) Bakes() int {
	return renderer.bakes
}

// Statistics are what the last frame drew; processorTime is how long the program took for it.
func (renderer *SceneRenderer) Statistics(processorTime time.Duration) sceneplan.FrameStatistics {
	return renderer.planner.Statistics(processorTime)
}

// Draw plans and draws a frame of the scene seen by the camera, between BeginFrame and EndFrame
// of the screen: Draw calls them itself.
func (renderer *SceneRenderer) Draw(world *scene.Scene, camera models.Camera) {
	renderer.DrawStereo(world, camera, 0, 0)
}

// DrawStereo draws a frame for both eyes, which stand separation apart and see what lies at the
// focal distance on the plane of the screen. The frame is planned once, for the camera between
// them. With no separation it draws one picture, as Draw. It begins and ends the frame itself; to
// draw the bottom screen in the same frame, use PaintStereo between BeginFrame and EndFrame.
func (renderer *SceneRenderer) DrawStereo(world *scene.Scene, camera models.Camera, separation, focalDistance float32) {
	BeginFrame()
	renderer.PaintStereo(world, camera, separation, focalDistance)
	EndFrame()
}

// PaintStereo is DrawStereo inside a frame begun with BeginFrame.
func (renderer *SceneRenderer) PaintStereo(world *scene.Scene, camera models.Camera, separation, focalDistance float32) {
	renderer.nextFrame()
	planStart := time.Now()
	frame := renderer.planner.Plan(world, camera, ScreenWidth, ScreenHeight, sceneplan.ShadowSettings{})
	renderer.planTime = time.Since(planStart)
	renderer.frame = frame
	renderer.shading = shadingFor(world.Environment, frame.Light, camera)
	renderer.light = lightKeyOf(renderer.shading)
	clear := screenColour(frame.Light.Horizon)
	stereo := false
	if separation > 0 && focalDistance > 0 {
		stereo = renderer.screen.ClearStereo(clear) // clears one picture if it cannot do two
	} else {
		renderer.screen.Clear(clear)
	}
	if !stereo {
		renderer.drawEye(PICAViewProjection(frame.Matrices.ViewProjection), frame)
		return
	}
	for _, eye := range []Eye{LeftEye, RightEye} {
		renderer.screen.DrawOn(eye)
		viewProjection := EyeViewProjection(camera, eye, separation, focalDistance, ScreenWidth, ScreenHeight)
		renderer.drawEye(PICAViewProjection(viewProjection), frame)
	}
}

// PaintOn draws a frame of the scene on either screen, in one picture, inside a frame begun with
// BeginFrame: on the bottom one, the scene can be touched.
func (renderer *SceneRenderer) PaintOn(surface Surface, world *scene.Scene, camera models.Camera) {
	renderer.nextFrame()
	width, height := surface.size()
	planStart := time.Now()
	frame := renderer.planner.Plan(world, camera, width, height, sceneplan.ShadowSettings{})
	renderer.planTime = time.Since(planStart)
	renderer.frame = frame
	renderer.shading = shadingFor(world.Environment, frame.Light, camera)
	renderer.light = lightKeyOf(renderer.shading)
	surface.clear(screenColour(frame.Light.Horizon))
	renderer.drawEye(PICAViewProjection(frame.Matrices.ViewProjection), frame)
}

// drawEye draws the planned frame with one view-projection.
func (renderer *SceneRenderer) drawEye(viewProjection graphics.Mat4, frame sceneplan.Frame) {
	renderer.viewProjection = viewProjection
	renderer.bound = nil
	C.prepareScene()
	renderer.drawCommands(frame.Opaque, frame.View)
	renderer.drawCommands(frame.Transparent, frame.View)
}

// bind makes a shader the one drawing, setting its per-frame uniforms the first time this frame.
func (renderer *SceneRenderer) bind(shader *Shader) {
	if renderer.bound == shader {
		return
	}
	shader.Use()
	renderer.bound = shader
	shading := renderer.shading
	viewProjection := renderer.viewProjection
	if shader == renderer.baked {
		places := renderer.bakedPlaces
		SetMatrix(places.viewProjection, viewProjection)
		setVector(places.camera, shading.camera)
		setVector(places.settings, shading.settings)
		setVector(places.fog, shading.fog)
		setVector(places.fogColour, inScreenGamma(shading.fogColour))
		return
	}
	places := renderer.waterPlaces
	SetMatrix(places.viewProjection, viewProjection)
	setVector(places.camera, shading.camera)
	setVector(places.sunDirection, shading.sunDirection)
	setVector(places.sunColour, shading.sunColour)
	setVector(places.ambientGround, shading.ambientGround)
	setVector(places.ambientSpan, shading.ambientSpan)
	setVector(places.waterLight, shading.waterLight)
	setVector(places.skyReflection, shading.skyReflection)
	setVector(places.fogColour, shading.fogColour)
	setVector(places.settings, shading.settings)
	setVector(places.fog, shading.fog)
}

func (renderer *SceneRenderer) drawCommands(commands []sceneplan.Command, view sceneplan.Viewpoint) {
	if len(commands) == 0 {
		return
	}
	translucent := 0
	if commands[0].Transparent {
		translucent = 1
	}
	C.setBlending(C.int(translucent))
	for _, command := range commands {
		renderer.drawCommand(command, view)
	}
}

func (renderer *SceneRenderer) drawCommand(command sceneplan.Command, view sceneplan.Viewpoint) {
	draw := renderer.drawBaked
	if command.Material.Kind == scene.KindWater || command.Node.Moving {
		// Water's light depends on the camera, and a moving node's on how it turns: both are lit
		// on the GPU, from a copy of the mesh made once, instead of baked again and again.
		draw = renderer.drawShaded
		renderer.prepareShaded(command)
	} else {
		renderer.bind(renderer.baked)
		SetMatrix(renderer.bakedPlaces.model, command.World)
	}
	if command.Node.Instances != nil {
		forest := renderer.wholeForest(command.Node)
		draw(command, forest)
		renderer.planner.Count(sceneplan.MainPass, command.Node.Mesh, len(command.Node.Instances.Transforms))
		return
	}
	mesh, visible := renderer.planner.MeshToDraw(command, view)
	if !visible {
		return
	}
	draw(command, mesh)
	renderer.planner.Count(sceneplan.MainPass, mesh, 0)
}

// wholeForest is all of a node's instances merged into one mesh, made again only when the
// instances change. sceneplan's batches hold only the visible cells, at their level of detail, so
// they change as the camera turns, and merging and baking a forest of 30 000 vertices again took
// a quarter of a second on a New 3DS XL. The whole node at its finest level is a few thousand
// triangles more for the GPU, which has room for them.
func (renderer *SceneRenderer) wholeForest(node *scene.Node) *scene.Mesh {
	instances := node.Instances
	forest := renderer.forests[instances]
	if forest != nil {
		forest.lastDrawn = renderer.frames
	}
	if forest != nil && forest.source == node.Mesh && !instances.Dirty && len(instances.Changed) == 0 {
		return forest.merged
	}
	if forest == nil {
		forest = &mergedForest{merged: &scene.Mesh{Name: node.Mesh.Name + " (all instances)"}, lastDrawn: renderer.frames}
		renderer.forests[instances] = forest
	}
	forest.source = node.Mesh
	sceneplan.MergeInstances(forest.merged, node.Mesh, sceneplan.PackInstances(instances), len(instances.Transforms))
	forest.merged.Dirty = true // bake and upload it again
	instances.Dirty, instances.Changed = false, instances.Changed[:0]
	return forest.merged
}

// mergedForest is a node's instances as one mesh.
type mergedForest struct {
	source    *scene.Mesh // the instanced mesh it was made from
	merged    *scene.Mesh
	lastDrawn int
}

// prepareShaded binds the full shader and sets what one command it draws needs: water, or a
// moving node's lit, flat or glowing mesh.
func (renderer *SceneRenderer) prepareShaded(command sceneplan.Command) {
	renderer.bind(renderer.water)
	places := renderer.waterPlaces
	setVector(places.baseColour, sceneplan.BaseColorOf(command.Material))
	setVector(places.weights, materialWeights(command.Material.Kind, renderer.light.brightness))
	setVector(places.minimumAlpha, graphics.Vec4{minimumAlpha(command.Material.Kind), 0, 0, 0})
	SetMatrix(places.model, command.World)
	normal := sceneplan.NormalMatrix(command.World) // column by column
	for row := range 3 {
		setVector(places.normalMatrix+row, graphics.Vec4{normal[row], normal[3+row], normal[6+row], 0})
	}
}

func (renderer *SceneRenderer) drawShaded(command sceneplan.Command, mesh *scene.Mesh) {
	for _, piece := range renderer.uploadShaded(mesh).pieces {
		C.drawSceneElements(piece.vertices, piece.indices, C.int(piece.count))
	}
}

func (renderer *SceneRenderer) drawBaked(command sceneplan.Command, mesh *scene.Mesh) {
	key := bakeKey{
		normalMatrix: sceneplan.NormalMatrix(command.World),
		baseColour:   sceneplan.BaseColorOf(command.Material),
		kind:         command.Material.Kind,
		light:        renderer.light,
		lamps:        lampKeyOf(command.Lights, renderer.frame.Lights, command.World),
	}
	for _, piece := range renderer.bakedFor(mesh, key).pieces {
		C.drawBakedElements(piece.vertices, piece.indices, C.int(piece.count))
	}
}

// forgetAfter is how many frames a mesh stays on the GPU without being drawn: a program that
// makes its scene again (a plan whose machines change colour) leaves its old meshes behind, and
// their memory, in the Go heap and in linear memory, would never come back.
const forgetAfter = 60

// nextFrame counts a frame, and every forgetAfter frames lets go of the meshes not drawn since.
func (renderer *SceneRenderer) nextFrame() {
	renderer.frames++
	if renderer.frames%forgetAfter != 0 {
		return
	}
	stale := renderer.frames - forgetAfter
	for mesh, baked := range renderer.bakedMeshes {
		if baked.lastDrawn < stale {
			for _, variant := range baked.variants {
				variant.onGPU.free()
			}
			delete(renderer.bakedMeshes, mesh)
		}
	}
	for mesh, shaded := range renderer.shaded {
		if shaded.lastDrawn < stale {
			shaded.onGPU.free()
			delete(renderer.shaded, mesh)
		}
	}
	for instances, forest := range renderer.forests {
		if forest.lastDrawn < stale {
			delete(renderer.forests, instances) // its merged mesh goes with bakedMeshes
		}
	}
}

// bakedMesh is a mesh's baked versions on the GPU: usually one, more when the mesh is drawn by
// nodes turned differently or with different materials.
type bakedMesh struct {
	variants  []bakedVariant
	lastDrawn int // the frame it was last drawn in
}

// bakedFor returns the mesh baked for key, baking it when it is new, Dirty, or lit differently.
func (renderer *SceneRenderer) bakedFor(mesh *scene.Mesh, key bakeKey) *meshOnGPU {
	baked := renderer.bakedMeshes[mesh]
	if baked == nil {
		baked = &bakedMesh{}
		renderer.bakedMeshes[mesh] = baked
	}
	baked.lastDrawn = renderer.frames
	if mesh.Dirty {
		for _, variant := range baked.variants {
			variant.onGPU.free()
		}
		baked.variants = baked.variants[:0]
		mesh.Dirty = false
	}
	for index := range baked.variants {
		variant := &baked.variants[index]
		if variant.key.normalMatrix != key.normalMatrix || variant.key.baseColour != key.baseColour || variant.key.kind != key.kind ||
			variant.key.lamps.model != key.lamps.model {
			continue // a mesh drawn in two places under lamps is lit differently in each
		}
		if !variant.key.light.closeTo(key.light) || !variant.key.lamps.closeTo(key.lamps) {
			variant.onGPU.free()
			variant.onGPU = renderer.uploadBaked(mesh, key)
			variant.key = key
		}
		variant.lastUsed = renderer.frames
		return variant.onGPU
	}
	onGPU := renderer.uploadBaked(mesh, key)
	fresh := bakedVariant{key: key, onGPU: onGPU, lastUsed: renderer.frames}
	place := placeFor(baked.variants, key.lamps.count > 0)
	if place < 0 {
		baked.variants = append(baked.variants, fresh)
		return onGPU
	}
	baked.variants[place].onGPU.free()
	baked.variants[place] = fresh
	return onGPU
}

// uploadBaked bakes a mesh's light and copies it to linear memory, through the renderer's own
// buffers: a mesh baked again (its colours changed) makes no new garbage.
func (renderer *SceneRenderer) uploadBaked(mesh *scene.Mesh, key bakeKey) *meshOnGPU {
	renderer.bakes++
	renderer.vertexScratch = bakeVertices(renderer.vertexScratch, mesh, key)
	return uploadPieces(renderer.vertexScratch, mesh.Indices, floatsPerBakedVertex, &renderer.indexScratch)
}

// shadedMesh is a mesh with its normals on the GPU, for the full shader.
type shadedMesh struct {
	onGPU     *meshOnGPU
	lastDrawn int
}

// uploadShaded copies a mesh, with its normals, to linear memory for the full shader the first
// time it is drawn, and again when it is Dirty.
func (renderer *SceneRenderer) uploadShaded(mesh *scene.Mesh) *meshOnGPU {
	shaded := renderer.shaded[mesh]
	if shaded != nil && !mesh.Dirty {
		shaded.lastDrawn = renderer.frames
		return shaded.onGPU
	}
	if shaded != nil {
		shaded.onGPU.free()
	}
	shaded = &shadedMesh{onGPU: uploadPieces(sceneplan.InterleaveVertices(mesh), mesh.Indices, sceneplan.FloatsPerVertex, &renderer.indexScratch), lastDrawn: renderer.frames}
	renderer.shaded[mesh] = shaded
	mesh.Dirty = false
	return shaded.onGPU
}

// uploadPieces copies interleaved vertices and their triangles to linear memory, cut into pieces
// of 16-bit indices.
func uploadPieces(vertices []float32, indices []uint32, floatsPerVertex int, reusedIndices *[]uint16) *meshOnGPU {
	onGPU := &meshOnGPU{}
	if len(vertices) == 0 {
		return onGPU
	}
	for _, piece := range splitForShortIndices(vertices, indices, floatsPerVertex, reusedIndices) {
		pieceVertices := copyToLinearMemory(unsafe.Pointer(&piece.vertices[0]), len(piece.vertices)*4)
		pieceIndices := copyToLinearMemory(unsafe.Pointer(&piece.indices[0]), len(piece.indices)*2)
		if pieceVertices == nil || pieceIndices == nil {
			freeLinearMemory(pieceVertices)
			freeLinearMemory(pieceIndices)
			break // out of linear memory: draw what fits
		}
		onGPU.pieces = append(onGPU.pieces, pieceOnGPU{vertices: pieceVertices, indices: pieceIndices, count: len(piece.indices)})
	}
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
