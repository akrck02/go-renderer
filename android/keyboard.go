//go:build android

package android

/*
#include "activity.h"
*/
import "C"

// keyboardShown is whether the on-screen keyboard was last asked for. Only the drawing thread
// touches it: ShowKeyboard is called from a program's update step, which runs there.
var keyboardShown bool

// ShowKeyboard asks for the on-screen keyboard while a text field is being typed in, and puts it
// away when none is: call it in every step (go-ui's window does, with Context.Typing). A finger
// coming down while typing asks again: the person may have put the keyboard away with its own
// button, and touched the field to have it back.
func ShowKeyboard(typing bool) {
	if drawing == nil {
		return
	}
	asked := typing && (!keyboardShown || drawing.reader.touched)
	drawing.reader.touched = false
	if asked {
		C.soft_keyboard(1)
	} else if !typing && keyboardShown {
		C.soft_keyboard(0)
	}
	keyboardShown = typing
}
