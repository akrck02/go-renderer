package gles

import (
	"math"

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
	// LinearOutput makes the colours, which are the screen's (sRGB), be written linear: for a
	// target that encodes sRGB itself, as a headset's panel does, where they would otherwise come
	// out twice as light.
	LinearOutput bool

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

// Draw clears the target framebuffer (width × height pixels) to the background colour, its alpha
// premultiplied (0 leaves it see-through where nothing is painted), and paints the list. scale is framebuffer pixels per layout pixel: 2 on a high-density screen.
func (renderer *FlatRenderer) Draw(list *draw2d.List, background draw2d.Color, width, height int, scale float32) {
	renderer.functions.BindFramebuffer(FramebufferTarget, renderer.Target)
	renderer.quads.target = renderer.Target
	renderer.quads.linearOutput = renderer.LinearOutput
	renderer.functions.SetViewport(0, 0, width, height)
	if renderer.LinearOutput {
		background = screenToLinear(background)
	}
	// premultiplied by its alpha, as a compositor blends a picture: a see-through background lets
	// what is behind the picture show where nothing is painted (a pill-shaped label on a panel)
	alpha := background[3]
	renderer.functions.SetClearColor(background[0]*alpha, background[1]*alpha, background[2]*alpha, alpha)
	renderer.functions.Clear(ColorBufferBit)
	renderer.layer.Drawing = list
	renderer.quads.paintFlat(&renderer.layer, width, height, scale)
}

// screenToLinear turns a colour in the screen's gamma (an interface's) into linear colour.
func screenToLinear(color draw2d.Color) draw2d.Color {
	return draw2d.Color{linearOf(color[0]), linearOf(color[1]), linearOf(color[2]), color[3]}
}

func linearOf(channel float32) float32 {
	if channel <= 0 {
		return 0
	}
	return float32(math.Pow(float64(channel), screenGamma))
}
