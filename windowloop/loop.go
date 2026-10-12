// Package windowloop runs a GLFW window's main loop for any drawing backend (OpenGL, Metal): the
// window's events, input, drawing on demand, the fixed-step update clock, frames asked for while
// the window is being resized, the pointer's shape and lock. A backend only has to draw and show
// a frame.
package windowloop

import (
	"fmt"
	"math"
	"time"

	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// Run calls the application's update steps and drawFrame until the window closes or the
// application stops. drawFrame draws and shows one frame; it may skip one that would look like
// the last (models.Application.Unchanged). It must be called on the main thread.
func Run(window *glfw.Window, app *models.Application, clock *Clock, drawFrame func() error) error {
	var failure error
	resize := followResizing(window, app, clock, drawFrame, &failure)
	onDemand := app.DrawOnDemand && app.Input != nil
	followUp := false
	cursorLocked := false
	var shapes CursorShapes
	frames, lastTitle := 0, time.Now()
	for !window.ShouldClose() && !app.Stop {
		// events first, so the frame drawn next is for the window as it is now
		if onDemand {
			followUp = waitForWork(app.Input, followUp)
		} else {
			glfw.PollEvents()
		}
		if failure != nil {
			return failure
		}
		stepsBefore := clock.steps
		if err := resize.advance(); err != nil {
			return err
		}
		if !onDemand && clock.steps == stepsBefore {
			// nothing was updated, so the frame would repeat the last one: wait for the next
			// step instead (the display may run faster than the updates, and vsync alone does
			// not hold the loop back on every system)
			clock.sleepUntilNextStep()
			continue
		}

		if err := drawFrame(); err != nil {
			return err
		}
		if app.Input != nil {
			ApplyCursorLock(window, app.Input, &cursorLocked)
			shapes.Apply(window, app.Input)
		}

		frames++
		if !onDemand && time.Since(lastTitle) >= time.Second {
			window.SetTitle(fmt.Sprintf("%s %s   |   %d FPS", app.Title, app.Version, frames))
			lastTitle, frames = time.Now(), 0
		}
	}
	return nil
}

// Clock runs Application.Update at a fixed time step, independent of the frame rate, so that
// simulations advance the same way on fast and slow machines.
type Clock struct {
	step        float64
	previous    time.Time
	accumulated float64
	steps       int // update steps run so far
}

// NewClock makes a clock of step seconds (1/60 when 0).
func NewClock(step float64) *Clock {
	if step <= 0 {
		step = 1.0 / 60
	}
	return &Clock{step: step, previous: time.Now()}
}

// Step is the clock's time step in seconds.
func (clock *Clock) Step() float64 { return clock.step }

// advance runs as many update steps as the elapsed time requires (at most a quarter of a second
// worth, so a stall does not trigger a long catch-up).
func (clock *Clock) advance(app *models.Application) error {
	elapsed := time.Since(clock.previous).Seconds()
	clock.previous = time.Now()
	clock.accumulated += math.Min(elapsed, 0.25)
	for clock.accumulated >= clock.step {
		if err := RunUpdateStep(app, clock.step); err != nil {
			return err
		}
		clock.accumulated -= clock.step
		clock.steps++
	}
	return nil
}

// sleepUntilNextStep waits until the next update step is due.
func (clock *Clock) sleepUntilNextStep() {
	elapsed := time.Since(clock.previous).Seconds()
	if wait := clock.step - clock.accumulated - elapsed; wait > 0 {
		time.Sleep(time.Duration(wait * float64(time.Second)))
	}
}

// RunUpdateStep calls the application update once and clears the per-step input deltas.
func RunUpdateStep(app *models.Application, step float64) error {
	if app.Update != nil {
		if err := app.Update(app, step); err != nil {
			return err
		}
	}
	if app.Input != nil {
		app.Input.EndFrame()
	}
	return nil
}

// resizeFollower keeps a window's content in step with its size. While a window is being resized,
// some systems (macOS) hold the program inside glfw.PollEvents until the mouse is let go, so the
// main loop draws nothing and the window shows a stretched or empty frame: it flickers. The
// follower draws from the window's refresh callback meanwhile, after an update step at the new
// size so an interface lays itself out again.
type resizeFollower struct {
	window        *glfw.Window
	app           *models.Application
	clock         *Clock
	width, height int // the framebuffer size the last update step saw
}

// followResizing installs the refresh callback; an error drawing in it lands in failure for the
// main loop to return.
func followResizing(window *glfw.Window, app *models.Application, clock *Clock, drawFrame func() error, failure *error) *resizeFollower {
	follower := &resizeFollower{window: window, app: app, clock: clock}
	window.SetRefreshCallback(func(*glfw.Window) {
		if *failure != nil {
			return
		}
		if err := follower.advance(); err != nil {
			*failure = err
			return
		}
		*failure = drawFrame()
	})
	return follower
}

// advance runs the update steps that are due. When the window changed size and no step is due, it
// runs one anyway, borrowed from the next frame's time, so what is drawn is laid out for the new
// size.
func (follower *resizeFollower) advance() error {
	width, height := follower.window.GetFramebufferSize()
	if follower.app.Input != nil {
		UpdateSizes(follower.window, follower.app.Input)
	}
	if follower.app.DrawOnDemand && follower.app.Input != nil {
		// one step per frame: after a long sleep the clock would otherwise catch up in a burst
		follower.clock.previous, follower.clock.accumulated = time.Now(), 0
		follower.clock.steps++
		follower.width, follower.height = width, height
		return RunUpdateStep(follower.app, follower.clock.step)
	}
	before := follower.clock.steps
	if err := follower.clock.advance(follower.app); err != nil {
		return err
	}
	stepped := follower.clock.steps > before
	if !stepped && (width != follower.width || height != follower.height) {
		if err := RunUpdateStep(follower.app, follower.clock.step); err != nil {
			return err
		}
		follower.clock.accumulated -= follower.clock.step
	}
	follower.width, follower.height = width, height
	return nil
}

// waitForWork sleeps until a frame is needed, for applications that draw on demand: until input
// arrives, until the time of a frame asked for with RequestFrame, or until Wake. A frame that had
// input is followed by one more, so what depends on where things were laid out (the look of the
// control under the pointer) catches up with a layout that just changed. It returns whether the
// next frame is such a follow-up.
func waitForWork(state *input.State, followUp bool) bool {
	delay, wanted := state.TakeFrameRequest()
	switch {
	case followUp || wanted && delay == 0:
		glfw.PollEvents()
	case wanted:
		glfw.WaitEventsTimeout(delay.Seconds())
	default:
		glfw.WaitEvents()
	}
	return state.HasInput()
}

// Wake makes a window that draws on demand draw a frame now: for a program whose data changes
// on another goroutine (a download finishing, a message arriving). Safe from any goroutine.
func Wake() { glfw.PostEmptyEvent() }
