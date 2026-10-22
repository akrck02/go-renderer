//go:build js && wasm

package browser

import (
	"testing"

	"github.com/akrck02/go-renderer/input"
)

func TestKeysFollowTheirPlaceOnTheKeyboard(test *testing.T) {
	for code, want := range map[string]input.Key{
		"KeyA": input.KeyA, "KeyZ": input.KeyZ, "Digit0": input.Key0, "Digit9": input.Key9,
		"ArrowLeft": input.KeyLeft, "MetaRight": input.KeySuper, "NumpadEnter": input.KeyEnter,
	} {
		if key, known := keyFromCode(code); !known || key != want {
			test.Errorf("%s = %v, %v; want %v", code, key, known, want)
		}
	}
	if _, known := keyFromCode("F13"); known {
		test.Error("F13 should be unknown")
	}
}

func TestPrintableKeysTypeTheirCharacter(test *testing.T) {
	for key, want := range map[string]rune{"a": 'a', "Ñ": 'Ñ', "€": '€', " ": ' '} {
		if character, typed := typedCharacter(key, false); !typed || character != want {
			test.Errorf("%q typed %q, %v; want %q", key, character, typed, want)
		}
	}
	for _, key := range []string{"Enter", "ArrowUp", "Backspace", "Shift"} {
		if _, typed := typedCharacter(key, false); typed {
			test.Errorf("%q typed a character", key)
		}
	}
	if _, typed := typedCharacter("c", true); typed {
		test.Error("a shortcut typed a character")
	}
}

func TestTheWheelScrollsInLinesUpPositive(test *testing.T) {
	if lines := wheelLines(100, 0); lines != -1 {
		test.Errorf("a notch down in pixels = %v lines, want -1", lines)
	}
	if lines := wheelLines(-3, 1); lines != 3 {
		test.Errorf("three lines up = %v, want 3", lines)
	}
}

func TestPointerButtons(test *testing.T) {
	if button, known := mouseButtonFrom(2); !known || button != input.MouseRight {
		test.Errorf("button 2 = %v, %v; want the right button", button, known)
	}
	if _, known := mouseButtonFrom(4); known {
		test.Error("button 4 should be unknown")
	}
}
