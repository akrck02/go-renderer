// Package camera provides camera controllers that turn input into camera placement.
//
// Controllers are independent of the graphics backend: they read an input.State and
// write a models.Camera. A GroundFunc lets them follow a terrain without knowing about it.
package camera

import (
	"math"

	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
)

// GroundFunc returns the ground height at (x, z) in world units, and false when there is no ground there.
type GroundFunc func(x, z float32) (float32, bool)

// Controller moves a camera from input.
type Controller interface {
	Update(state *input.State, seconds float64, camera *models.Camera)
}

func clamp(value, minimum, maximum float64) float64 {
	return math.Max(minimum, math.Min(maximum, value))
}

func scaleOrOne(scale float32) float32 {
	if scale == 0 {
		return 1
	}
	return scale
}
