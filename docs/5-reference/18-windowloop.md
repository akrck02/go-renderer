# windowloop

```go
import "github.com/akrck02/go-renderer/windowloop"
```

Package windowloop runs a GLFW window's main loop for any drawing backend
(OpenGL, Metal): the window's events, input, drawing on demand, the fixed-step
update clock, frames asked for while the window is being resized, the pointer's
shape and lock. A backend only has to draw and show a frame.

### Functions

| Function | |
|---|---|
| `func ApplyCursorLock(window *glfw.Window, state *input.State, currentlyLocked *bool)` | ApplyCursorLock captures or releases the cursor when the application changes its request. |
| `func ForwardInput(window *glfw.Window, state *input.State)` | ForwardInput sends GLFW keyboard, mouse, scroll and text events into an input.State, and gives it the system clipboard. |
| `func OnMainThread() bool` | OnMainThread is only checked on macOS, the one system that requires it. |
| `func ReadGamePad(state *input.State)` | ReadGamePad copies the first game pad's sticks into the input state, or leaves them at rest without one. |
| `func Run(window *glfw.Window, app *models.Application, clock *Clock, drawFrame func() error) error` | Run calls the application's update steps and drawFrame until the window closes or the application stops. |
| `func RunUpdateStep(app *models.Application, step float64) error` | RunUpdateStep calls the application update once and clears the per-step input deltas. |
| `func UpdateSizes(window *glfw.Window, state *input.State)` | UpdateSizes copies the framebuffer and window sizes into the input state. |
| `func Wake()` | Wake makes a window that draws on demand draw a frame now: for a program whose data changes on another goroutine (a download finishing, a message arriving). |

### Clock

Clock runs Application.Update at a fixed time step; it lives in frameloop, which the browser loop shares.

```go
type Clock = frameloop.Clock
```

| Function or method | |
|---|---|
| `func NewClock(step float64) *Clock` | NewClock makes a clock of step seconds (1/60 when 0). |

### CursorShapes

CursorShapes shows the pointer shape the application asks for, creating each system cursor once.

```go
type CursorShapes struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (shapes *CursorShapes) Apply(window *glfw.Window, state *input.State)` |  |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
