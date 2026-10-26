//go:build android

package android

/*
#cgo LDFLAGS: -lGLESv3
#include "activity.h"
*/
import "C"

import (
	"runtime"

	"github.com/akrck02/go-renderer/failure"
)

// HeadsetContextNotMade is a headset whose drawing thread could not get an OpenGL ES 3 context.
var HeadsetContextNotMade = failure.Define("android.headset-context-not-made",
	"could not make an OpenGL ES 3 context for the headset")

// Headset is what an OpenXR runtime needs from the program: the Java VM, the activity, and the EGL
// context its drawing thread holds (JavaVM*, jobject, EGLDisplay, EGLConfig, EGLContext), handed
// over as they are for the package that speaks to the runtime. They are numbers, not Go pointers:
// a jobject global reference is an encoded index and an EGLDisplay is often 1, which the garbage
// collector must never take for an address.
type Headset struct {
	JavaVM, Activity, Display, Config, Context uintptr
}

// HeadsetOptions tune RunHeadset.
type HeadsetOptions struct {
	// Hidden is called on the drawing thread when the activity is paused (the headset taken off, the
	// system's menu over the program), Shown when it is resumed, the first time included.
	Hidden, Shown func()
}

// headsetThread is RunHeadset's drawing thread's state.
type headsetThread struct {
	options HeadsetOptions
	frame   func(headset Headset, functions GL) bool
	context C.headset_context
	env     *C.JNIEnv
	handles C.headset_handles
	headset Headset
	queue   *C.AInputQueue
}

// RunHeadset runs a program that draws for a headset rather than into a window: an OpenXR runtime
// takes its pictures from images of its own. A thread of its own holds an OpenGL ES 3 context, current
// on a small pbuffer, and calls frame again and again until it returns false; frame paces itself
// (OpenXR's xrWaitFrame, or a short sleep while the session does not run). The activity's news is
// still read, and its windows and input queue let go of as Android requires. When frame returns
// false the context is let go and the activity finished. It must be called from main.
func RunHeadset(options HeadsetOptions, frame func(headset Headset, functions GL) bool) error {
	keepCrashReports()
	current := &headsetThread{options: options, frame: frame}
	done := make(chan error, 1)
	go func() { done <- current.run() }()
	return <-done
}

func (current *headsetThread) run() error {
	runtime.LockOSThread() // the graphics context, the looper and the runtime's session belong to it
	current.prepareLooper()
	defer current.end()
	current.env = C.attach_thread()
	if C.headset_context_open(&current.context) == 0 {
		return failure.Wrap(HeadsetContextNotMade, "EGL refused a context", nil)
	}
	C.headset_handles_fill(current.env, &current.context, &current.handles)
	handles := current.handles
	current.headset = Headset{
		JavaVM: uintptr(handles.java_vm), Activity: uintptr(handles.activity),
		Display: uintptr(handles.display), Config: uintptr(handles.config), Context: uintptr(handles.context),
	}
	logInfo("the headset's drawing thread holds an OpenGL ES 3 context")
	for {
		// never sleeps here: frame waits on the runtime, and the news must not hold it up
		C.ALooper_pollOnce(0, nil, nil, nil)
		current.takeNews()
		if current.queue != nil {
			C.headset_drain_queue(current.queue)
		}
		if !current.frame(current.headset, GL{}) {
			return nil
		}
	}
}

// prepareLooper makes the thread's looper, as Run's thread does: the input queue is attached with no
// callback, and news that came before the looper is read at the first poll.
func (current *headsetThread) prepareLooper() {
	waking.Lock()
	defer waking.Unlock()
	looper = C.ALooper_prepare(C.ALOOPER_PREPARE_ALLOW_NON_CALLBACKS)
	C.ALooper_wake(looper)
	running.Store(true)
}

// takeNews acts on what the activity said. There is no window to draw on: a window is only let go
// when Android takes it away.
func (current *headsetThread) takeNews() {
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

func (current *headsetThread) act(news event) {
	switch news.kind {
	case inputQueueCreated:
		current.queue = news.queue
		C.AInputQueue_attachLooper(news.queue, looper, 1, nil, nil)
	case inputQueueDestroyed:
		C.AInputQueue_detachLooper(news.queue)
		current.queue = nil
	case activityChanged:
		current.changed(news.change)
	}
}

func (current *headsetThread) changed(change int) {
	if change == changePaused && current.options.Hidden != nil {
		current.options.Hidden()
	}
	if change == changeResumed && current.options.Shown != nil {
		current.options.Shown()
	}
}

// end lets go of what the thread held, finishes the activity, and keeps answering the activity's
// news: Android's interface thread waits on it when it takes a window or a queue away.
func (current *headsetThread) end() {
	if current.queue != nil {
		C.AInputQueue_detachLooper(current.queue)
		current.queue = nil
	}
	C.headset_context_close(&current.context)
	C.headset_handles_release(current.env, &current.handles)
	if current.env != nil {
		C.detach_thread() // a thread that ends attached to Java stops the whole program
	}
	waking.Lock()
	looper = nil // it goes with the thread
	waking.Unlock()
	C.finish_activity()
	go answerNewsForever()
	logInfo("the headset's drawing thread ended")
}

// answerNewsForever lets the activity's callbacks return once nothing draws any more.
func answerNewsForever() {
	for news := range events {
		if news.done != nil {
			close(news.done)
		}
	}
}
