package windowloop

import (
	"testing"

	"github.com/akrck02/go-renderer/input"
	"github.com/go-gl/glfw/v3.4/glfw"
)

func TestAWindowIsLaidOutByTheScaleItsFramebufferDoesNotCarry(test *testing.T) {
	cases := []struct {
		name                          string
		framebufferWidth, windowWidth int
		contentScale                  float32
		want                          float64
	}{
		{"X11 on a screen of twice the density", 2560, 2560, 2, 2},
		{"Wayland or macOS on a screen of twice the density", 2560, 1280, 2, 1},
		{"a screen that is not scaled", 1280, 1280, 1, 1},
		{"X11 at one and a half", 1920, 1920, 1.5, 1.5},
		{"a window not shown yet", 0, 0, 2, 1},
	}
	for _, each := range cases {
		if got := missingScale(each.framebufferWidth, each.windowWidth, each.contentScale); got != each.want {
			test.Errorf("%s: scale %v, want %v", each.name, got, each.want)
		}
	}
}

func TestPlusAndMinusAreKnownByTheCharacterTheyWriteOnAnyLayout(test *testing.T) {
	cases := []struct {
		name string
		key  glfw.Key
		want input.Key
	}{
		{"+ on a Spanish keyboard, where the US layout has ]", glfw.KeyRightBracket, input.KeyPlus},
		{"- on a Spanish keyboard, where the US layout has /", glfw.KeySlash, input.KeyMinus},
		{"= on a US keyboard, which writes + with Shift", glfw.KeyEqual, input.KeyPlus},
		{"the keypad's +, which has no name", glfw.KeyKPAdd, input.KeyPlus},
	}
	names := map[glfw.Key]string{glfw.KeyRightBracket: "+", glfw.KeySlash: "-", glfw.KeyEqual: "="}
	for _, each := range cases {
		if got, known := keyOf(each.key, names[each.key]); !known || got != each.want {
			test.Errorf("%s: read as %v (%v), want %v", each.name, got, known, each.want)
		}
	}
}
