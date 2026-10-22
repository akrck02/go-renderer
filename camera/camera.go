// Package camera provides camera controllers that turn input into camera placement.
//
// Controllers are independent of the graphics backend: they read an input.State and
// write a Camera. A GroundFunc lets them follow a terrain without knowing about it.
package camera

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
)

// Default projection values, used where a Camera's fields are zero.
const (
	DefaultFieldOfViewRadians = math.Pi / 4
	DefaultNear               = 0.1
	DefaultFar                = 1000
)

// Camera is where a scene is seen from: a position looking at a target, with up saying which way
// is up, and how the view is projected onto the screen.
type Camera struct {
	Position graphics.Vector
	Target   graphics.Vector
	Up       graphics.Vector
	// FieldOfViewRadians is the vertical field of view; 0 is DefaultFieldOfViewRadians.
	FieldOfViewRadians float32
	// Near and Far are the distances of the clipping planes; 0 is DefaultNear and DefaultFar.
	Near float32
	Far  float32
	// OrthographicHeight, when above 0, makes the camera orthographic: no perspective, and this
	// many world units fit between the bottom and the top of the screen. Maps and isometric views
	// use it; 0 keeps the perspective of FieldOfViewRadians.
	OrthographicHeight float32
}

// IsOrthographic tells whether the camera has no perspective.
func (camera Camera) IsOrthographic() bool { return camera.OrthographicHeight > 0 }

// ProjectionMatrix returns the camera's projection for OpenGL, perspective or orthographic, for a
// screen of this width over height.
func (camera Camera) ProjectionMatrix(aspect float64) graphics.Matrix {
	fieldOfView, near, far := camera.Projection()
	if !camera.IsOrthographic() {
		return graphics.PerspectiveOpenGL(float64(fieldOfView), aspect, float64(near), float64(far))
	}
	halfHeight := camera.OrthographicHeight / 2
	halfWidth := halfHeight * float32(aspect)
	return graphics.OrthographicOpenGL(-halfWidth, halfWidth, -halfHeight, halfHeight, near, far)
}

// Projection returns the camera's field of view (radians) and clipping planes, with the defaults
// where its fields are zero.
func (camera Camera) Projection() (fieldOfViewRadians, near, far float32) {
	fieldOfViewRadians, near, far = camera.FieldOfViewRadians, camera.Near, camera.Far
	if fieldOfViewRadians == 0 {
		fieldOfViewRadians = DefaultFieldOfViewRadians
	}
	if near == 0 {
		near = DefaultNear
	}
	if far == 0 {
		far = DefaultFar
	}
	return fieldOfViewRadians, near, far
}

// GroundFunc returns the ground height at (x, z) in world units, and false when there is no ground there.
type GroundFunc func(x, z float32) (float32, bool)

// Controller moves a camera from input.
type Controller interface {
	Update(state *input.State, seconds float64, camera *Camera)
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
