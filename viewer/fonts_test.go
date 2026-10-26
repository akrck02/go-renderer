package viewer

import (
	"testing"

	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
	"golang.org/x/image/font/gofont/gomonobold"
)

// The options' fonts reach the session, which hands them to the renderer it makes.
func TestTheOptionsFontsAreTheSessions(test *testing.T) {
	session := newViewer(scene.New(), false)
	options := DefaultOptions()
	options.Rendering.Fonts = overlay.Fonts{Display: gomonobold.TTF}
	session.configure(options)
	if len(session.fonts.Display) != len(gomonobold.TTF) {
		test.Fatalf("the session draws with %d bytes of display font, want the options' %d", len(session.fonts.Display), len(gomonobold.TTF))
	}
}
