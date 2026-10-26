package openxr

import (
	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/graphics"
)

// MostPanels is how many panels a session can have: each is a layer the runtime blends every
// frame, and runtimes take a handful.
const MostPanels = 8

// Panel is a flat picture the runtime places in the world, over what the eyes show: an interface
// the person reads in the headset. The runtime draws it itself, sharper than the same picture
// drawn into the eyes would be. A panel is drawn into (Target) only when its picture changes: the
// runtime keeps showing the last one.
type Panel struct {
	// Pose is where the middle of the panel is and which way it faces, in the session's space: its
	// picture faces +Z, so a panel turned like the head faces whoever looks from its +Z side.
	Pose graphics.Pose
	// WidthMetres and HeightMetres are its size in the world (FitWidth keeps its pixels square).
	WidthMetres, HeightMetres float32
	// Shown puts it in the frames given back; a panel never drawn into shows nothing either way.
	Shown bool

	width, height int // in pixels
	// acquire takes the frame's image to draw into (the session's, on Android)
	acquire func() (gles.Framebuffer, error)
}

// DetachedPanel is a panel of width × height pixels no session made: it cannot be drawn (Target
// fails), only placed, for working out what a panel shows and where off the headset, as in tests.
func DetachedPanel(width, height int) *Panel { return &Panel{width: width, height: height} }

// Width and Height are the panel's picture in pixels, what it was made with.
func (panel *Panel) Width() int  { return panel.width }
func (panel *Panel) Height() int { return panel.height }

// FitWidth makes the panel metres wide and as tall as its pixels' proportions say.
func (panel *Panel) FitWidth(metres float32) {
	panel.WidthMetres = metres
	panel.HeightMetres = metres * float32(panel.height) / float32(panel.width)
}

// Target takes the panel's image for this frame and hands out its framebuffer, Width × Height
// pixels with no depth buffer (gles.FlatRenderer draws into it). It is only for a frame that is
// drawn (Frame.ShouldRender), between BeginFrame and EndFrame; the picture is shown from EndFrame
// on. Asked twice in a frame, it is the same image. The image encodes sRGB when the eyes' do
// (Session.SRGB): draw linear colour into it then (gles.FlatRenderer.LinearOutput).
func (panel *Panel) Target() (gles.Framebuffer, error) {
	if panel.acquire == nil {
		return 0, PanelNotDrawable
	}
	return panel.acquire()
}

// shows tells whether the panel goes in the frame given back: shown, with a size.
func (panel *Panel) shows() bool {
	return panel.Shown && panel.WidthMetres > 0 && panel.HeightMetres > 0
}
