package models

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
)

type ApplicationType int

const (
	HeadlessApplication ApplicationType = 1
	WindowApplication   ApplicationType = 2
)

type Camera struct {
	Position    graphics.Vec4
	Target      graphics.Vec4
	Up          graphics.Vec4
	FieldOfView float64 // vertical field of view in radians (0 = pi/4)
	Near        float32 // near clipping plane (0 = 0.1)
	Far         float32 // far clipping plane (0 = 1000)
	// OrthographicHeight, when above 0, makes the camera orthographic: no perspective, and this
	// many world units fit between the bottom and the top of the screen. Maps and isometric views
	// use it; 0 keeps the perspective of FieldOfView.
	OrthographicHeight float32
}

// IsOrthographic tells whether the camera has no perspective.
func (camera Camera) IsOrthographic() bool { return camera.OrthographicHeight > 0 }

// ProjectionMatrix returns the camera's projection for OpenGL, perspective or orthographic, for a
// screen of this width over height.
func (camera Camera) ProjectionMatrix(aspect float64) graphics.Mat4 {
	fieldOfView, near, far := camera.Projection()
	if !camera.IsOrthographic() {
		return graphics.PerspectiveOpenGL(fieldOfView, aspect, float64(near), float64(far))
	}
	halfHeight := camera.OrthographicHeight / 2
	halfWidth := halfHeight * float32(aspect)
	return graphics.OrthographicOpenGL(-halfWidth, halfWidth, -halfHeight, halfHeight, near, far)
}

// Projection returns the camera frustum values, filling defaults for zero fields.
func (camera Camera) Projection() (fieldOfView float64, near, far float32) {
	fieldOfView, near, far = camera.FieldOfView, camera.Near, camera.Far
	if fieldOfView == 0 {
		fieldOfView = 0.7853981633974483
	}
	if near == 0 {
		near = 0.1
	}
	if far == 0 {
		far = 1000
	}
	return fieldOfView, near, far
}

type Application struct {
	Type       ApplicationType
	Background graphics.Vec4
	Width      int
	Height     int
	// MinWidth and MinHeight, when above 0, stop the window from being made smaller.
	MinWidth, MinHeight int
	Title               string
	Version             string
	Renderer            Renderer
	Draw                func(*Application) error
	// Update runs at a fixed time step (FixedStep seconds, 1/60 by default) before drawing.
	// It is where simulations and camera controllers advance; dt is always FixedStep.
	Update    func(app *Application, dt float64) error
	FixedStep float64
	// Input, when not nil, is filled by the window backend every frame (keyboard, mouse, scroll).
	Input *input.State
	// CapturePath is where a HeadlessApplication saves its frame (PNG); "frame.png" by default.
	CapturePath string
	// DrawOnDemand draws only when something happens (input, a resize, a frame asked for with
	// Input.RequestFrame, or opengl.Wake from another goroutine) instead of every frame: the
	// program sleeps in between, which an interface that is mostly still needs for the battery.
	// Each frame runs exactly one update step. Needs Input.
	DrawOnDemand bool
	// Unchanged, when set, is asked before each frame is drawn: true means the frame would look
	// exactly like the last one shown, and the backend skips drawing and presenting it. An
	// interface answers by comparing its draw list's fingerprint (draw2d.List.Fingerprint), so a
	// pointer moving over still parts of a window costs an update step and nothing more.
	Unchanged func(app *Application) bool
	// FlatOnly asks for a window with no depth or stencil buffer: a program that only draws flat
	// shapes (an interface, see opengl.FlatRenderer) saves a buffer the size of the window, about
	// 14 MB for a 1280 × 720 window on a high-density screen. It also skips the 3D renderer's
	// shaders and its part of each frame (the flat renderer clears and draws the whole window), and
	// on a Mac with two GPUs lets the system keep the window on the integrated, low-power one.
	FlatOnly bool
	// DisableVsync draws frames as fast as possible instead of waiting for the display (benchmarks).
	DisableVsync bool
	// Stop, when set by Update or Draw, ends a window application after the current frame.
	Stop             bool
	Camera           Camera
	ModelMatrix      graphics.Mat4
	ViewMatrix       graphics.Mat4
	ProjectionMatrix graphics.Mat4
}
