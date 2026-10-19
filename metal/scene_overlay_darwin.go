package metal

/*
#include "scene_darwin.h"
*/
import "C"

import (
	"unsafe"

	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/sceneplan"
)

// overlayLayer draws the scene's overlay (labels, header, notices) over the finished frame, with
// the same painter and atlas as the OpenGL renderer and the flat window.
type overlayLayer struct {
	painter      *overlay.Painter
	atlas        unsafe.Pointer
	atlasVersion int
	pictures     unsafe.Pointer // the images' colours; nil before any picture
	picturesSize int
	buffer       unsafe.Pointer
	capacity     int // instances the buffer holds
	instances    []instance
}

// drawOverlay lays out the overlay in screen space and paints it on top of everything, in one
// draw call.
func (renderer *SceneRenderer) drawOverlay(encoder unsafe.Pointer, world *scene.Scene, matrices sceneplan.CameraMatrices, width, height int) {
	if world.Overlay.IsEmpty() {
		return
	}
	layer := &renderer.overlay
	frame := overlay.Frame{ViewProjection: matrices.ViewProjection, Width: width, Height: height,
		Scale: renderer.OverlayScale, VerticalScale: world.Environment.EffectiveVerticalScale()}
	quads := layer.painter.Paint(&world.Overlay, world.WorldTransforms(anchoredNodes(world.Overlay.Labels)), frame)
	if len(quads) == 0 {
		return
	}
	renderer.uploadAtlas()
	renderer.uploadPictures()
	layer.instances = packInstances(quads, layer.instances, false) // TODO(akrck02): blur backdrops over a 3D scene in Metal too
	renderer.uploadOverlayInstances()
	size := float32(layer.painter.Atlas().Size())
	picturesSize := float32(layer.picturesSize)
	sizes := overlaySizes{screen: [2]float32{float32(width), float32(height)}, atlas: [2]float32{size, size},
		pictures: [2]float32{picturesSize, picturesSize}}
	C.sceneEncoderSetPipeline(encoder, renderer.pipelines.overlay)
	C.sceneEncoderSetDepthState(encoder, renderer.depthStates.overlay)
	C.sceneEncoderSetVertexBuffer(encoder, layer.buffer, 0)
	C.sceneEncoderSetVertexBytes(encoder, unsafe.Pointer(&sizes), C.long(unsafe.Sizeof(sizes)), 1)
	C.sceneEncoderSetFragmentTexture(encoder, layer.atlas, 0)
	pictures := layer.pictures
	if pictures == nil {
		pictures = layer.atlas // read by no quad until there is a picture
	}
	C.sceneEncoderSetFragmentTexture(encoder, pictures, 1)
	C.sceneEncoderDraw(encoder, C.scenePrimitiveTriangleStrip, 0, 4, C.int(len(layer.instances)))
}

// uploadAtlas sends the atlas to the GPU when glyphs, icons or corners were added to it.
func (renderer *SceneRenderer) uploadAtlas() {
	layer := &renderer.overlay
	atlas := layer.painter.Atlas()
	if atlas.Version == layer.atlasVersion {
		return
	}
	C.sceneRelease(layer.atlas)
	size := atlas.Size()
	layer.atlas = renderer.newTexture(C.sceneFormatR8, size, size, false, false)
	C.sceneTextureWrite(layer.atlas, 0, C.int(size), C.int(size), unsafe.Pointer(&atlas.Image.Pix[0]), C.long(atlas.Image.Stride))
	layer.atlasVersion = atlas.Version
}

// uploadPictures sends the images' colours: all of them in a new texture when the picture atlas
// is new, grew or was emptied, else the areas pictures were copied into.
func (renderer *SceneRenderer) uploadPictures() {
	layer := &renderer.overlay
	pictures := layer.painter.Pictures()
	size := pictures.Size()
	if size == 0 {
		return
	}
	whole, areas := pictures.Pending()
	stride := C.long(pictures.Image.Stride)
	if whole || layer.pictures == nil || size != layer.picturesSize {
		C.sceneRelease(layer.pictures)
		layer.pictures = renderer.newTexture(C.sceneFormatRGBA8, size, size, false, false)
		C.sceneTextureWrite(layer.pictures, 0, C.int(size), C.int(size), unsafe.Pointer(&pictures.Image.Pix[0]), stride)
		layer.picturesSize = size
	} else {
		for _, area := range areas {
			start := unsafe.Pointer(&pictures.Image.Pix[pictures.Image.PixOffset(area.Min.X, area.Min.Y)])
			C.sceneTextureWriteArea(layer.pictures, C.int(area.Min.X), C.int(area.Min.Y), C.int(area.Dx()), C.int(area.Dy()), start, stride)
		}
	}
	pictures.Uploaded()
}

// uploadOverlayInstances writes this frame's quads, in a larger buffer when they do not fit.
func (renderer *SceneRenderer) uploadOverlayInstances() {
	layer := &renderer.overlay
	bytes := len(layer.instances) * int(unsafe.Sizeof(instance{}))
	if layer.buffer == nil || layer.capacity < len(layer.instances) {
		C.sceneRelease(layer.buffer)
		layer.capacity = max(2*len(layer.instances), 256)
		layer.buffer = C.sceneBufferCreate(renderer.device, nil, C.long(layer.capacity*int(unsafe.Sizeof(instance{}))))
	}
	C.sceneBufferWrite(layer.buffer, 0, unsafe.Pointer(&layer.instances[0]), C.long(bytes))
}

// anchoredNodes lists the nodes labels follow, so only their transforms are worked out.
func anchoredNodes(labels []*scene.Label) map[*scene.Node]bool {
	nodes := map[*scene.Node]bool{}
	for _, label := range labels {
		if label != nil && label.Node != nil && !label.Hidden {
			nodes[label.Node] = true
		}
	}
	return nodes
}
