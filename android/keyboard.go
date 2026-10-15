//go:build android

package android

/*
#include "activity.h"
*/
import "C"

import "unsafe"

// keyboardShown is whether the on-screen keyboard was last asked for. Only the drawing thread
// touches it: ShowKeyboard is called from a program's update step, which runs there.
var keyboardShown bool

// keyboard is the field's text as it goes between the program and the keyboard's view.
var keyboard keyboardText

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
	if !typing {
		keyboard.forget()
	}
	if asked || (!typing && keyboardShown) {
		showKeyboard(typing)
	}
	keyboardShown = typing
}

// showKeyboard asks the keyboard's view, or, in a package without it, the window.
func showKeyboard(typing bool) {
	if C.text_input_show(boolInt(typing)) != 0 {
		return
	}
	C.soft_keyboard(boolInt(typing))
}

// EditText tells the on-screen keyboard the text of the field being typed in, so it edits that
// text: it suggests words from it, composes in it and deletes from it. Call it in every step while
// a field is typed in, before ShowKeyboard; field tells fields apart (any number but 0), and secret
// asks for a password's keyboard, with no suggestions. It costs nothing while the text is unchanged.
func EditText(field uint64, text TextInput, secret bool) {
	if drawing == nil || field == 0 || !keyboard.toSend(field, text, secret) {
		return
	}
	characters, selectionStart, selectionEnd, composingStart, composingEnd := utf16Text(text)
	C.text_input_set(C.jlong(field), C.int(keyboard.takenEdit), (*C.jchar)(first(characters)), C.int(len(characters)),
		C.int(selectionStart), C.int(selectionEnd), C.int(composingStart), C.int(composingEnd), boolInt(secret))
}

// TakeTextEdit returns what the on-screen keyboard made of a field's text since the last call: its
// whole text, the selection and the word being composed. The program puts it in the field.
func TakeTextEdit(field uint64) (TextInput, bool) {
	return keyboard.take(field)
}

//export goTextEdited
func goTextEdited(field C.jlong, number C.int, characters *C.jchar, length, selectionStart, selectionEnd, composingStart, composingEnd C.int) {
	units := unsafe.Slice((*uint16)(unsafe.Pointer(characters)), int(length))
	edited := fromUTF16(units, int(selectionStart), int(selectionEnd), int(composingStart), int(composingEnd))
	keyboard.keep(uint64(field), int32(number), edited)
	Wake()
}
