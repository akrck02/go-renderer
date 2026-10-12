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
	layer.instances = packInstances(quads, layer.instances)
	renderer.uploadOverlayInstances()
	size := float32(layer.painter.Atlas().Size())
	sizes := overlaySizes{screen: [2]float32{float32(width), float32(height)}, atlas: [2]float32{size, size}}
	C.sceneEncoderSetPipeline(encoder, renderer.pipelines.overlay)
	C.sceneEncoderSetDepthState(encoder, renderer.depthStates.overlay)
	C.sceneEncoderSetVertexBuffer(encoder, layer.buffer, 0)
	C.sceneEncoderSetVertexBytes(encoder, unsafe.Pointer(&sizes), C.long(unsafe.Sizeof(sizes)), 1)
	C.sceneEncoderSetFragmentTexture(encoder, layer.atlas, 0)
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
