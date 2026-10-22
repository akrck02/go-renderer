package metal

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework Metal -framework QuartzCore
#include <stdlib.h>
#include "flat_darwin.h"
*/
import "C"

import (
	"image"
	"unsafe"

	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/failure"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
)

// FlatRenderer draws a 2D draw list on a plain background through Metal: the window of an
// application that is only an interface. It shares the OpenGL path's painter, so both draw the
// same quads of the same atlas; here each quad is one 52-byte instance and a frame is one draw
// call, or one more around each snapshot of a frosted layer. Create and use it on the main thread.
type FlatRenderer struct {
	handle       unsafe.Pointer
	painter      *overlay.Painter
	layer        scene.Overlay
	atlasVersion int
	instances    []instance
	snapshots    []snapshot // where the frame is blurred for the backdrops after it
	lights       []light    // the frame's 2D lights
	lightings    []lighting // where the frame is lit by them
	// capturing draws frames into an off-screen picture (captured) instead of the window.
	capturing bool
	captured  *image.NRGBA
}

// NewFlatRenderer puts a Metal layer in a window made with no OpenGL context
// (glfw.ClientAPI = glfw.NoAPI) and prepares the shaders and atlas. vsync paces frames with the
// display.
func NewFlatRenderer(window *glfw.Window, vsync bool) (*FlatRenderer, error) {
	painter, err := overlay.NewPainter()
	if err != nil {
		return nil, err
	}
	var problem *C.char
	handle := C.flatRendererCreate(window.GetCocoaWindow(), C.int(boolean(vsync)), &problem)
	if handle == nil {
		defer C.free(unsafe.Pointer(problem))
		return nil, failure.Wrap(Unavailable, C.GoString(problem), nil)
	}
	return &FlatRenderer{handle: handle, painter: painter}, nil
}

// Draw clears the window (width × height framebuffer pixels) to the background colour, paints the
// list and shows the frame. scale is framebuffer pixels per layout pixel: 2 on a high-density
// screen.
func (renderer *FlatRenderer) Draw(list *draw2d.List, background draw2d.Color, width, height int, scale float32) {
	if width <= 0 || height <= 0 {
		return
	}
	renderer.layer.Drawing = list
	quads := renderer.painter.Paint(&renderer.layer, nil, overlay.Frame{Width: width, Height: height, Scale: scale})
	renderer.uploadAtlas()
	renderer.uploadPictures()
	renderer.instances = packInstances(quads, renderer.instances, true)
	renderer.snapshots = packSnapshots(renderer.painter.Snapshots(), renderer.snapshots)
	renderer.lights = packLights(renderer.painter.Lights(), renderer.lights)
	renderer.lightings = packLightings(renderer.painter.Lightings(), renderer.lightings)
	colour := [4]C.float{C.float(background[0]), C.float(background[1]), C.float(background[2]), 1}
	if renderer.capturing {
		renderer.capture(&colour[0], width, height)
		return
	}
	C.flatRendererDraw(renderer.handle, renderer.instancePointer(), C.int(len(renderer.instances)),
		renderer.snapshotPointer(), C.int(len(renderer.snapshots)),
		renderer.lightPointer(), C.int(len(renderer.lights)), renderer.lightingPointer(), C.int(len(renderer.lightings)),
		&colour[0], C.int(width), C.int(height))
}

// Close lets go of the layer's GPU resources.
func (renderer *FlatRenderer) Close() {
	if renderer.handle != nil {
		C.flatRendererDestroy(renderer.handle)
		renderer.handle = nil
	}
}

// uploadAtlas sends the atlas to the GPU when glyphs, icons or corners were added to it.
func (renderer *FlatRenderer) uploadAtlas() {
	atlas := renderer.painter.Atlas()
	if atlas.Version == renderer.atlasVersion {
		return
	}
	C.flatRendererUploadAtlas(renderer.handle, (*C.uint8_t)(unsafe.Pointer(&atlas.Image.Pix[0])), C.int(atlas.Size()))
	renderer.atlasVersion = atlas.Version
}

// uploadPictures sends the images' colours: all of them when the picture atlas is new, grew or
// was emptied, else the areas pictures were copied into.
func (renderer *FlatRenderer) uploadPictures() {
	pictures := renderer.painter.Pictures()
	size := pictures.Size()
	if size == 0 {
		return // no picture drawn yet
	}
	pixels := (*C.uint8_t)(unsafe.Pointer(&pictures.Image.Pix[0]))
	whole, areas := pictures.Pending()
	if whole {
		C.flatRendererUploadPictures(renderer.handle, pixels, C.int(size), 1, 0, 0, C.int(size), C.int(size))
	}
	for _, area := range areas {
		C.flatRendererUploadPictures(renderer.handle, pixels, C.int(size), 0,
			C.int(area.Min.X), C.int(area.Min.Y), C.int(area.Dx()), C.int(area.Dy()))
	}
	pictures.Uploaded()
}

func (renderer *FlatRenderer) snapshotPointer() *C.flatSnapshot {
	if len(renderer.snapshots) == 0 {
		return nil
	}
	return (*C.flatSnapshot)(unsafe.Pointer(&renderer.snapshots[0]))
}

func (renderer *FlatRenderer) lightPointer() *C.flatLight {
	if len(renderer.lights) == 0 {
		return nil
	}
	return (*C.flatLight)(unsafe.Pointer(&renderer.lights[0]))
}

func (renderer *FlatRenderer) lightingPointer() *C.flatLighting {
	if len(renderer.lightings) == 0 {
		return nil
	}
	return (*C.flatLighting)(unsafe.Pointer(&renderer.lightings[0]))
}

func (renderer *FlatRenderer) instancePointer() *C.flatInstance {
	if len(renderer.instances) == 0 {
		return nil
	}
	return (*C.flatInstance)(unsafe.Pointer(&renderer.instances[0]))
}

// capture draws the frame off screen and keeps it as a picture, turning Metal's BGRA into RGBA.
func (renderer *FlatRenderer) capture(background *C.float, width, height int) {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	C.flatRendererCapture(renderer.handle, renderer.instancePointer(), C.int(len(renderer.instances)),
		renderer.snapshotPointer(), C.int(len(renderer.snapshots)),
		renderer.lightPointer(), C.int(len(renderer.lights)), renderer.lightingPointer(), C.int(len(renderer.lightings)),
		background, C.int(width), C.int(height), (*C.uint8_t)(unsafe.Pointer(&picture.Pix[0])))
	swapRedAndBlue(picture)
	renderer.captured = picture
}
