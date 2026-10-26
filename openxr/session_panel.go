//go:build android

package openxr

/*
#include "bridge.h"
*/
import "C"

import (
	"time"

	"github.com/akrck02/go-renderer/failure"
	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/graphics"
)

// NewPanel makes a panel of width × height pixels, in the eyes' format, hidden until the program
// shows it (Panel.Shown). Make panels once, after Open, on the drawing thread: each is a swapchain
// of the runtime's and framebuffers of the context's, let go of by Close. A session has at most
// MostPanels.
func (session *Session) NewPanel(width, height int) (*Panel, error) {
	if width <= 0 || height <= 0 {
		return nil, failure.Wrap(PanelNotCreated, "a panel needs a size in pixels", nil)
	}
	var index C.int
	result := C.xr_create_panel(session.program, C.int32_t(width), C.int32_t(height), &index)
	if session.program.framebuffer_incomplete != 0 {
		return nil, FramebufferIncomplete
	}
	if err := session.check(PanelNotCreated, "xrCreateSwapchain", result); err != nil {
		return nil, err
	}
	panel := &Panel{width: width, height: height}
	panel.acquire = func() (gles.Framebuffer, error) { return session.acquirePanel(index) }
	session.panels = append(session.panels, panel)
	return panel, nil
}

// acquirePanel takes the frame's image of a panel, while the frame is drawn.
func (session *Session) acquirePanel(index C.int) (gles.Framebuffer, error) {
	if !session.drawing {
		return 0, PanelNotDrawable
	}
	var framebuffer C.GLuint
	if err := session.frameCheck("xrAcquireSwapchainImage", C.xr_acquire_panel(session.program, index, &framebuffer)); err != nil {
		return 0, err
	}
	return gles.Framebuffer(framebuffer), nil
}

// fillQuads puts the shown panels in the frame's quads and says how many; their order is the
// order they were made, the last made over the others.
func (session *Session) fillQuads() int {
	count := 0
	for index, panel := range session.panels {
		if !panel.shows() {
			continue
		}
		session.quads[count] = C.xr_quad{panel: C.int(index), pose: cPose(panel.Pose),
			width: C.float(panel.WidthMetres), height: C.float(panel.HeightMetres)}
		count++
	}
	return count
}

func cPose(pose graphics.Pose) C.XrPosef {
	var turned C.XrPosef
	turned.orientation = C.XrQuaternionf{x: C.float(pose.Orientation.X), y: C.float(pose.Orientation.Y),
		z: C.float(pose.Orientation.Z), w: C.float(pose.Orientation.W)}
	turned.position = C.XrVector3f{x: C.float(pose.Position[0]), y: C.float(pose.Position[1]), z: C.float(pose.Position[2])}
	return turned
}

// Vibrate shakes a hand's controller, amplitude from 0 to 1, for a time: a touch the hand feels
// when something it points at answers. A new vibration replaces the one still going. It does
// nothing while the program does not have the controls.
func (session *Session) Vibrate(hand Side, amplitude float32, duration time.Duration) error {
	if !session.life.focused() {
		return nil
	}
	amplitude = max(0, min(1, amplitude))
	result := C.xr_vibrate(session.program, C.int(hand), C.float(amplitude), C.int64_t(duration.Nanoseconds()))
	return session.check(FrameFailed, "xrApplyHapticFeedback", result)
}
