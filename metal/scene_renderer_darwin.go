package metal

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework Metal -framework QuartzCore
#include <stdlib.h>
#include "scene_darwin.h"
*/
import "C"

import (
	"image"
	"image/draw"
	"time"
	"unsafe"

	"github.com/akrck02/go-renderer/failure"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// SceneRenderer draws a scene.Scene through Metal: the same frame as the OpenGL scene renderer
// (sceneplan decides what to draw; the shaders are a faithful translation), with its sky, water,
// shadows, levels of detail, instancing and overlay. Meshes and instances are uploaded once and
// kept on the GPU, again only when marked Dirty. Create and use it on the main thread.
type SceneRenderer struct {
	Shadows    sceneplan.ShadowSettings
	Profiling  bool                      // measure the GPU time of each frame (waits for the GPU every frame)
	Statistics sceneplan.FrameStatistics // the last frame drawn
	// OverlayScale is how many framebuffer pixels make one pixel of the overlay (labels, header,
	// notices): 2 on a high-density screen, so they keep their size; 0 = 1.
	OverlayScale float32

	device    unsafe.Pointer
	planner   *sceneplan.Planner
	pipelines struct {
		scene, sceneBlended, sceneAdditive, depthOnly, sky, constellationLines, constellationPoints, overlay unsafe.Pointer
	}
	depthStates struct {
		drawing, blended, shadow, overlay unsafe.Pointer
	}
	shadowMap        unsafe.Pointer
	shadowResolution int32
	seabedMap        unsafe.Pointer
	placeholders     struct{ panorama, cube, levelVariation, instances unsafe.Pointer }
	constellations   *constellationBuffer
	overlay          overlayLayer
	// capturing draws frames into an off-screen picture (captured) instead of the window.
	capturing bool
	captured  *image.NRGBA
}

// seabedResolution is the texels per side of the seabed map, as in the OpenGL renderer.
const seabedResolution = 2048

// NewSceneRenderer puts a Metal layer in a window made with no OpenGL context and prepares the
// shaders, the shadow and seabed maps.
func NewSceneRenderer(window *glfw.Window, vsync bool) (*SceneRenderer, error) {
	var problem *C.char
	device := C.sceneDeviceCreate(window.GetCocoaWindow(), C.int(boolean(vsync)), &problem)
	if device == nil {
		return nil, takeFailure(problem)
	}
	painter, err := overlay.NewPainter()
	if err != nil {
		return nil, err
	}
	renderer := &SceneRenderer{device: device, Shadows: sceneplan.DefaultShadowSettings(), overlay: overlayLayer{painter: painter}}
	renderer.planner = sceneplan.NewPlanner(metalBatches{renderer})
	if err := renderer.preparePipelines(); err != nil {
		renderer.Close()
		return nil, err
	}
	renderer.prepareTextures()
	return renderer, nil
}

// takeFailure turns a message from the Objective-C side into an error, freeing it.
func takeFailure(problem *C.char) error {
	defer C.free(unsafe.Pointer(problem))
	return failure.Wrap(Unavailable, C.GoString(problem), nil)
}

func (renderer *SceneRenderer) preparePipelines() error {
	source := C.CString(sceneShaderSource)
	defer C.free(unsafe.Pointer(source))
	var problem *C.char
	library := C.sceneLibraryCreate(renderer.device, source, &problem)
	if library == nil {
		return takeFailure(problem)
	}
	defer C.sceneRelease(library)
	pipelines := &renderer.pipelines
	for _, wanted := range []struct {
		into             *unsafe.Pointer
		vertex, fragment string
		blending         C.int
	}{
		{&pipelines.scene, "sceneVertex", "sceneFragment", C.sceneBlendNone},
		{&pipelines.sceneBlended, "sceneVertex", "sceneFragment", C.sceneBlendAlpha},
		{&pipelines.sceneAdditive, "sceneVertex", "sceneFragment", C.sceneBlendAdditive},
		{&pipelines.depthOnly, "sceneVertex", "", C.sceneBlendNone},
		{&pipelines.sky, "skyVertex", "skyFragment", C.sceneBlendNone},
		{&pipelines.constellationLines, "constellationVertex", "constellationLineFragment", C.sceneBlendAlpha},
		{&pipelines.constellationPoints, "constellationVertex", "constellationPointFragment", C.sceneBlendAlpha},
		{&pipelines.overlay, "overlayVertex", "overlayFragment", C.sceneBlendOverlay},
	} {
		pipeline, err := renderer.makePipeline(library, wanted.vertex, wanted.fragment, wanted.blending)
		if err != nil {
			return err
		}
		*wanted.into = pipeline
	}
	states := &renderer.depthStates
	states.drawing = C.sceneDepthStateCreate(renderer.device, C.sceneCompareLessEqual, 1)
	states.blended = C.sceneDepthStateCreate(renderer.device, C.sceneCompareLessEqual, 0)
	states.shadow = C.sceneDepthStateCreate(renderer.device, C.sceneCompareLess, 1)
	states.overlay = C.sceneDepthStateCreate(renderer.device, C.sceneCompareAlways, 0)
	return nil
}

// makePipeline makes a pipeline; an empty fragment name makes one that draws only depth.
func (renderer *SceneRenderer) makePipeline(library unsafe.Pointer, vertex, fragment string, blending C.int) (unsafe.Pointer, error) {
	vertexName := C.CString(vertex)
	defer C.free(unsafe.Pointer(vertexName))
	var fragmentName *C.char
	if fragment != "" {
		fragmentName = C.CString(fragment)
		defer C.free(unsafe.Pointer(fragmentName))
	}
	var problem *C.char
	pipeline := C.scenePipelineCreate(renderer.device, library, vertexName, fragmentName, blending, &problem)
	if pipeline == nil {
		return nil, takeFailure(problem)
	}
	return pipeline, nil
}

// prepareTextures makes the seabed map and the placeholders bound where a shader declares a
// texture the frame has none for: Metal needs every declared texture bound.
func (renderer *SceneRenderer) prepareTextures() {
	renderer.seabedMap = C.sceneTextureCreate(renderer.device, C.sceneFormatDepth32, seabedResolution, seabedResolution, 0, 0, 1)
	black := []byte{0, 0, 0, 255}
	placeholders := &renderer.placeholders
	placeholders.panorama = renderer.newTexture(C.sceneFormatRGBA8sRGB, 1, 1, false, false)
	C.sceneTextureWrite(placeholders.panorama, 0, 1, 1, unsafe.Pointer(&black[0]), 4)
	placeholders.cube = renderer.newTexture(C.sceneFormatRGBA8sRGB, 1, 1, true, false)
	for face := 0; face < 6; face++ {
		C.sceneTextureWrite(placeholders.cube, C.int(face), 1, 1, unsafe.Pointer(&black[0]), 4)
	}
	flat := []float32{0, 0}
	placeholders.levelVariation = renderer.newTexture(C.sceneFormatRG32Float, 1, 1, false, false)
	C.sceneTextureWrite(placeholders.levelVariation, 0, 1, 1, unsafe.Pointer(&flat[0]), 8)
	placeholders.instances = renderer.newBuffer(nil)
}

func (renderer *SceneRenderer) newTexture(format C.int, width, height int, cube, mipmapped bool) unsafe.Pointer {
	return C.sceneTextureCreate(renderer.device, format, C.int(width), C.int(height), C.int(boolean(cube)), C.int(boolean(mipmapped)), 0)
}

// newBuffer makes a GPU buffer holding the values (an empty one when there are none).
func (renderer *SceneRenderer) newBuffer(values []float32) unsafe.Pointer {
	if len(values) == 0 {
		return C.sceneBufferCreate(renderer.device, nil, 0)
	}
	return C.sceneBufferCreate(renderer.device, unsafe.Pointer(&values[0]), C.long(len(values)*4))
}

// Close lets go of the renderer's GPU objects. Meshes keep their buffers (in Mesh.GPU) until
// they are collected.
func (renderer *SceneRenderer) Close() {
	for _, object := range []unsafe.Pointer{
		renderer.pipelines.scene, renderer.pipelines.sceneBlended, renderer.pipelines.sceneAdditive, renderer.pipelines.depthOnly, renderer.pipelines.sky,
		renderer.pipelines.constellationLines, renderer.pipelines.constellationPoints, renderer.pipelines.overlay,
		renderer.depthStates.drawing, renderer.depthStates.blended, renderer.depthStates.shadow, renderer.depthStates.overlay,
		renderer.shadowMap, renderer.seabedMap, renderer.placeholders.panorama, renderer.placeholders.cube,
		renderer.placeholders.levelVariation, renderer.placeholders.instances, renderer.overlay.atlas, renderer.overlay.buffer,
		renderer.device,
	} {
		C.sceneRelease(object)
	}
	renderer.device = nil
}

// Draw renders the scene from the camera into the window (width × height framebuffer pixels),
// then its overlay on top, and shows the frame. seconds is the elapsed time, which animates water
// and waterfalls.
func (renderer *SceneRenderer) Draw(world *scene.Scene, camera models.Camera, width, height int, seconds float64) {
	if width <= 0 || height <= 0 {
		return
	}
	frame := C.sceneFrameBegin(renderer.device, C.int(width), C.int(height), C.int(boolean(renderer.capturing)))
	if frame == nil {
		return // the window has nothing to draw into (hidden, or being closed)
	}
	started := time.Now()
	plan := renderer.planner.Plan(world, camera, width, height, renderer.Shadows)
	renderer.drawSeabed(frame, plan.Seabed)
	shadowsReady := renderer.drawShadows(frame, world.Environment, plan, seconds)
	horizon := plan.Light.Horizon
	clear := [4]C.float{C.float(horizon[0]), C.float(horizon[1]), C.float(horizon[2]), 1}
	encoder := C.scenePassBeginMain(frame, &clear[0])
	uniforms := frameUniformsFor(world.Environment, plan, camera, seconds, shadowsReady)
	levelVariation, levelVariationActive := renderer.levelVariationTexture(world.Environment.Water)
	scenePass := sceneResources{uniforms: &uniforms, levelVariation: levelVariation, levelVariationActive: levelVariationActive, lights: plan.Lights}
	renderer.drawCommands(encoder, plan.Opaque, plan.View, scenePass, false)
	renderer.drawSky(encoder, world.Environment, plan, camera, seconds)
	renderer.drawCommands(encoder, plan.Transparent, plan.View, scenePass, true)
	renderer.drawOverlay(encoder, world, plan.Matrices, width, height)
	C.sceneEncoderEnd(encoder)
	renderer.finish(frame, width, height, started)
}

// finish sends the frame, reading it back when capturing and timing it when profiling.
func (renderer *SceneRenderer) finish(frame unsafe.Pointer, width, height int, started time.Time) {
	processorTime := time.Since(started)
	var pixels *C.uint8_t
	var picture *image.NRGBA
	if renderer.capturing {
		picture = image.NewNRGBA(image.Rect(0, 0, width, height))
		pixels = (*C.uint8_t)(unsafe.Pointer(&picture.Pix[0]))
	}
	var gpuSeconds C.double
	var timing *C.double
	if renderer.Profiling {
		timing = &gpuSeconds
	}
	C.sceneFrameFinish(frame, pixels, timing)
	renderer.Statistics = renderer.planner.Statistics(processorTime)
	if renderer.Profiling {
		renderer.Statistics.GPU = sceneplan.PassTimes{Scene: time.Duration(float64(gpuSeconds) * float64(time.Second))}
	}
	if picture != nil {
		swapRedAndBlue(picture)
		renderer.captured = picture
	}
}

// swapRedAndBlue turns Metal's BGRA into RGBA, and makes the frame opaque.
func swapRedAndBlue(picture *image.NRGBA) {
	for index := 0; index < len(picture.Pix); index += 4 {
		picture.Pix[index], picture.Pix[index+2] = picture.Pix[index+2], picture.Pix[index]
		picture.Pix[index+3] = 255
	}
}

// sceneResources are what the scene shaders read besides a draw's own uniforms.
type sceneResources struct {
	uniforms             *frameUniforms
	levelVariation       unsafe.Pointer
	levelVariationActive bool
	lights               []sceneplan.FrameLight // the frame's point lights, which each draw takes a few of
}

// bindScene sets the scene pipeline and what it reads: the frame's uniforms, the level variation,
// the shadow and seabed maps. Called again after the sky, which binds its own in the same places.
func (renderer *SceneRenderer) bindScene(encoder unsafe.Pointer, resources sceneResources, blended bool) {
	if blended {
		C.sceneEncoderSetPipeline(encoder, renderer.pipelines.sceneBlended)
		C.sceneEncoderSetDepthState(encoder, renderer.depthStates.blended)
	} else {
		C.sceneEncoderSetPipeline(encoder, renderer.pipelines.scene)
		C.sceneEncoderSetDepthState(encoder, renderer.depthStates.drawing)
	}
	size := C.long(unsafe.Sizeof(*resources.uniforms))
	C.sceneEncoderSetVertexBytes(encoder, unsafe.Pointer(resources.uniforms), size, 2)
	C.sceneEncoderSetFragmentBytes(encoder, unsafe.Pointer(resources.uniforms), size, 0)
	C.sceneEncoderSetVertexTexture(encoder, resources.levelVariation, 0)
	C.sceneEncoderSetFragmentTexture(encoder, renderer.shadowMapOrPlaceholder(), 0)
	C.sceneEncoderSetFragmentTexture(encoder, renderer.seabedMap, 1)
}

// setBlendedPipeline draws the next transparent commands by alpha, or adding their light.
func (renderer *SceneRenderer) setBlendedPipeline(encoder unsafe.Pointer, additive bool) {
	if additive {
		C.sceneEncoderSetPipeline(encoder, renderer.pipelines.sceneAdditive)
		return
	}
	C.sceneEncoderSetPipeline(encoder, renderer.pipelines.sceneBlended)
}

// drawCommands draws opaque or transparent commands from the camera.
func (renderer *SceneRenderer) drawCommands(encoder unsafe.Pointer, commands []sceneplan.Command, view sceneplan.Viewpoint, resources sceneResources, blended bool) {
	if len(commands) == 0 {
		return
	}
	renderer.bindScene(encoder, resources, blended)
	additive := false
	for _, command := range commands {
		if wanted := sceneplan.BlendingOf(command) == sceneplan.BlendAdditive; wanted != additive {
			additive = wanted
			renderer.setBlendedPipeline(encoder, additive)
		}
		uniforms := drawUniformsFor(command, resources.levelVariationActive, resources.lights)
		renderer.drawGeometry(encoder, command, view, sceneplan.MainPass, uniforms)
	}
}

// drawGeometry draws a command's mesh as seen from a viewpoint, culled and at the level of detail
// sceneplan chooses, with these uniforms (their instanced flag is set here).
func (renderer *SceneRenderer) drawGeometry(encoder unsafe.Pointer, command sceneplan.Command, view sceneplan.Viewpoint, which sceneplan.Pass, uniforms drawUniforms) {
	if command.Node.Instances != nil {
		uniforms.instanced = 1
		for _, batch := range renderer.planner.InstanceBatches(command, view, which) {
			instances, _ := batch.Handle.(*metalBatch)
			if instances == nil || batch.Count == 0 {
				continue
			}
			renderer.drawMesh(encoder, renderer.uploadMesh(batch.Mesh), instances.buffer, batch.Count, &uniforms)
			renderer.planner.Count(which, batch.Mesh, batch.Count)
		}
		return
	}
	mesh, visible := renderer.planner.MeshToDraw(command, view)
	if !visible {
		return
	}
	renderer.drawMesh(encoder, renderer.uploadMesh(mesh), renderer.placeholders.instances, 0, &uniforms)
	renderer.planner.Count(which, mesh, 0)
}

// drawMesh makes one draw call of a mesh, instanced when instances is above 0.
func (renderer *SceneRenderer) drawMesh(encoder unsafe.Pointer, mesh *metalMesh, instanceBuffer unsafe.Pointer, instances int, uniforms *drawUniforms) {
	size := C.long(unsafe.Sizeof(*uniforms))
	C.sceneEncoderSetVertexBytes(encoder, unsafe.Pointer(uniforms), size, 3)
	C.sceneEncoderSetFragmentBytes(encoder, unsafe.Pointer(uniforms), size, 1)
	C.sceneEncoderSetVertexBuffer(encoder, mesh.vertices, 0)
	C.sceneEncoderSetVertexBuffer(encoder, instanceBuffer, 1)
	if mesh.count == 0 {
		return
	}
	if mesh.indices != nil {
		C.sceneEncoderDrawIndexed(encoder, C.int(mesh.count), mesh.indices, C.int(instances))
		return
	}
	C.sceneEncoderDraw(encoder, C.scenePrimitiveTriangles, 0, C.int(mesh.count), C.int(instances))
}

// drawShadows draws the shadow map from the sun when shadows are enabled, and tells whether the
// lighting can use it.
func (renderer *SceneRenderer) drawShadows(frame unsafe.Pointer, environment scene.Environment, plan sceneplan.Frame, seconds float64) bool {
	if !plan.Shadow.Enabled {
		return false
	}
	renderer.matchShadowResolution()
	encoder := C.scenePassBeginDepth(frame, renderer.shadowMap)
	C.sceneEncoderSetPipeline(encoder, renderer.pipelines.depthOnly)
	C.sceneEncoderSetDepthState(encoder, renderer.depthStates.shadow)
	// the OpenGL renderer's PolygonOffset(2, 4): pushes casters back so lit surfaces do not shadow themselves
	C.sceneEncoderSetDepthBias(encoder, 4, 2)
	uniforms := shadowPassUniforms(environment, plan.Shadow.Region, seconds)
	C.sceneEncoderSetVertexBytes(encoder, unsafe.Pointer(&uniforms), C.long(unsafe.Sizeof(uniforms)), 2)
	C.sceneEncoderSetVertexTexture(encoder, renderer.placeholders.levelVariation, 0)
	for _, command := range plan.Commands {
		if !sceneplan.CastsShadow(command) {
			continue
		}
		caster := drawUniforms{model: command.World, sway: command.Material.Sway, depthBias: command.Material.DepthBias}
		renderer.drawGeometry(encoder, command, plan.Shadow.View, sceneplan.ShadowPass, caster)
	}
	C.sceneEncoderEnd(encoder)
	return true
}

// matchShadowResolution makes the shadow map again when the requested resolution changes.
func (renderer *SceneRenderer) matchShadowResolution() {
	wanted := renderer.Shadows.EffectiveResolution()
	if renderer.shadowMap != nil && renderer.shadowResolution == wanted {
		return
	}
	C.sceneRelease(renderer.shadowMap)
	renderer.shadowMap = C.sceneTextureCreate(renderer.device, C.sceneFormatDepth32, C.int(wanted), C.int(wanted), 0, 0, 1)
	renderer.shadowResolution = wanted
}

// shadowMapOrPlaceholder is the shadow map, or the seabed map before the first shadow is drawn:
// the shader reads neither then, but Metal wants a depth texture bound.
func (renderer *SceneRenderer) shadowMapOrPlaceholder() unsafe.Pointer {
	if renderer.shadowMap != nil {
		return renderer.shadowMap
	}
	return renderer.seabedMap
}

// drawSeabed draws the depth of the ground seen from above when the plan says it changed.
func (renderer *SceneRenderer) drawSeabed(frame unsafe.Pointer, plan sceneplan.SeabedPlan) {
	if !plan.Redraw || !plan.Available {
		return
	}
	encoder := C.scenePassBeginDepth(frame, renderer.seabedMap)
	C.sceneEncoderSetPipeline(encoder, renderer.pipelines.depthOnly)
	C.sceneEncoderSetDepthState(encoder, renderer.depthStates.shadow)
	uniforms := seabedPassUniforms(plan.Region)
	C.sceneEncoderSetVertexBytes(encoder, unsafe.Pointer(&uniforms), C.long(unsafe.Sizeof(uniforms)), 2)
	C.sceneEncoderSetVertexTexture(encoder, renderer.placeholders.levelVariation, 0)
	for _, command := range plan.Casters {
		// the whole mesh, without culling or levels of detail
		ground := drawUniforms{model: command.World, sway: command.Material.Sway}
		mesh := renderer.uploadMesh(command.Node.Mesh)
		if command.Node.Instances != nil {
			ground.instanced = 1
			renderer.drawMesh(encoder, mesh, renderer.uploadInstances(command.Node.Instances), len(command.Node.Instances.Transforms), &ground)
			continue
		}
		renderer.drawMesh(encoder, mesh, renderer.placeholders.instances, 0, &ground)
	}
	C.sceneEncoderEnd(encoder)
}

// levelVariationTexture returns the texture of the water's level variation, uploading it when new
// or changed, and whether there is one; without one, a flat placeholder.
func (renderer *SceneRenderer) levelVariationTexture(water *scene.Water) (unsafe.Pointer, bool) {
	variation := levelVariationOf(water)
	if variation == nil {
		return renderer.placeholders.levelVariation, false
	}
	texture, uploaded := variation.GPU.(*metalTexture)
	if uploaded && !variation.Dirty {
		return texture.handle, true
	}
	if uploaded {
		C.sceneRelease(texture.handle)
	}
	data := sceneplan.InterleaveHarmonics(variation)
	texture = &metalTexture{handle: renderer.newTexture(C.sceneFormatRG32Float, variation.Columns, variation.Rows, false, false)}
	if len(data) > 0 {
		C.sceneTextureWrite(texture.handle, 0, C.int(variation.Columns), C.int(variation.Rows), unsafe.Pointer(&data[0]), C.long(variation.Columns*8))
	}
	variation.GPU, variation.Dirty = texture, false
	return texture.handle, true
}

// drawSky paints the sky where the opaque scene left the far plane uncovered, then the
// constellations, as the OpenGL sky renderer does.
func (renderer *SceneRenderer) drawSky(encoder unsafe.Pointer, environment scene.Environment, plan sceneplan.Frame, camera models.Camera, seconds float64) {
	inverse, invertible := plan.Matrices.ViewProjection.Inverse()
	if !invertible {
		return
	}
	sky := environment.EffectiveSky()
	uniforms := skyUniformsFor(environment, plan.Light, camera, inverse, seconds)
	C.sceneEncoderSetPipeline(encoder, renderer.pipelines.sky)
	C.sceneEncoderSetDepthState(encoder, renderer.depthStates.blended)
	C.sceneEncoderSetFragmentBytes(encoder, unsafe.Pointer(&uniforms), C.long(unsafe.Sizeof(uniforms)), 0)
	C.sceneEncoderSetFragmentTexture(encoder, renderer.skyTexture(sky.DayImage, false), 0)
	C.sceneEncoderSetFragmentTexture(encoder, renderer.skyTexture(sky.NightImage, false), 1)
	C.sceneEncoderSetFragmentTexture(encoder, renderer.skyTexture(sky.DayImage, true), 2)
	C.sceneEncoderSetFragmentTexture(encoder, renderer.skyTexture(sky.NightImage, true), 3)
	C.sceneEncoderDraw(encoder, C.scenePrimitiveTriangles, 0, 3, 0)
	renderer.drawConstellations(encoder, sky, plan, camera)
}

// skyTexture returns the texture of a sky image where the shader reads it as a panorama (cube
// false) or a cube, or a placeholder when the image is absent or the other kind.
func (renderer *SceneRenderer) skyTexture(skyImage *scene.SkyImage, cube bool) unsafe.Pointer {
	if skyImage == nil || skyImage.IsCube() != cube {
		if cube {
			return renderer.placeholders.cube
		}
		return renderer.placeholders.panorama
	}
	if texture, uploaded := skyImage.GPU.(*metalTexture); uploaded {
		return texture.handle
	}
	texture := &metalTexture{}
	if cube {
		size := skyImage.Faces[0].Bounds().Dx()
		texture.handle = renderer.newTexture(C.sceneFormatRGBA8sRGB, size, size, true, false)
		for face, picture := range skyImage.Faces {
			writePicture(texture.handle, face, picture)
		}
	} else {
		bounds := skyImage.Equirectangular.Bounds()
		texture.handle = renderer.newTexture(C.sceneFormatRGBA8sRGB, bounds.Dx(), bounds.Dy(), false, true)
		writePicture(texture.handle, 0, skyImage.Equirectangular)
		C.sceneTextureGenerateMipmaps(renderer.device, texture.handle)
	}
	skyImage.GPU = texture
	return texture.handle
}

// writePicture fills a texture slice with a picture's pixels, as RGBA.
func writePicture(texture unsafe.Pointer, slice int, picture image.Image) {
	bounds := picture.Bounds()
	pixels, isRGBA := picture.(*image.RGBA)
	if !isRGBA || pixels.Stride != bounds.Dx()*4 {
		pixels = image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(pixels, pixels.Bounds(), picture, bounds.Min, draw.Src)
	}
	C.sceneTextureWrite(texture, C.int(slice), C.int(bounds.Dx()), C.int(bounds.Dy()), unsafe.Pointer(&pixels.Pix[0]), C.long(pixels.Stride))
}

// constellationBuffer holds the stars and lines of the constellations of one sky.
type constellationBuffer struct {
	sky        *scene.Sky
	buffer     unsafe.Pointer
	pointCount int32
	lineRanges []sceneplan.LineRange
}

// drawConstellations draws the lines and stars of every constellation, fading them with daylight.
func (renderer *SceneRenderer) drawConstellations(encoder unsafe.Pointer, sky *scene.Sky, plan sceneplan.Frame, camera models.Camera) {
	if len(sky.Constellations) == 0 {
		return
	}
	buffer := renderer.constellationBufferFor(sky)
	uniforms := constellationUniforms{viewProjection: metalClipSpace(plan.Matrices.ViewProjection),
		starRotation: plan.Light.StarRotation, cameraPosition: camera.Position,
		visibility: sceneplan.ConstellationVisibility(sky, plan.Light), pointSize: 5}
	size := C.long(unsafe.Sizeof(uniforms))
	C.sceneEncoderSetPipeline(encoder, renderer.pipelines.constellationLines)
	C.sceneEncoderSetVertexBuffer(encoder, buffer.buffer, 0)
	for _, lines := range buffer.lineRanges {
		uniforms.lineColor = sceneplan.LineColor(lines.Color)
		C.sceneEncoderSetVertexBytes(encoder, unsafe.Pointer(&uniforms), size, 1)
		C.sceneEncoderSetFragmentBytes(encoder, unsafe.Pointer(&uniforms), size, 0)
		if lines.Count > 0 {
			C.sceneEncoderDraw(encoder, C.scenePrimitiveLines, C.int(lines.First), C.int(lines.Count), 0)
		}
	}
	uniforms.lineColor = graphics.Vector{1, 1, 1, 1}
	C.sceneEncoderSetPipeline(encoder, renderer.pipelines.constellationPoints)
	C.sceneEncoderSetVertexBytes(encoder, unsafe.Pointer(&uniforms), size, 1)
	C.sceneEncoderSetFragmentBytes(encoder, unsafe.Pointer(&uniforms), size, 0)
	C.sceneEncoderDraw(encoder, C.scenePrimitivePoints, 0, C.int(buffer.pointCount), 0)
}

// constellationBufferFor uploads the constellations of a sky the first time it is drawn.
func (renderer *SceneRenderer) constellationBufferFor(sky *scene.Sky) *constellationBuffer {
	if renderer.constellations != nil && renderer.constellations.sky == sky {
		return renderer.constellations
	}
	if renderer.constellations != nil {
		C.sceneRelease(renderer.constellations.buffer)
	}
	vertices, pointCount, lineRanges := sceneplan.ConstellationVertices(sky.Constellations)
	renderer.constellations = &constellationBuffer{sky: sky, buffer: renderer.newBuffer(vertices), pointCount: pointCount, lineRanges: lineRanges}
	return renderer.constellations
}

// metalTexture is a texture kept in a scene object's GPU field.
type metalTexture struct{ handle unsafe.Pointer }

// metalMesh is a mesh's buffers on the GPU, kept in Mesh.GPU.
type metalMesh struct {
	vertices, indices unsafe.Pointer // indices is nil for a mesh without them
	count             int            // vertices read by a draw call
}

// uploadMesh returns the GPU buffers of a mesh, making or refreshing them when needed. A frame
// waits for the previous one to finish, so replacing buffers here never pulls them from under it.
func (renderer *SceneRenderer) uploadMesh(mesh *scene.Mesh) *metalMesh {
	buffers, _ := mesh.GPU.(*metalMesh)
	if buffers != nil && !mesh.Dirty {
		return buffers
	}
	if buffers != nil {
		C.sceneRelease(buffers.vertices)
		C.sceneRelease(buffers.indices)
	}
	buffers = &metalMesh{vertices: renderer.newBuffer(sceneplan.InterleaveVertices(mesh)), count: sceneplan.ElementCount(mesh)}
	if len(mesh.Indices) > 0 {
		buffers.indices = C.sceneBufferCreate(renderer.device, unsafe.Pointer(&mesh.Indices[0]), C.long(len(mesh.Indices)*4))
	}
	mesh.GPU, mesh.Dirty = buffers, false
	return buffers
}

// metalInstances is every instance of a node on the GPU, kept in Instances.GPU: the seabed map
// draws them all.
type metalInstances struct{ buffer unsafe.Pointer }

func (renderer *SceneRenderer) uploadInstances(instances *scene.Instances) unsafe.Pointer {
	buffers, _ := instances.GPU.(*metalInstances)
	if buffers != nil && !instances.Dirty {
		return buffers.buffer
	}
	if buffers != nil {
		C.sceneRelease(buffers.buffer)
	}
	buffers = &metalInstances{buffer: renderer.newBuffer(sceneplan.PackInstances(instances))}
	instances.GPU, instances.Dirty = buffers, false
	return buffers.buffer
}

// metalBatch is the buffer of one instance batch the planner chose.
type metalBatch struct {
	buffer   unsafe.Pointer
	capacity int // floats the buffer holds
}

// metalBatches keeps the planner's instance batches in Metal buffers.
type metalBatches struct{ renderer *SceneRenderer }

// FillBatch writes a batch's instances, in its buffer when they fit, else in a larger one.
func (batches metalBatches) FillBatch(handle any, mesh *scene.Mesh, packed []float32) any {
	batch, _ := handle.(*metalBatch)
	if batch == nil || batch.capacity < len(packed) {
		if batch != nil {
			C.sceneRelease(batch.buffer)
		}
		batch = &metalBatch{buffer: batches.renderer.newBuffer(packed), capacity: len(packed)}
		return batch
	}
	if len(packed) > 0 {
		C.sceneBufferWrite(batch.buffer, 0, unsafe.Pointer(&packed[0]), C.long(len(packed)*4))
	}
	return batch
}

// RewriteInstances replaces some instances of a batch in place.
func (batches metalBatches) RewriteInstances(handle any, position int, packed []float32) {
	batch := handle.(*metalBatch)
	offset := position * sceneplan.FloatsPerInstance * 4
	C.sceneBufferWrite(batch.buffer, C.long(offset), unsafe.Pointer(&packed[0]), C.long(len(packed)*4))
}
