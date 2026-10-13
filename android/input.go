//go:build android

package android

/*
#include <android/input.h>
#include <android/keycodes.h>
#include "activity.h"
*/
import "C"

import "github.com/akrck02/go-renderer/input"

// touches turns the input queue's events into an input.State: fingers as the pointer, and a
// hardware keyboard's keys and letters.
type touches struct {
	state   *input.State
	pointer input.TouchPointer
	scale   float32   // screen pixels per layout pixel
	env     *C.JNIEnv // the drawing thread's, to ask Android what a key types
	touched bool      // a finger came down since the on-screen keyboard was last asked for
}

// readQueue takes every waiting event from the queue, and tells whether any reached the program.
func (reader *touches) readQueue(queue *C.AInputQueue) bool {
	if queue == nil {
		return false
	}
	any := false
	var event *C.AInputEvent
	for C.AInputQueue_getEvent(queue, &event) >= 0 {
		if C.AInputQueue_preDispatchEvent(queue, event) != 0 {
			continue // the on-screen keyboard took it
		}
		handled := reader.handle(event)
		any = any || handled
		C.AInputQueue_finishEvent(queue, event, boolInt(handled))
	}
	return any
}

func boolInt(value bool) C.int {
	if value {
		return 1
	}
	return 0
}

// handle passes one event on; false leaves it to Android (the back key closes the activity).
func (reader *touches) handle(event *C.AInputEvent) bool {
	switch C.AInputEvent_getType(event) {
	case C.AINPUT_EVENT_TYPE_MOTION:
		reader.motion(event)
		return true
	case C.AINPUT_EVENT_TYPE_KEY:
		return reader.key(event)
	}
	return false
}

// motion follows fingers: the first one down is the pointer.
func (reader *touches) motion(event *C.AInputEvent) {
	action := C.AMotionEvent_getAction(event)
	kind := action & C.AMOTION_EVENT_ACTION_MASK
	index := C.size_t((action & C.AMOTION_EVENT_ACTION_POINTER_INDEX_MASK) >> C.AMOTION_EVENT_ACTION_POINTER_INDEX_SHIFT)
	finger := int64(C.AMotionEvent_getPointerId(event, index))
	x, y := float32(C.AMotionEvent_getX(event, index)), float32(C.AMotionEvent_getY(event, index))
	switch kind {
	case C.AMOTION_EVENT_ACTION_DOWN, C.AMOTION_EVENT_ACTION_POINTER_DOWN:
		reader.pointer.Begin(reader.state, finger, x, y, reader.scale)
		reader.touched = true
	case C.AMOTION_EVENT_ACTION_UP, C.AMOTION_EVENT_ACTION_POINTER_UP:
		reader.pointer.End(reader.state, finger, x, y, reader.scale)
	case C.AMOTION_EVENT_ACTION_CANCEL:
		reader.pointer.Cancel(reader.state)
	case C.AMOTION_EVENT_ACTION_MOVE:
		// a move event carries every finger: the pointer follows its own
		for each := C.size_t(0); each < C.AMotionEvent_getPointerCount(event); each++ {
			reader.pointer.Move(reader.state, int64(C.AMotionEvent_getPointerId(event, each)),
				float32(C.AMotionEvent_getX(event, each)), float32(C.AMotionEvent_getY(event, each)), reader.scale)
		}
	}
}

// key passes a key on, from a hardware keyboard or the on-screen one, with the character it types.
func (reader *touches) key(event *C.AInputEvent) bool {
	code := C.AKeyEvent_getKeyCode(event)
	if code == C.AKEYCODE_BACK {
		return false
	}
	action, meta := C.AKeyEvent_getAction(event), C.AKeyEvent_getMetaState(event)
	pressed := action == C.AKEY_EVENT_ACTION_DOWN
	key, character := keyFromCode(int(code), meta&C.AMETA_SHIFT_ON != 0)
	if pressed {
		character = reader.characterOf(action, code, meta, character)
	}
	if key != input.KeyUnknown {
		if pressed && C.AKeyEvent_getRepeatCount(event) > 0 {
			reader.state.RepeatKey(key)
		} else {
			reader.state.SetKey(key, pressed)
		}
	}
	if pressed && character != 0 {
		reader.state.AddCharacter(character)
	}
	return key != input.KeyUnknown || character != 0
}

// characterOf asks Android what a key types, which knows every layout; what the key code tells is
// kept when it cannot be asked. Control characters (Enter's new line, Tab) are keys, not text.
func (reader *touches) characterOf(action, code, meta C.int32_t, known rune) rune {
	if reader.env == nil {
		return known
	}
	character := rune(C.key_character(reader.env, C.int(action), C.int(code), C.int(meta)))
	if character < ' ' || character == 0x7f {
		return known
	}
	return character
}

// keyFromCode maps an Android key code to an input key and the character it types (letters and
// digits, and a space), with Shift for capitals.
func keyFromCode(code int, shift bool) (input.Key, rune) {
	switch {
	case code >= C.AKEYCODE_A && code <= C.AKEYCODE_Z:
		letter := rune('a' + code - C.AKEYCODE_A)
		if shift {
			letter -= 'a' - 'A'
		}
		return input.KeyA + input.Key(code-C.AKEYCODE_A), letter
	case code >= C.AKEYCODE_0 && code <= C.AKEYCODE_9:
		return input.Key0 + input.Key(code-C.AKEYCODE_0), rune('0' + code - C.AKEYCODE_0)
	}
	keys := map[int]input.Key{
		C.AKEYCODE_DEL: input.KeyBackspace, C.AKEYCODE_FORWARD_DEL: input.KeyDelete,
		C.AKEYCODE_ENTER: input.KeyEnter, C.AKEYCODE_TAB: input.KeyTab, C.AKEYCODE_ESCAPE: input.KeyEscape,
		C.AKEYCODE_DPAD_UP: input.KeyUp, C.AKEYCODE_DPAD_DOWN: input.KeyDown,
		C.AKEYCODE_DPAD_LEFT: input.KeyLeft, C.AKEYCODE_DPAD_RIGHT: input.KeyRight,
		C.AKEYCODE_MOVE_HOME: input.KeyHome, C.AKEYCODE_MOVE_END: input.KeyEnd,
		C.AKEYCODE_SHIFT_LEFT: input.KeyShift, C.AKEYCODE_SHIFT_RIGHT: input.KeyShift,
	}
	if code == C.AKEYCODE_SPACE {
		return input.KeySpace, ' '
	}
	return keys[code], 0
}
