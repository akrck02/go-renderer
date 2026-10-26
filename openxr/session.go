//go:build android

package openxr

/*
#cgo CFLAGS: -I${SRCDIR}/include
#cgo LDFLAGS: -ldl -lEGL -lGLESv3
#include <stdlib.h>
#include "bridge.h"
*/
import "C"

import (
	"log"
	"unsafe"

	"github.com/akrck02/go-renderer/android"
	"github.com/akrck02/go-renderer/failure"
)

// Session is a program's session with a headset's runtime: its spaces, the eyes' swapchains and the
// controllers. Use it only from the drawing thread android.RunHeadset calls the frame function on:
// the swapchains' framebuffers belong to its OpenGL ES context.
type Session struct {
	program    *C.xr_program
	life       lifecycle
	frameBegun bool // between a BeginFrame that began a frame and its EndFrame
	drawing    bool // the begun frame is drawn: its eyes' images (and the panels') can be taken
	panels     []*Panel
	quads      [MostPanels]C.xr_quad // the frame's shown panels, filled in EndFrame without allocating
}

// Open starts the runtime for the program and makes a session for the headset's OpenGL ES context,
// with its spaces, the eyes' swapchains and the controllers. Call it from inside RunHeadset's frame
// function, where the context is current. The session runs once PollEvents hears it is ready.
func Open(headset android.Headset, options Options) (*Session, error) {
	session := &Session{program: C.xr_new_program()}
	if err := session.open(headset, options); err != nil {
		session.Close()
		return nil, err
	}
	return session, nil
}

func (session *Session) open(headset android.Headset, options Options) error {
	if err := session.startRuntime(headset, options); err != nil {
		return err
	}
	if err := session.check(HeadsetNotFound, "xrGetSystem", C.xr_find_headset(session.program)); err != nil {
		return err
	}
	if err := session.checkGraphics(); err != nil {
		return err
	}
	if err := session.createSession(headset); err != nil {
		return err
	}
	return session.check(ControlsNotCreated, "actions", C.xr_create_controls(session.program))
}

// startRuntime opens the loader, hands it the Java VM and the activity, and makes the instance.
func (session *Session) startRuntime(headset android.Headset, options Options) error {
	if problem := C.xr_open_loader(session.program); problem != nil {
		return failure.Wrap(LoaderNotFound, C.GoString(problem), nil)
	}
	javaVM, activity := C.uintptr_t(headset.JavaVM), C.uintptr_t(headset.Activity)
	result := C.xr_start_loader(session.program, javaVM, activity)
	if err := session.check(InstanceNotCreated, "xrInitializeLoaderKHR", result); err != nil {
		return err
	}
	name := C.CString(options.applicationName())
	defer C.free(unsafe.Pointer(name))
	var missing *C.char
	result = C.xr_create_instance(session.program, name, javaVM, activity, &missing)
	if missing != nil {
		return failure.Wrap(ExtensionMissing, C.GoString(missing), nil)
	}
	return session.check(InstanceNotCreated, "xrCreateInstance", result)
}

func (session *Session) checkGraphics() error {
	var supported C.int
	result := C.xr_check_graphics(session.program, &supported)
	if err := session.check(GraphicsUnsupported, "xrGetOpenGLESGraphicsRequirementsKHR", result); err != nil {
		return err
	}
	if supported == 0 {
		return GraphicsUnsupported
	}
	return nil
}

// createSession makes the session, its spaces and the eyes' swapchains.
func (session *Session) createSession(headset android.Headset) error {
	display, config, context := C.uintptr_t(headset.Display), C.uintptr_t(headset.Config), C.uintptr_t(headset.Context)
	result := C.xr_create_session(session.program, display, config, context)
	if err := session.check(SessionNotCreated, "xrCreateSession", result); err != nil {
		return err
	}
	if err := session.check(SessionNotCreated, "xrCreateReferenceSpace", C.xr_create_spaces(session.program)); err != nil {
		return err
	}
	result = C.xr_create_swapchains(session.program)
	if session.program.framebuffer_incomplete != 0 {
		return FramebufferIncomplete
	}
	return session.check(SessionNotCreated, "xrCreateSwapchain", result)
}

// Close ends the session and lets go of the runtime. Call it on the drawing thread, once Finished
// is true, or when the session was lost: a program that wants to leave a running session asks the
// runtime first (RequestExit).
func (session *Session) Close() {
	if session.program == nil {
		return
	}
	C.xr_destroy(session.program)
	session.program = nil
}

// PollEvents takes the runtime's news: the session begins when it is ready and ends when it is
// stopping, and a runtime going away or asking the program to leave makes Exiting true. Call it
// once a frame, before BeginFrame.
func (session *Session) PollEvents() {
	for {
		var kind, state C.int
		result := C.xr_poll_event(session.program, &kind, &state)
		if err := session.check(SessionLost, "xrPollEvent", result); err != nil {
			log.Print(err)
			session.life.lose()
			return
		}
		switch kind {
		case C.xr_event_none:
			return
		case C.xr_event_state:
			session.changeState(sessionState(state))
		case C.xr_event_loss:
			session.life.lose()
		}
	}
}

// changeState does what the runtime's new state asks. A session that cannot begin leaves the
// program nothing to show, so it asks the program to end.
func (session *Session) changeState(state sessionState) {
	switch session.life.change(state) {
	case stepBegin:
		if err := session.check(SessionLost, "xrBeginSession", C.xr_begin_session(session.program)); err != nil {
			log.Print(err)
			session.life.lose()
			return
		}
		session.life.began()
	case stepEnd:
		if err := session.check(SessionLost, "xrEndSession", C.xr_end_session(session.program)); err != nil {
			log.Print(err)
		}
		session.life.ended()
	}
}

// RequestExit asks the runtime to stop the session because the program wants to leave. Destroying
// a running session under the runtime is not how OpenXR ends one: after this, keep calling
// PollEvents and handing back frames (empty ones will do) until Finished, then Close. It fails when
// the session does not run (XR_ERROR_SESSION_NOT_RUNNING); close it straight away then.
func (session *Session) RequestExit() error {
	if err := session.check(SessionLost, "xrRequestExitSession", C.xr_request_exit(session.program)); err != nil {
		return err
	}
	session.life.requestExit()
	return nil
}

// Finished tells whether the session may be closed: the runtime asked the program to end (Exiting),
// or the session RequestExit asked to stop has ended.
func (session *Session) Finished() bool { return session.life.finished() }

// Running tells whether frames are to be submitted: the session began and has not ended.
func (session *Session) Running() bool { return session.life.running }

// Focused tells whether the program has the controls (no system menu is over it).
func (session *Session) Focused() bool { return session.life.focused() }

// Exiting tells whether the runtime asked the program to end, or was lost.
func (session *Session) Exiting() bool { return session.life.exiting }

// SRGB tells whether the swapchains encode sRGB themselves: draw linear colour into them (gles'
// SceneRenderer.LinearOutput), or the picture comes out too bright.
func (session *Session) SRGB() bool { return session.program != nil && session.program.srgb != 0 }

// check turns a failed result into an error of a code that names the call; success is nil.
func (session *Session) check(code *failure.Code, call string, result C.XrResult) error {
	if result >= 0 {
		return nil
	}
	return callError(code, call, session.resultName(result))
}

// resultName is what the runtime calls a result (XR_ERROR_…).
func (session *Session) resultName(result C.XrResult) string {
	var buffer [C.XR_MAX_RESULT_STRING_SIZE]C.char
	C.xr_result_name(session.program, result, &buffer[0])
	return C.GoString(&buffer[0])
}
