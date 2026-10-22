# windowloop

```go
import "github.com/akrck02/go-renderer/windowloop"
```

Package windowloop runs a GLFW window's main loop for any drawing backend
(OpenGL, Metal): the window's events, input, drawing on demand, the fixed-step
update clock, frames asked for while the window is being resized, the pointer's
shape and lock, and the whole screen. A backend only has to draw and show a
frame.

### Functions

| Function | |
|---|---|
| `func ApplyCursorLock(window *glfw.Window, state *input.State, currentlyLocked *bool)` | ApplyCursorLock captures or releases the cursor when the application changes its request. |
| `func ForwardInput(window *glfw.Window, state *input.State)` | ForwardInput sends GLFW keyboard, mouse, scroll and text events into an input.State, and gives it the system clipboard. |
| `func OnMainThread() bool` | OnMainThread is only checked on macOS, the one system that requires it. |
| `func Run(glfwWindow *glfw.Window, app *window.Application, clock *frameloop.Clock, drawFrame func() error) error` | Run calls the application's update steps and drawFrame until the window closes or the application stops. |
| `func UpdateSizes(window *glfw.Window, state *input.State)` | UpdateSizes copies the framebuffer and window sizes into the input state. |
| `func Wake()` | Wake makes a window that draws on demand draw a frame now: for a program whose data changes on another goroutine (a download finishing, a message arriving). |

### Constants and variables

```go
var NotOnMainThread = failure.Define("windowloop.not-on-main-thread",
```

### CursorShapes

CursorShapes shows the pointer shape the application asks for, creating each system cursor once.

```go
type CursorShapes struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (shapes *CursorShapes) Apply(window *glfw.Window, state *input.State)` | Apply shows the pointer shape the state asks for, if it is not already shown. |

### Fullscreen

Fullscreen puts a window on the whole screen when the application asks (input.State's SetFullscreen), and back in its place and at its size when it asks no more.

```go
type Fullscreen struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (fullscreen *Fullscreen) Apply(window *glfw.Window, state *input.State)` | Apply follows the application's request, when it changed. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
