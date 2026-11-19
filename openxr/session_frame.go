//go:build android

package openxr

/*
#include "bridge.h"
*/
import "C"

import (
	"log"
	"time"

	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/graphics"
)

// idlePause is how long BeginFrame waits while the session does not run: there is no xrWaitFrame
// to pace the program then, and polling the runtime's events in a tight loop would spend a core.
const idlePause = 10 * time.Millisecond

// BeginFrame waits for the runtime's next frame, begins it, finds the head and the eyes, and takes
// an image of each eye to draw into. The head is found even when ShouldRender is false. When
// ShouldRender is false, draw nothing; either way, give the frame back with EndFrame. While the
// session does not run it waits a little and returns an empty frame, which EndFrame ignores.
func (session *Session) BeginFrame() (Frame, error) {
	if !session.life.running {
		time.Sleep(idlePause)
		return emptyFrame(0), nil
	}
	var shouldRender C.int
	var displayTime C.int64_t
	if err := session.frameCheck("xrWaitFrame", C.xr_wait_frame(session.program, &shouldRender, &displayTime)); err != nil {
		return emptyFrame(0), err
	}
	if err := session.frameCheck("xrBeginFrame", C.xr_begin_frame(session.program)); err != nil {
		return emptyFrame(0), err
	}
	session.frameBegun = true
	frame := emptyFrame(int64(displayTime))
	if err := session.prepare(&frame, shouldRender != 0); err != nil {
		session.EndFrame(emptyFrame(frame.DisplayTime)) // a begun frame must end, here with nothing
		return emptyFrame(0), err
	}
	return frame, nil
}

// prepare finds the head of a begun frame and, when the runtime will show it, the eyes.
func (session *Session) prepare(frame *Frame, shouldRender bool) error {
	if err := session.locateHead(frame); err != nil {
		return err
	}
	if !shouldRender {
		return nil
	}
	return session.prepareEyes(frame)
}

// locateHead finds the head at the frame's display time.
func (session *Session) locateHead(frame *Frame) error {
	var tracked C.int
	var head C.XrPosef
	if err := session.frameCheck("xrLocateSpace", C.xr_locate_head(session.program, C.int64_t(frame.DisplayTime), &tracked, &head)); err != nil {
		return err
	}
	frame.Head = poseOf(head)
	frame.HeadTracked = tracked != 0
	return nil
}

// prepareEyes locates the eyes and acquires their images. A frame whose eyes the runtime cannot
// place is left not to be drawn.
func (session *Session) prepareEyes(frame *Frame) error {
	var located C.int
	if err := session.frameCheck("xrLocateViews", C.xr_locate_eyes(session.program, C.int64_t(frame.DisplayTime), &located)); err != nil {
		return err
	}
	if located == 0 {
		return nil
	}
	var framebuffers [2]C.GLuint
	if err := session.frameCheck("xrAcquireSwapchainImage", C.xr_acquire_images(session.program, &framebuffers[0])); err != nil {
		return err
	}
	for index := range frame.Eyes {
		frame.Eyes[index] = session.eye(index, framebuffers[index])
	}
	frame.ShouldRender = true
	session.drawing = true
	return nil
}

// eye is one located eye with the framebuffer of the image acquired for it.
func (session *Session) eye(index int, framebuffer C.GLuint) Eye {
	view := session.program.views[index]
	swapchain := session.program.eyes[index]
	field := view.fov
	return Eye{
		Pose:   poseOf(view.pose),
		Field:  fieldFrom(float32(field.angleLeft), float32(field.angleRight), float32(field.angleUp), float32(field.angleDown)),
		Target: gles.Framebuffer(framebuffer),
		Width:  int(swapchain.width),
		Height: int(swapchain.height),
	}
}

// EndFrame gives the eyes' images back and ends the frame, showing them as a projection layer when
// the frame was to be drawn. A frame BeginFrame did not begin is ignored.
func (session *Session) EndFrame(frame Frame) error {
	if !session.frameBegun {
		return nil
	}
	session.frameBegun, session.drawing = false, false
	render := C.int(0)
	if frame.ShouldRender {
		render = 1
	}
	count := session.fillQuads()
	result := C.xr_end_frame(session.program, C.int64_t(frame.DisplayTime), render, &session.quads[0], C.int(count))
	return session.frameCheck("xrEndFrame", result)
}

// ReadControls reads both controllers at the frame's display time, so a hand drawn in the frame is
// where the hand will be when it is shown. It reads nothing (an empty Controls) while the program
// does not have the controls or the frame was not begun.
func (session *Session) ReadControls(frame Frame) Controls {
	if !session.life.focused() || frame.DisplayTime == 0 {
		return Controls{}
	}
	var hands [2]C.xr_hand
	var menu C.int
	result := C.xr_read_controls(session.program, C.int64_t(frame.DisplayTime), &hands[0], &menu)
	if err := session.check(FrameFailed, "xrSyncActions", result); err != nil {
		log.Print(err)
		return Controls{}
	}
	return Controls{Left: handOf(hands[0]), Right: handOf(hands[1]), Menu: menu != 0}
}

func handOf(hand C.xr_hand) Hand {
	return Hand{
		Grip:       poseOf(hand.grip),
		Aim:        poseOf(hand.aim),
		Tracked:    hand.tracked != 0,
		Trigger:    float32(hand.trigger),
		Squeeze:    float32(hand.squeeze),
		StickX:     float32(hand.stick_x),
		StickY:     float32(hand.stick_y),
		StickClick: hand.stick_click != 0,
		Upper:      hand.upper != 0,
		Lower:      hand.lower != 0,
	}
}

func poseOf(pose C.XrPosef) graphics.Pose {
	orientation, position := pose.orientation, pose.position
	return poseFrom(
		[4]float32{float32(orientation.x), float32(orientation.y), float32(orientation.z), float32(orientation.w)},
		[3]float32{float32(position.x), float32(position.y), float32(position.z)},
	)
}

// frameCheck is check for the frame's calls: a lost session also asks the program to end.
func (session *Session) frameCheck(call string, result C.XrResult) error {
	if result == C.XR_ERROR_SESSION_LOST {
		session.life.lose()
		return callError(SessionLost, call, session.resultName(result))
	}
	return session.check(FrameFailed, call, result)
}
