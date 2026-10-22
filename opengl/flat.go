package opengl

import (
	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// FlatRenderer draws a 2D draw list on a plain background, with no scene, sky or camera: the
// window of an application that is only an interface. It uses the overlay's atlas and shaders, so
// a whole frame is one clear and one draw call. Create and use it on the thread that owns the
// OpenGL context.
type FlatRenderer struct {
	overlay *overlayRenderer
	layer   scene.Overlay
}

// NewFlatRenderer prepares the atlas and shaders; text is drawn in fonts (overlay.Fonts{} for the
// default ones). Call it after the OpenGL context exists.
func NewFlatRenderer(fonts overlay.Fonts) (*FlatRenderer, error) {
	overlay, err := newOverlayRenderer(fonts)
	if err != nil {
		return nil, err
	}
	return &FlatRenderer{overlay: overlay}, nil
}

// Draw clears the framebuffer (width × height pixels) to the background colour and paints the
// list. scale is framebuffer pixels per layout pixel: 2 on a high-density screen.
func (renderer *FlatRenderer) Draw(list *draw2d.List, background draw2d.Color, width, height int, scale float32) {
	gl.Viewport(0, 0, int32(width), int32(height))
	gl.ClearColor(background[0], background[1], background[2], 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	renderer.layer.Drawing = list
	renderer.overlay.drawFlat(&renderer.layer, width, height, scale)
}
