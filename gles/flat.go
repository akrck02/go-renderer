package gles

import (
	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
)

// FlatRenderer draws a 2D draw list on a plain background, with no scene, sky or camera: the
// window of an application that is only an interface. It does what opengl.FlatRenderer does on
// the desktop, with the same atlas and one draw call per frame. Create and use it where the
// graphics context lives.
type FlatRenderer struct {
	// Target is the framebuffer Draw draws into: 0, the screen, unless the caller made one to
	// draw a picture offscreen.
	Target Framebuffer

	functions Functions
	quads     *quadPass
	layer     scene.Overlay
}

// NewFlatRenderer compiles the shaders and prepares the atlas; text is drawn in fonts
// (overlay.Fonts{} for the default ones).
func NewFlatRenderer(functions Functions, fonts overlay.Fonts) (*FlatRenderer, error) {
	quads, err := newQuadPass(functions, fonts)
	if err != nil {
		return nil, err
	}
	return &FlatRenderer{functions: functions, quads: quads}, nil
}

// Draw clears the target framebuffer (width × height pixels) to the background colour and paints
// the list. scale is framebuffer pixels per layout pixel: 2 on a high-density screen.
func (renderer *FlatRenderer) Draw(list *draw2d.List, background draw2d.Color, width, height int, scale float32) {
	renderer.functions.BindFramebuffer(FramebufferTarget, renderer.Target)
	renderer.quads.target = renderer.Target
	renderer.functions.Viewport(0, 0, width, height)
	renderer.functions.ClearColor(background[0], background[1], background[2], 1)
	renderer.functions.Clear(ColorBufferBit)
	renderer.layer.Drawing = list
	renderer.quads.paintFlat(&renderer.layer, width, height, scale)
}
