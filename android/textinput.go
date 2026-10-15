package android

import (
	"sync"
	"unicode/utf16"
)

// TextInput is the text of the field being typed in, as the on-screen keyboard sees and edits it.
// Positions count characters (runes); a composing word is the one the keyboard is still making (a
// prediction, a Chinese or Japanese reading), shown underlined until the keyboard commits it.
type TextInput struct {
	Text                         string
	SelectionStart, SelectionEnd int // the caret when they are equal
	ComposingStart, ComposingEnd int // equal when nothing is being composed
}

// keyboardText is what goes between the program and the keyboard's view (TextInput.java). The
// keyboard's edits arrive on Android's interface thread; the program takes them and sends its own
// text on the drawing thread.
type keyboardText struct {
	sync.Mutex
	edited      TextInput // the keyboard's latest edit, not yet taken
	editedField uint64
	editNumber  int32
	waiting     bool

	// what the view has, as far as the drawing thread knows: the program's text is sent only when it
	// differs
	sentField  uint64
	sent       TextInput
	sentSecret bool
	takenEdit  int32 // the keyboard's last edit taken, which the program's text comes after
}

// keep holds an edit of the keyboard's until the program takes it; a newer one replaces it.
func (keyboard *keyboardText) keep(field uint64, number int32, edited TextInput) {
	keyboard.Lock()
	defer keyboard.Unlock()
	keyboard.edited, keyboard.editedField, keyboard.editNumber, keyboard.waiting = edited, field, number, true
}

// take returns the keyboard's latest edit of a field, if there is one. An edit of a field no longer
// typed in is dropped.
func (keyboard *keyboardText) take(field uint64) (TextInput, bool) {
	keyboard.Lock()
	defer keyboard.Unlock()
	if !keyboard.waiting {
		return TextInput{}, false
	}
	keyboard.waiting = false
	if keyboard.editedField != field || field != keyboard.sentField {
		return TextInput{}, false
	}
	keyboard.sent, keyboard.takenEdit = keyboard.edited, keyboard.editNumber
	return keyboard.edited, true
}

// toSend tells whether the program's text differs from what the view has, and remembers it as sent.
func (keyboard *keyboardText) toSend(field uint64, text TextInput, secret bool) bool {
	if field == keyboard.sentField && text == keyboard.sent && secret == keyboard.sentSecret {
		return false
	}
	keyboard.sentField, keyboard.sent, keyboard.sentSecret = field, text, secret
	return true
}

// forget is called when nothing is typed in: the next field is sent whole.
func (keyboard *keyboardText) forget() {
	keyboard.sentField, keyboard.sent = 0, TextInput{}
}

// utf16Text is a text in UTF-16 with its positions in UTF-16 units, as Java counts them.
func utf16Text(text TextInput) (characters []uint16, selectionStart, selectionEnd, composingStart, composingEnd int) {
	runes := []rune(text.Text)
	unitsBefore := func(position int) int {
		position = min(max(position, 0), len(runes))
		units := 0
		for _, character := range runes[:position] {
			units += utf16.RuneLen(character)
		}
		return units
	}
	composingStart, composingEnd = -1, -1
	if text.ComposingEnd > text.ComposingStart {
		composingStart, composingEnd = unitsBefore(text.ComposingStart), unitsBefore(text.ComposingEnd)
	}
	return utf16.Encode(runes), unitsBefore(text.SelectionStart), unitsBefore(text.SelectionEnd), composingStart, composingEnd
}

// fromUTF16 is a text Java sent, with its positions turned from UTF-16 units into characters. A
// composing span of -1 is none.
func fromUTF16(characters []uint16, selectionStart, selectionEnd, composingStart, composingEnd int) TextInput {
	runesBefore := func(units int) int {
		units = min(max(units, 0), len(characters))
		return len(utf16.Decode(characters[:units]))
	}
	text := TextInput{Text: textFrom(characters), SelectionStart: runesBefore(selectionStart), SelectionEnd: runesBefore(selectionEnd)}
	if composingStart >= 0 && composingEnd > composingStart {
		text.ComposingStart, text.ComposingEnd = runesBefore(composingStart), runesBefore(composingEnd)
	}
	return text
}
