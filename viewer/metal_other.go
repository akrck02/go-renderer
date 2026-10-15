//go:build !darwin

package viewer

import (
	"github.com/akrck02/go-renderer/metal"
	"github.com/akrck02/go-renderer/models"
)

// runMetal fails everywhere but macOS, and the session falls back to OpenGL.
func runMetal(session *viewer, app *models.Application) error {
	return (&metal.SceneWindow{}).StartLoop(app)
}
