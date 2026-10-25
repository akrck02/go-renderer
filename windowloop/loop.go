// Package windowloop runs a GLFW window's main loop for any drawing backend (OpenGL, Metal): the
// window's events, input, drawing on demand, the fixed-step update clock, frames asked for while
// the window is being resized, the pointer's shape and lock, and the whole screen. A window that
// draws on demand and stays idle for three seconds gives the memory the program no longer uses
// back to the system. A backend only has to draw and show a frame.
package windowloop

import (
	"runtime/debug"
	"strconv"
	"time"

	"github.com/akrck02/go-renderer/frameloop"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/window"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// titleInterval is how often the frames per second are written in the title.
const titleInterval = time.Second

// loop is the state of a running window loop.
type loop struct {
	window    *glfw.Window
	app       *window.Application
	clock     *frameloop.Clock
	drawFrame func() error
	resize    *resizeFollower
	failure   error // an error of a frame drawn while resizing, for the loop to return
	onDemand  bool
	followUp  bool // the next frame follows one with input (see waitForWork)

	cursorLocked bool
	shapes       CursorShapes
	fullscreen   Fullscreen
	pad          gamePad
	titleCounter framesCounter
}

// Run calls the application's update steps and drawFrame until the window closes or the
// application stops. drawFrame draws and shows one frame; it may skip one that would look like
// the last (window.Callbacks.Unchanged). It must be called on the main thread.
func Run(glfwWindow *glfw.Window, app *window.Application, clock *frameloop.Clock, drawFrame func() error) error {
	running := &loop{window: glfwWindow, app: app, clock: clock, drawFrame: drawFrame,
		onDemand: app.DrawOnDemand && app.Input != nil, titleCounter: framesCounter{since: time.Now()}}
	running.resize = followResizing(glfwWindow, app, clock, drawFrame, &running.failure)
	for !glfwWindow.ShouldClose() && !app.Stop {
		if err := running.frame(); err != nil {
			return err
		}
	}
	return nil
}

// frame runs one turn of the loop: events, update steps, and a frame when one is due.
func (running *loop) frame() error {
	running.waitForEvents() // events first, so the frame drawn next is for the window as it is now
	if running.failure != nil {
		return running.failure
	}
	if running.app.Input != nil {
		running.pad.read(running.app.Input)
	}
	stepsBefore := running.clock.Steps()
	if err := running.resize.advance(); err != nil {
		return err
	}
	if !frameDue(running.onDemand, stepsBefore, running.clock.Steps()) {
		// the display may run faster than the updates, and vsync alone does not hold the loop
		// back on every system
		running.clock.SleepUntilNextStep()
		return nil
	}
	if err := running.drawFrame(); err != nil {
		return err
	}
	running.applyRequests()
	running.updateTitle()
	return nil
}

// waitForEvents polls the window's events, or, drawing on demand, sleeps until there is work.
func (running *loop) waitForEvents() {
	if running.onDemand {
		running.followUp = waitForWork(running.app.Input, running.followUp)
		return
	}
	glfw.PollEvents()
}

// frameDue tells whether a frame should be drawn: always when drawing on demand (the loop only
// wakes for one), otherwise only when an update step ran, since the frame would repeat the last.
func frameDue(onDemand bool, stepsBefore, stepsAfter int) bool {
	return onDemand || stepsAfter != stepsBefore
}

// applyRequests does what the program asked of the window this frame: the cursor's lock and
// shape, and the whole screen.
func (running *loop) applyRequests() {
	state := running.app.Input
	if state == nil {
		return
	}
	ApplyCursorLock(running.window, state, &running.cursorLocked)
	running.shapes.Apply(running.window, state)
	running.fullscreen.Apply(running.window, state)
}

// updateTitle writes the frames per second in the title once a second, when the application asks
// for it and draws every frame.
func (running *loop) updateTitle() {
	if running.onDemand || !running.app.ShowFramesPerSecond {
		return
	}
	if frames, due := running.titleCounter.count(time.Now()); due {
		running.window.SetTitle(framesTitle(running.app.Title, running.app.Version, frames))
	}
}

// framesCounter counts frames and says, once an interval has passed, how many there were.
type framesCounter struct {
	frames int
	since  time.Time
}

// count adds a frame at now; it returns the frames of the interval when one has passed.
func (counter *framesCounter) count(now time.Time) (frames int, due bool) {
	counter.frames++
	if now.Sub(counter.since) < titleInterval {
		return 0, false
	}
	frames = counter.frames
	counter.frames, counter.since = 0, now
	return frames, true
}

// framesTitle is a window's title with its frames per second.
func framesTitle(title, version string, frames int) string {
	if version != "" {
		title += " " + version
	}
	return title + "   |   " + strconv.Itoa(frames) + " FPS"
}

// resizeFollower keeps a window's content in step with its size. While a window is being resized,
// some systems (macOS) hold the program inside glfw.PollEvents until the mouse is let go, so the
// main loop draws nothing and the window shows a stretched or empty frame: it flickers. The
// follower draws from the window's refresh and size callbacks meanwhile, after an update step at the new
// size so an interface lays itself out again.
type resizeFollower struct {
	framebufferSize func() (width, height int)
	updateSizes     func() // copies the window's sizes into the input state
	app             *window.Application
	clock           *frameloop.Clock
	width, height   int // the framebuffer size the last update step saw
}

// followResizing installs the refresh callback; an error drawing in it lands in failure for the
// main loop to return.
func followResizing(glfwWindow *glfw.Window, app *window.Application, clock *frameloop.Clock, drawFrame func() error, failure *error) *resizeFollower {
	follower := &resizeFollower{framebufferSize: glfwWindow.GetFramebufferSize, app: app, clock: clock}
	if app.Input != nil {
		follower.updateSizes = func() { UpdateSizes(glfwWindow, app.Input) }
	}
	draw := func() {
		if *failure != nil {
			return
		}
		if err := follower.advance(); err != nil {
			*failure = err
			return
		}
		*failure = drawFrame()
	}
	glfwWindow.SetRefreshCallback(func(*glfw.Window) { draw() })
	// macOS asks for a refresh only when AppKit paints the view, which it never does for a view that
	// hosts a Metal layer; the new size does arrive at every step of a resize, so draw then too.
	var previous glfw.FramebufferSizeCallback
	previous = glfwWindow.SetFramebufferSizeCallback(func(resized *glfw.Window, width, height int) {
		afterSizeChange(previous, resized, width, height, draw)
	})
	return follower
}

// afterSizeChange runs the size callback the backend installed (OpenGL's viewport), then draws:
// in that order, so the frame is drawn for the new size.
func afterSizeChange(previous glfw.FramebufferSizeCallback, resized *glfw.Window, width, height int, draw func()) {
	if previous != nil {
		previous(resized, width, height)
	}
	draw()
}

// advance runs the update steps that are due. When the window changed size and no step is due, it
// runs one anyway, borrowed from the next frame's time, so what is drawn is laid out for the new
// size.
func (follower *resizeFollower) advance() error {
	width, height := follower.framebufferSize()
	if follower.updateSizes != nil {
		follower.updateSizes()
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
	delay, wanted := state.Feed().TakeFrameRequest()
	switch {
	case followUp || wanted && delay == 0:
		glfw.PollEvents()
	case wanted:
		glfw.WaitEventsTimeout(delay.Seconds())
	default:
		sleepReleasingMemory(idleWaiter{state: state})
	}
	return state.Feed().HasInput()
}

// idleBeforeRelease is how long a window that draws on demand waits with nothing to do before
// it gives the system back the memory the program no longer uses.
const idleBeforeRelease = 3 * time.Second

// waiter is what sleepReleasingMemory needs from the window: the real one waits for GLFW's
// events, a test's pretends to.
type waiter interface {
	waitAtMost(time.Duration) (slept bool) // slept: nothing woke it before the time was up
	waitForever()
	hadInput() bool
	release()
}

// sleepReleasingMemory sleeps until there is work. A program's memory holds what it no longer
// uses until the collector runs, and the runtime gives freed pages back to the system only
// slowly, so an idle window would keep the memory of its busiest moment. Once the window has had
// nothing to do for idleBeforeRelease, it collects and gives that memory back, then sleeps on:
// once per idle spell, while nobody is waiting for a frame.
func sleepReleasingMemory(window waiter) {
	slept := window.waitAtMost(idleBeforeRelease)
	if !slept || window.hadInput() {
		return // woken by input or Wake: there is a frame to draw
	}
	window.release()
	window.waitForever()
}

// idleWaiter waits for a GLFW window's events and gives memory back through the runtime.
type idleWaiter struct{ state *input.State }

func (idleWaiter) waitAtMost(duration time.Duration) bool {
	started := time.Now()
	glfw.WaitEventsTimeout(duration.Seconds())
	return time.Since(started) >= duration
}
func (idleWaiter) waitForever()          { glfw.WaitEvents() }
func (waiter idleWaiter) hadInput() bool { return waiter.state.Feed().HasInput() }
func (idleWaiter) release()              { debug.FreeOSMemory() }

// Wake makes a window that draws on demand draw a frame now: for a program whose data changes
// on another goroutine (a download finishing, a message arriving). Safe from any goroutine.
func Wake() { glfw.PostEmptyEvent() }
