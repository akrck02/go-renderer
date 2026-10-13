package gles

import (
	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/scene"
)

// FlatRenderer draws a 2D draw list on a plain background, with no scene, sky or camera: the
// window of an application that is only an interface. It does what opengl.FlatRenderer does on
// the desktop, with the same atlas and one draw call per frame. Create and use it where the
// graphics context lives.
type FlatRenderer struct {
	functions Functions
	quads     *quadPass
	layer     scene.Overlay
}

// NewFlatRenderer compiles the shaders and prepares the atlas.
func NewFlatRenderer(functions Functions) (*FlatRenderer, error) {
	quads, err := newQuadPass(functions)
	if err != nil {
		return nil, err
	}
	return &FlatRenderer{functions: functions, quads: quads}, nil
}

// Draw clears the framebuffer (width × height pixels) to the background colour and paints the
// list. scale is framebuffer pixels per layout pixel: 2 on a high-density screen.
func (renderer *FlatRenderer) Draw(list *draw2d.List, background draw2d.Color, width, height int, scale float32) {
	renderer.functions.Viewport(0, 0, width, height)
	renderer.functions.ClearColor(background[0], background[1], background[2], 1)
	renderer.functions.Clear(ColorBufferBit)
	renderer.layer.Drawing = list
	renderer.quads.paintFlat(&renderer.layer, width, height, scale)
}
