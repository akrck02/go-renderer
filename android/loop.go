//go:build android

package android

/*
#include <stdlib.h>
#include "activity.h"
*/
import "C"

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/akrck02/go-renderer/frameloop"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
)

// Options tune the window.
type Options struct {
	// MaxFramesPerSecond caps how often it draws; 0 = 60. Phones refresh at 90 or 120 Hz, and a
	// finger dragging a 3D view would otherwise ask for a frame at every touch event.
	MaxFramesPerSecond int
}

// Insets are what the system's bars cover of each edge, in layout pixels.
type Insets struct{ Top, Bottom, Left, Right float32 }

var (
	insets     atomic.Pointer[Insets]
	systemDark atomic.Bool
	graphics   atomic.Pointer[string]
	lastCrash  string
	wanted     atomic.Bool // a frame was asked for from another goroutine (Wake)
)

// CurrentInsets returns what the system's bars cover, as last read.
func CurrentInsets() Insets {
	if read := insets.Load(); read != nil {
		return *read
	}
	return Insets{}
}

// SystemPrefersDark tells whether the phone is in its dark theme, as last read.
func SystemPrefersDark() bool { return systemDark.Load() }

// Graphics names the phone's OpenGL ES and GPU once the window has drawn, or what stopped it.
func Graphics() string {
	if text := graphics.Load(); text != nil {
		return *text
	}
	return ""
}

// LastCrash is the report of the last run's crash, or "" when it ended well.
func LastCrash() string { return lastCrash }

// Wake makes the window draw a frame now: for a program whose data changes on another goroutine.
// Safe from any goroutine.
func Wake() {
	wanted.Store(true)
	wake()
}

// window is the drawing thread's state.
type window struct {
	app      *models.Application
	draw     func(gl GL, fresh bool) error
	clock    *frameloop.Clock
	reader   touches
	surface  C.surface
	open     bool // the surface is there to draw on
	fresh    bool // the graphics context is new: the renderers are made again
	queue    *C.AInputQueue
	env      *C.JNIEnv
	interval time.Duration // between frames, at most MaxFramesPerSecond
	lastDraw time.Time
	frameAt  time.Time // when a frame asked for with RequestFrame is due; zero = none
	asked    bool      // a frame is wanted as soon as the cap lets it
	insetsOK bool      // the window's own insets were read, not the bars' standard heights
}

// Run draws the application in the activity's window until the program ends (Android ends it).
// draw paints one frame with gl; fresh says the graphics context is new (the first frame, or the
// first after coming back from the background), when renderers must be made again. It must be
// called from main.
func Run(app *models.Application, options Options, draw func(gl GL, fresh bool) error) error {
	keepCrashReports()
	if app.Input == nil {
		app.Input = input.NewState()
	}
	rate := options.MaxFramesPerSecond
	if rate <= 0 {
		rate = 60
	}
	current := &window{app: app, draw: draw, clock: frameloop.NewClock(app.FixedStep), asked: true,
		interval: time.Second / time.Duration(rate)}
	current.reader.state = app.Input
	done := make(chan error, 1)
	go func() { done <- current.run() }()
	return <-done
}

// run is the drawing thread: it waits on its looper for touches and news, and draws.
func (current *window) run() error {
	runtime.LockOSThread() // the graphics context and the looper belong to this thread
	waking.Lock()
	// the input queue is attached with no callback: the looper must accept that, and pollOnce then
	// returns when it has events
	looper = C.ALooper_prepare(C.ALOOPER_PREPARE_ALLOW_NON_CALLBACKS)
	// the activity's news (its window, above all) may have come before there was a looper to wake:
	// the first wait must not sleep on it
	C.ALooper_wake(looper)
	waking.Unlock()
	running.Store(true)
	current.env = C.attach_thread()
	current.readSystem()
	for !current.app.Stop {
		C.ALooper_pollOnce(current.timeout(), nil, nil, nil)
		current.takeNews()
		if current.reader.readQueue(current.queue) {
			current.asked = true
		}
		if wanted.Swap(false) {
			current.asked = true
		}
		if !current.frameAt.IsZero() && !time.Now().Before(current.frameAt) {
			current.frameAt, current.asked = time.Time{}, true
		}
		if err := current.paintWhenDue(); err != nil {
			return err
		}
	}
	return nil
}

// timeout is how long the looper may sleep, in milliseconds: -1 until something happens.
func (current *window) timeout() C.int {
	if !current.open {
		return -1
	}
	now := time.Now()
	if current.asked {
		return milliseconds(current.lastDraw.Add(current.interval).Sub(now))
	}
	if !current.frameAt.IsZero() {
		return milliseconds(current.frameAt.Sub(now))
	}
	return -1
}

func milliseconds(wait time.Duration) C.int {
	if wait <= 0 {
		return 0
	}
	return C.int(math.Ceil(float64(wait) / float64(time.Millisecond)))
}

// takeNews acts on what the activity said.
func (current *window) takeNews() {
	for {
		select {
		case news := <-events:
			current.act(news)
			if news.done != nil {
				close(news.done)
			}
		default:
			return
		}
	}
}

