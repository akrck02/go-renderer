//go:build !darwin

package metal

import (
	"fmt"

	"github.com/akrck02/go-renderer/models"
)

// FlatWindow draws through Metal, which only macOS has; elsewhere StartLoop says so and a program
// uses the OpenGL backend.
type FlatWindow struct{}

// StartLoop fails: Metal is only on macOS.
func (flat *FlatWindow) StartLoop(app *models.Application) error {
	return fmt.Errorf("%w: only on macOS", ErrUnavailable)
}

// SceneWindow draws a scene through Metal, which only macOS has; elsewhere StartLoop says so.
type SceneWindow struct{}

// StartLoop fails: Metal is only on macOS.
func (scene *SceneWindow) StartLoop(app *models.Application) error {
	return fmt.Errorf("%w: only on macOS", ErrUnavailable)
}
