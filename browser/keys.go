//go:build js && wasm

package browser

import (
	"unicode/utf8"

	"github.com/akrck02/go-renderer/input"
)

// keysFromCodes maps KeyboardEvent.code (the key's place on the keyboard, whatever its layout
// prints on it) to input keys, as GLFW's key codes do on the desktop.
var keysFromCodes = map[string]input.Key{
	"ArrowUp": input.KeyUp, "ArrowDown": input.KeyDown, "ArrowLeft": input.KeyLeft, "ArrowRight": input.KeyRight,
	"Space": input.KeySpace, "Tab": input.KeyTab, "Enter": input.KeyEnter, "NumpadEnter": input.KeyEnter,
	"Escape":    input.KeyEscape,
	"ShiftLeft": input.KeyShift, "ShiftRight": input.KeyShift,
	"ControlLeft": input.KeyControl, "ControlRight": input.KeyControl,
	"AltLeft": input.KeyAlt, "AltRight": input.KeyAlt,
	"MetaLeft": input.KeySuper, "MetaRight": input.KeySuper,
	"Equal": input.KeyPlus, "NumpadAdd": input.KeyPlus, "Minus": input.KeyMinus, "NumpadSubtract": input.KeyMinus,
	"PageUp": input.KeyPageUp, "PageDown": input.KeyPageDown,
	"BracketLeft": input.KeyLeftBracket, "BracketRight": input.KeyRightBracket,
	"Comma": input.KeyComma, "Period": input.KeyPeriod,
	"Backspace": input.KeyBackspace, "Delete": input.KeyDelete,
	"Home": input.KeyHome, "End": input.KeyEnd,
}

func init() {
	letters := []input.Key{input.KeyA, input.KeyB, input.KeyC, input.KeyD, input.KeyE, input.KeyF, input.KeyG,
		input.KeyH, input.KeyI, input.KeyJ, input.KeyK, input.KeyL, input.KeyM, input.KeyN, input.KeyO, input.KeyP,
		input.KeyQ, input.KeyR, input.KeyS, input.KeyT, input.KeyU, input.KeyV, input.KeyW, input.KeyX, input.KeyY,
		input.KeyZ}
	for index, key := range letters {
		keysFromCodes["Key"+string(rune('A'+index))] = key
	}
	digits := []input.Key{input.Key0, input.Key1, input.Key2, input.Key3, input.Key4, input.Key5, input.Key6,
		input.Key7, input.Key8, input.Key9}
	for index, key := range digits {
		keysFromCodes["Digit"+string(rune('0'+index))] = key
	}
}

// keyFromCode returns the input key for a KeyboardEvent.code, if it is one the input knows.
func keyFromCode(code string) (input.Key, bool) {
	key, known := keysFromCodes[code]
	return key, known
}

// typedCharacter returns the character a key press types: KeyboardEvent.key is the character
// itself for printable keys ("a", "Ñ", "€") and a name for the others ("Enter", "ArrowUp"). A press
// with Control or Command held is a shortcut, not typing.
func typedCharacter(key string, shortcut bool) (rune, bool) {
	if shortcut || utf8.RuneCountInString(key) != 1 {
		return 0, false
	}
	character, _ := utf8.DecodeRuneInString(key)
	return character, character >= ' '
}

// mouseButtonFrom maps PointerEvent.button (0 main, 1 middle, 2 secondary) to an input button.
func mouseButtonFrom(button int) (input.MouseButton, bool) {
	switch button {
	case 0:
		return input.MouseLeft, true
	case 1:
		return input.MouseMiddle, true
	case 2:
		return input.MouseRight, true
	}
	return 0, false
}

// wheelLines turns a WheelEvent's delta into lines scrolled, the unit GLFW reports, with up
// positive as there. deltaMode 0 counts pixels (about 100 for a notch of a mouse wheel), 1 lines,
// 2 pages.
func wheelLines(delta float64, deltaMode int) float64 {
	switch deltaMode {
	case 1:
		return -delta
	case 2:
		return -delta * 10
	}
	return -delta / 100
}

// cursorStyles are the CSS cursors for the input's pointer shapes.
var cursorStyles = map[input.CursorShape]string{
	input.CursorArrow:            "default",
	input.CursorText:             "text",
	input.CursorHand:             "pointer",
	input.CursorResizeHorizontal: "ew-resize",
	input.CursorResizeVertical:   "ns-resize",
	input.CursorMove:             "move",
}
