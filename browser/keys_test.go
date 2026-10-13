//go:build js && wasm

package browser

import (
	"testing"

	"github.com/akrck02/go-renderer/input"
)

func TestKeysFollowTheirPlaceOnTheKeyboard(t *testing.T) {
	for code, want := range map[string]input.Key{
		"KeyA": input.KeyA, "KeyZ": input.KeyZ, "Digit0": input.Key0, "Digit9": input.Key9,
		"ArrowLeft": input.KeyLeft, "MetaRight": input.KeySuper, "NumpadEnter": input.KeyEnter,
	} {
		if key, known := keyFromCode(code); !known || key != want {
			t.Errorf("%s = %v, %v; want %v", code, key, known, want)
		}
	}
	if _, known := keyFromCode("F13"); known {
		t.Error("F13 should be unknown")
	}
}

func TestPrintableKeysTypeTheirCharacter(t *testing.T) {
	for key, want := range map[string]rune{"a": 'a', "Ñ": 'Ñ', "€": '€', " ": ' '} {
		if character, typed := typedCharacter(key, false); !typed || character != want {
			t.Errorf("%q typed %q, %v; want %q", key, character, typed, want)
		}
	}
	for _, key := range []string{"Enter", "ArrowUp", "Backspace", "Shift"} {
		if _, typed := typedCharacter(key, false); typed {
			t.Errorf("%q typed a character", key)
		}
	}
	if _, typed := typedCharacter("c", true); typed {
		t.Error("a shortcut typed a character")
	}
}

func TestTheWheelScrollsInLinesUpPositive(t *testing.T) {
	if lines := wheelLines(100, 0); lines != -1 {
		t.Errorf("a notch down in pixels = %v lines, want -1", lines)
	}
	if lines := wheelLines(-3, 1); lines != 3 {
		t.Errorf("three lines up = %v, want 3", lines)
	}
}

func TestPointerButtons(t *testing.T) {
	if button, known := mouseButtonFrom(2); !known || button != input.MouseRight {
		t.Errorf("button 2 = %v, %v; want the right button", button, known)
	}
	if _, known := mouseButtonFrom(4); known {
		t.Error("button 4 should be unknown")
	}
}
