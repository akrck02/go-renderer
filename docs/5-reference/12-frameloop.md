# frameloop

```go
import "github.com/akrck02/go-renderer/frameloop"
```

Package frameloop holds the parts of a window's main loop that do not depend
on any window system: the fixed-step update clock and the update step itself.
The desktop loop (windowloop, over GLFW) and the browser loop (browser, over a
canvas) share them, so an application advances the same way everywhere.

### Functions

| Function | |
|---|---|
| `func RunUpdateStep(app *window.Application, step float64) error` | RunUpdateStep calls the application update once and clears the per-step input deltas. |

### Clock

Clock runs Application.Update at a fixed time step, independent of the frame rate, so that simulations advance the same way on fast and slow machines.

```go
type Clock struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewClock(step float64) *Clock` | NewClock makes a clock of step seconds (1/60 when 0). |
| `func (clock *Clock) Advance(app *window.Application) error` | Advance runs as many update steps as the elapsed time requires (at most a quarter of a second worth, so a stall does not trigger a long catch-up). |
| `func (clock *Clock) BorrowStep(app *window.Application) error` | BorrowStep runs one update step now, taken from the time of the next one: for a frame that has to be laid out again (the window changed size) before a step is due. |
| `func (clock *Clock) SleepUntilNextStep()` | SleepUntilNextStep waits until the next update step is due. |
| `func (clock *Clock) Step() float64` | Step is the clock's time step in seconds. |
| `func (clock *Clock) StepOnce(app *window.Application) error` | StepOnce runs exactly one update step and restarts the clock from now: a loop that draws on demand runs one step per frame, and after a long sleep the clock would otherwise catch up in a burst. |
| `func (clock *Clock) Steps() int` | Steps is the number of update steps run so far. |
| `func (clock *Clock) UntilNextStep() time.Duration` | UntilNextStep is how long until the next update step is due. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
