//go:build android

// Package android runs go-renderer programs as Android applications, with no Java and no
// golang.org/x/mobile: the program is a shared library (-buildmode=c-shared) that Android's own
// NativeActivity loads. A little C (activity.c) receives the activity's callbacks and starts
// main; a thread of the program's own holds the OpenGL ES context, waits on an Android looper for
// touches and the activity's news, and draws only when something happens. Its OpenGL ES calls are
// direct (gl.go), which matters: going through x/mobile's bindings, which cross to another thread
// every three calls, was about two thirds of what a 3D view cost on a phone.
package android

/*
#cgo LDFLAGS: -landroid -llog -lEGL -lGLESv2
#include "activity.h"
*/
import "C"

import (
	"os"
	"sync"
	"sync/atomic"

	"github.com/akrck02/go-renderer/android/callmain"
)

// eventKind is what the activity told the program.
type eventKind int

const (
	activityChanged eventKind = iota // started, resumed, paused, stopped, configuration, focus, redraw
	windowCreated
	windowDestroyed
	inputQueueCreated
	inputQueueDestroyed
)

// The activity's changes, as activity.c numbers them.
const (
	changeResumed       = 1
	changePaused        = 2
	changeConfiguration = 5
	changeRedrawNeeded  = 8
	changeContent       = 9 // what the system covers changed: the on-screen keyboard came or went
)

// event is news from the activity for the drawing thread. done, when set, is closed once the
// drawing thread has acted: Android waits for a window or an input queue to be let go before it
// takes it away.
type event struct {
	kind   eventKind
	change int
	window *C.ANativeWindow
	queue  *C.AInputQueue
	done   chan struct{}
}

var (
	events  = make(chan event, 64)
	looper  *C.ALooper // the drawing thread's, once it runs
	running atomic.Bool
	waking  sync.Mutex
)

//export goStartMain
func goStartMain(code C.uintptr_t, folder *C.char) {
	// the Go runtime read the environment when the library was loaded: what C sets now, Go does
	// not see, so the folder is set here
	if folder != nil {
		_ = os.Setenv("TMPDIR", C.GoString(folder))
	}
	sendStandardErrorToLog()
	logInfo("the Go runtime runs; starting main")
	go callmain.Call(uintptr(code))
}

//export goActivityChanged
func goActivityChanged(change C.int) { send(event{kind: activityChanged, change: int(change)}) }

//export goWindowCreated
func goWindowCreated(window *C.ANativeWindow) { send(event{kind: windowCreated, window: window}) }

//export goWindowDestroyed
func goWindowDestroyed(window *C.ANativeWindow) {
	sendAndWait(event{kind: windowDestroyed, window: window})
}

//export goInputQueueCreated
func goInputQueueCreated(queue *C.AInputQueue) { send(event{kind: inputQueueCreated, queue: queue}) }

//export goInputQueueDestroyed
func goInputQueueDestroyed(queue *C.AInputQueue) {
	sendAndWait(event{kind: inputQueueDestroyed, queue: queue})
}

// send hands news to the drawing thread and wakes it.
func send(news event) {
	events <- news
	wake()
}

// sendAndWait hands news over and waits until the drawing thread has acted on it, if it runs:
// before it does, it never took what is being taken away.
func sendAndWait(news event) {
	if !running.Load() {
		return
	}
	news.done = make(chan struct{})
	send(news)
	<-news.done
}

// wake makes the drawing thread look at its news now. Safe from any thread.
func wake() {
	waking.Lock()
	defer waking.Unlock()
	if looper != nil {
		C.ALooper_wake(looper)
	}
}