func (current *window) act(news event) {
	switch news.kind {
	case windowCreated:
		current.openSurface(news.window)
	case windowDestroyed:
		current.closeSurface()
	case inputQueueCreated:
		current.queue = news.queue
		C.AInputQueue_attachLooper(news.queue, looper, 1, nil, nil)
	case inputQueueDestroyed:
		C.AInputQueue_detachLooper(news.queue)
		current.queue = nil
	case activityChanged:
		if news.change == changeConfiguration || news.change == changeResumed {
			current.readSystem() // the theme, the density or the bars may have changed
		}
		current.asked = true
	}
}

func (current *window) openSurface(native *C.ANativeWindow) {
	if C.surface_open(native, &current.surface) == 0 {
		failure := "could not make an OpenGL ES context for the window"
		graphics.Store(&failure)
		logError(failure)
		return
	}
	current.open, current.fresh, current.asked, current.insetsOK = true, true, true, false
}

func (current *window) closeSurface() {
	if !current.open {
		return
	}
	C.surface_close(&current.surface)
	current.open = false
}

// readSystem reads the screen's density, the phone's theme and the colours of its wallpaper.
func (current *window) readSystem() {
	density := float32(C.screen_density())
	current.reader.scale = 1
	if density > 0 {
		current.reader.scale = density / 160 // layout pixels are Android's dp, 1/160 inch
	}
	if current.env != nil {
		systemDark.Store(C.night_mode(current.env) != 0)
		readPalettes(current.env)
	}
	current.insetsOK = false
}

// readInsets reads what the system's bars cover until the window's own insets are known; before the
// window is shown Android only knows the bars' standard heights.
func (current *window) readInsets() {
	if current.insetsOK || current.env == nil {
		return
	}
	var top, bottom, left, right C.int
	scale := current.reader.scale
	if C.window_insets(current.env, &top, &bottom, &left, &right) != 0 {
		current.insetsOK = true
	} else {
		top, bottom = C.int(barHeight(current.env, "status_bar_height")), C.int(barHeight(current.env, "navigation_bar_height"))
	}
	insets.Store(&Insets{Top: float32(top) / scale, Bottom: float32(bottom) / scale, Left: float32(left) / scale, Right: float32(right) / scale})
}

func barHeight(env *C.JNIEnv, name string) int {
	text := C.CString(name)
	defer C.free(unsafe.Pointer(text))
	return int(C.bar_height(env, text))
}

// paintWhenDue runs a step and draws, when a frame is wanted and the cap allows one.
func (current *window) paintWhenDue() error {
	if !current.open || !current.asked || time.Since(current.lastDraw) < current.interval {
		return nil
	}
	started := time.Now() // the cap counts from a frame's start, so its work is inside the interval
	current.asked = false
	current.readInsets()
	current.updateSizes()
	state := current.app.Input
	hadInput := state.HasInput()
	if err := current.clock.StepOnce(current.app); err != nil {
		return err
	}
	// a new surface (back from the background) is empty: it is painted even if the frame looks the
	// same as the one shown before it was lost
	if current.fresh || current.app.Unchanged == nil || !current.app.Unchanged(current.app) {
		if err := current.paint(); err != nil {
			return err
		}
		current.lastDraw = started
	}
	current.asked = !current.app.DrawOnDemand || hadInput // a frame with input is followed by one more
	if delay, wantedFrame := state.TakeFrameRequest(); wantedFrame {
		at := time.Now().Add(delay)
		if current.frameAt.IsZero() || at.Before(current.frameAt) {
			current.frameAt = at
		}
	}
	return nil
}

func (current *window) paint() error {
	if current.fresh && Graphics() == "" {
		text := description() + " · ES " + string(rune('0'+current.surface.version))
		graphics.Store(&text)
	}
	if err := current.draw(GL{}, current.fresh); err != nil {
		failure := "could not draw: " + err.Error()
		graphics.Store(&failure)
		return nil // the window stays, and says why it is empty
	}
	current.fresh = false
	C.surface_present(&current.surface)
	return nil
}

// updateSizes tells the input state the framebuffer's size in pixels and the window's in layout
// pixels.
func (current *window) updateSizes() {
	var width, height C.int
	C.surface_size(&current.surface, &width, &height)
	state, scale := current.app.Input, current.reader.scale
	state.FramebufferWidth, state.FramebufferHeight = int(width), int(height)
	state.WindowWidth, state.WindowHeight = int(float32(width)/scale), int(float32(height)/scale)
}

// keepCrashReports makes the Go runtime write a crash's report (with the failing goroutine's stack:
// a shared library starts with no traceback) to the application's own folder, and reads the one the
// last run left. A phone shows a person nothing when an application closes.
func keepCrashReports() {
	folder := os.TempDir() // the app's own folder (goStartMain)
	_ = os.MkdirAll(folder, 0o700)
	path := filepath.Join(folder, "last-crash.txt")
	if previous, err := os.ReadFile(path); err == nil {
		lastCrash = strings.TrimSpace(string(previous))
	}
	file, err := os.Create(path) // empty: a run that ends well leaves nothing
	if err != nil {
		return
	}
	debug.SetTraceback("single")
	_ = debug.SetCrashOutput(file, debug.CrashOptions{})
	_ = file.Close() // the runtime keeps its own copy
}
