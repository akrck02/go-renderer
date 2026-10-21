// Package windowloop runs a GLFW window's main loop for any drawing backend (OpenGL, Metal): the
// window's events, input, drawing on demand, the fixed-step update clock, frames asked for while
// the window is being resized, the pointer's shape and lock. A backend only has to draw and show
// a frame.
package windowloop

import (
	"fmt"
	"time"

	"github.com/akrck02/go-renderer/frameloop"
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
		if app.Input != nil {
			ReadGamePad(app.Input)
		}
		stepsBefore := clock.Steps()
		if err := resize.advance(); err != nil {
			return err
		}
		if !onDemand && clock.Steps() == stepsBefore {
			// nothing was updated, so the frame would repeat the last one: wait for the next
			// step instead (the display may run faster than the updates, and vsync alone does
			// not hold the loop back on every system)
			clock.SleepUntilNextStep()
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

// Clock runs Application.Update at a fixed time step; it lives in frameloop, which the browser
// loop shares.
type Clock = frameloop.Clock

// NewClock makes a clock of step seconds (1/60 when 0).
func NewClock(step float64) *Clock { return frameloop.NewClock(step) }

// RunUpdateStep calls the application update once and clears the per-step input deltas.
func RunUpdateStep(app *models.Application, step float64) error {
	return frameloop.RunUpdateStep(app, step)
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
		follower.width, follower.height = width, height
		return follower.clock.StepOnce(follower.app)
	}
	before := follower.clock.Steps()
	if err := follower.clock.Advance(follower.app); err != nil {
		return err
	}
	stepped := follower.clock.Steps() > before
	if !stepped && (width != follower.width || height != follower.height) {
		if err := follower.clock.BorrowStep(follower.app); err != nil {
			return err
		}
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
