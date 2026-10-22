//go:build !darwin

package metal

import (
	"github.com/akrck02/go-renderer/failure"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/window"
)

// FlatWindow draws through Metal, which only macOS has; elsewhere Run says so and a program
// uses the OpenGL backend.
type FlatWindow struct {
	// Fonts are what text would be drawn with, as on macOS.
	Fonts overlay.Fonts
}

// Run fails: Metal is only on macOS.
func (flat *FlatWindow) Run(app *window.Application) error {
	return failure.Wrap(Unavailable, "only on macOS", nil)
}

// SceneWindow draws a scene through Metal, which only macOS has; elsewhere Run says so.
type SceneWindow struct {
	// Fonts are what the overlay's text would be drawn with, as on macOS.
	Fonts overlay.Fonts
}

// Run fails: Metal is only on macOS.
func (scene *SceneWindow) Run(app *window.Application) error {
	return failure.Wrap(Unavailable, "only on macOS", nil)
}
