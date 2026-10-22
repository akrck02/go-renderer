# window

```go
import "github.com/akrck02/go-renderer/window"
```

Package window describes a program that draws in a window: what the window is
(Options), what the program does every frame (Callbacks) and what the two share
while it runs (Application). A backend (opengl.Window, metal.Window, the Android
and browser loops) opens the window and drives the application.

### Constants and variables

```go
const DefaultCapturePath = "frame.png"
```

DefaultCapturePath is where a Headless application saves its frame when CapturePath is empty.

### Application

Application is a program running in a window: its Options and Callbacks, and what the backend and the program share while it runs.

```go
type Application struct {
	Options
	Callbacks
	// Input, when not nil, is filled by the backend every frame (keyboard, mouse, touches, pads).
	Input *input.State
	// Renderer draws immediate-mode shapes; the backend fills it.
	Renderer immediate.Renderer
	// Camera is where the immediate-mode shapes are seen from.
	Camera camera.Lens
	// Stop, when set by Update or Draw, ends the application after the current frame.
	Stop bool
	// ModelMatrix, ViewMatrix and ProjectionMatrix are the immediate-mode matrices: the backend
	// sets the view and projection from Camera each frame, and the model matrix starts as the
	// identity.
	ModelMatrix      graphics.Matrix
	ViewMatrix       graphics.Matrix
	ProjectionMatrix graphics.Matrix
}
```

### Backend

Backend opens a window (or a hidden one, for a Headless application) and drives an Application in it: opengl.Window, metal.Window and the software, Vulkan and WebGPU stubs.

```go
type Backend interface {
	// Run opens the window and runs the application until it closes or sets Stop.
	Run(app *Application) error
	// Draw draws one frame of the application.
	Draw(app *Application) error
}
```

### Callbacks

Callbacks are what the program does every frame. Each may be nil.

```go
type Callbacks struct {
	// Update runs at a fixed time step (Options.FixedStepSeconds) before drawing: where
	// simulations and camera controllers advance. stepSeconds is always that step.
	Update func(app *Application, stepSeconds float64) error
	// Draw draws a frame.
	Draw func(app *Application) error
	// Unchanged, when set, is asked before each frame is drawn: true means the frame would look
	// exactly like the last one shown, and the backend skips drawing and presenting it. An
	// interface answers by comparing its draw list's fingerprint (draw2d.List.Fingerprint), so a
	// pointer moving over still parts of a window costs an update step and nothing more.
	Unchanged func(app *Application) bool
}
```

### Kind

Kind says whether a window is shown.

```go
type Kind int
```

```go
const (
	// OnScreen is a window shown on the screen, the default.
	OnScreen Kind = iota
	// Headless draws one frame in a hidden window and saves it to Options.CapturePath.
	Headless
)
```

### Options

Options are what the window is: chosen before it opens.

```go
type Options struct {
	Kind Kind
	// Title and Version are shown in the window's title bar.
	Title   string
	Version string
	// Width and Height are the window's size in screen coordinates.
	Width  int
	Height int
	// MinWidth and MinHeight, when above 0, stop the window from being made smaller.
	MinWidth, MinHeight int
	// Background is the colour (red, green, blue, alpha from 0 to 1) a frame is cleared to.
	Background graphics.Vector
	// FixedStepSeconds is the time step of Callbacks.Update; 0 is 1/60 of a second.
	FixedStepSeconds float64
	// CapturePath is where a Headless application saves its frame (PNG); DefaultCapturePath
	// when empty.
	CapturePath string
	// DrawOnDemand draws only when something happens (input, a resize, a frame asked for with
	// Input.RequestFrame, or opengl.Wake from another goroutine) instead of every frame: the
	// program sleeps in between, which an interface that is mostly still needs for the battery.
	// Each frame runs exactly one update step. Needs Application.Input.
	DrawOnDemand bool
	// FlatOnly asks for a window with no depth or stencil buffer: a program that only draws flat
	// shapes (an interface, see opengl.FlatRenderer) saves a buffer the size of the window, about
	// 14 MB for a 1280 × 720 window on a high-density screen. It also skips the 3D renderer's
	// shaders and its part of each frame (the flat renderer clears and draws the whole window), and
	// on a Mac with two GPUs lets the system keep the window on the integrated, low-power one.
	FlatOnly bool
	// DisableVsync draws frames as fast as possible instead of waiting for the display (benchmarks).
	DisableVsync bool
	// ShowFramesPerSecond writes the frames drawn each second in the title, for a window that
	// draws every frame (not on demand).
	ShowFramesPerSecond bool
}
```

| Function or method | |
|---|---|
| `func (options Options) CapturePathOrDefault() string` | CapturePathOrDefault returns where a Headless application saves its frame. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
