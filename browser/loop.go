//go:build js && wasm

package browser

import (
	"sync"
	"syscall/js"
	"time"

	"github.com/akrck02/go-renderer/frameloop"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/window"
)

// Run drives an application in the canvas until it sets Stop: update steps at the fixed step and
// drawFrame to draw one, called from requestAnimationFrame. With DrawOnDemand it asks for a frame
// only when something happens (an event, a size change, Input.RequestFrame, Wake) and the page
// otherwise does nothing; without it, it draws after every update step, as on the desktop.
// Frames that Application.Unchanged says would repeat the last one are not drawn.
//
// It blocks the calling goroutine, which keeps a WebAssembly program alive (main must not
// return while the page shows it).
func Run(canvas *Canvas, app *window.Application, drawFrame func() error) error {
	if app.Input == nil {
		app.Input = input.NewState()
	}
	loop := &frameLoop{canvas: canvas, app: app, drawFrame: drawFrame, clock: frameloop.NewClock(app.FixedStepSeconds),
		onDemand: app.DrawOnDemand, done: make(chan error, 1)}
	canvas.Attach(app.Input)
	canvas.changed = loop.request
	loop.frameCallback = js.FuncOf(func(js.Value, []js.Value) any {
		loop.frame()
		return nil
	})
	loop.timerCallback = js.FuncOf(func(js.Value, []js.Value) any {
		loop.timerPending = false
		loop.request()
		return nil
	})
	running.Lock()
	current = loop
	running.Unlock()
	loop.request()
	return <-loop.done
}

// Stop ends the running program's loop after one more frame, so that Run returns: for a page that
// takes the canvas away (an application of the page's own leaving the screen). Its listeners are
// taken off the page. Safe from any goroutine.
func Stop() {
	running.Lock()
	loop := current
	running.Unlock()
	if loop == nil {
		return
	}
	loop.app.Stop = true
	loop.request()
}

// Wake makes a canvas that draws on demand draw a frame now: for a program whose data changes on
// another goroutine (a fetch finishing). Safe from any goroutine.
func Wake() {
	running.Lock()
	loop := current
	running.Unlock()
	if loop != nil {
		loop.request()
	}
}

var (
	running sync.Mutex
	current *frameLoop
)

// frameLoop is the state of Run between animation frames.
type frameLoop struct {
	canvas        *Canvas
	app           *window.Application
	drawFrame     func() error
	clock         *frameloop.Clock
	onDemand      bool
	frameCallback js.Func
	timerCallback js.Func
	done          chan error

	mutex         sync.Mutex
	scheduled     bool // an animation frame is already asked for
	timerPending  bool // a timer for a frame asked with RequestFrame is running
	timerDeadline time.Time
	timer         js.Value
	stopped       bool
}

// request asks the browser for an animation frame, once however many times it is called before
// the frame comes.
func (loop *frameLoop) request() {
	loop.mutex.Lock()
	defer loop.mutex.Unlock()
	if loop.scheduled || loop.stopped {
		return
	}
	loop.scheduled = true
	js.Global().Call("requestAnimationFrame", loop.frameCallback)
}

// frame runs in an animation frame: the update steps due, then drawing.
func (loop *frameLoop) frame() {
	loop.mutex.Lock()
	loop.scheduled = false
	loop.mutex.Unlock()
	hadInput := loop.app.Input.Feed().HasInput()
	stepped, err := loop.step()
	if err != nil {
		loop.finish(err)
		return
	}
	if stepped && !(loop.app.Unchanged != nil && loop.app.Unchanged(loop.app)) {
		if err := loop.drawFrame(); err != nil {
			loop.finish(err)
			return
		}
	}
	loop.canvas.applyCursor()
	loop.canvas.applyFullscreen(false)
	if loop.app.Stop {
		loop.finish(nil)
		return
	}
	loop.scheduleNext(hadInput)
}

// step runs one update step when drawing on demand, or the steps the elapsed time requires, and
// tells whether any ran.
func (loop *frameLoop) step() (bool, error) {
	if loop.onDemand {
		return true, loop.clock.StepOnce(loop.app)
	}
	before := loop.clock.Steps()
	err := loop.clock.Advance(loop.app)
	return loop.clock.Steps() > before, err
}

// scheduleNext decides when the next frame comes. Continuously, at the next animation frame. On
// demand: right away after a frame with input (what depends on the new layout, like the look of
// the control under the pointer, catches up), at the time a frame was asked for, or when the next
// event arrives.
func (loop *frameLoop) scheduleNext(hadInput bool) {
	if !loop.onDemand || hadInput {
		loop.request()
		return
	}
	delay, wanted := loop.app.Input.Feed().TakeFrameRequest()
	if !wanted {
		return
	}
	if delay <= 0 {
		loop.request()
		return
	}
	loop.requestAfter(delay)
}

// requestAfter asks for a frame after a delay (a caret's next blink), keeping one timer: the
// one that fires first.
func (loop *frameLoop) requestAfter(delay time.Duration) {
	deadline := time.Now().Add(delay)
	if loop.timerPending && !deadline.Before(loop.timerDeadline) {
		return
	}
	if loop.timerPending {
		js.Global().Call("clearTimeout", loop.timer)
	}
	loop.timerPending, loop.timerDeadline = true, deadline
	loop.timer = js.Global().Call("setTimeout", loop.timerCallback, delay.Milliseconds())
}

// finish ends the loop, from inside its last frame (no other frame is asked for): it lets go of the
// page, a timer still pending and its own callbacks.
func (loop *frameLoop) finish(err error) {
	loop.mutex.Lock()
	loop.stopped = true
	loop.mutex.Unlock()
	if loop.timerPending {
		js.Global().Call("clearTimeout", loop.timer)
		loop.timerPending = false
	}
	loop.canvas.Detach()
	loop.frameCallback.Release()
	loop.timerCallback.Release()
	running.Lock()
	if current == loop {
		current = nil
	}
	running.Unlock()
	loop.done <- err
}
