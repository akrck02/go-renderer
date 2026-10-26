//go:build !darwin

package viewer

import (
	"github.com/akrck02/go-renderer/metal"
	"github.com/akrck02/go-renderer/window"
)

// runMetal fails everywhere but macOS, and the session falls back to OpenGL.
func runMetal(session *viewer, app *window.Application) error {
	return (&metal.SceneWindow{Fonts: session.fonts}).Run(app)
}
