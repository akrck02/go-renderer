package openxr

import (
	"testing"

	"github.com/akrck02/go-renderer/input"
)

func TestTheSticksAreThePadsSticks(test *testing.T) {
	state := input.NewState()
	controls := Controls{Left: Hand{StickX: 0.5, StickY: -1}, Right: Hand{StickX: -0.25, StickY: 0.75}}
	controls.Apply(state)
	if x, y := state.Stick(); x != 0.5 || y != -1 {
		test.Fatalf("left stick read as %v, %v", x, y)
	}
	if x, y := state.RightStick(); x != -0.25 || y != 0.75 {
		test.Fatalf("right stick read as %v, %v", x, y)
	}
}

func TestASticksRestIsZero(test *testing.T) {
	state := input.NewState()
	Controls{Left: Hand{StickX: 0.05, StickY: -0.05}, Right: Hand{StickX: 0.02}}.Apply(state)
	if x, y := state.Stick(); x != 0 || y != 0 {
		test.Fatalf("a stick at rest read as %v, %v", x, y)
	}
	if x, y := state.RightStick(); x != 0 || y != 0 {
		test.Fatalf("a right stick at rest read as %v, %v", x, y)
	}
}

func TestAStickPastItsEdgeStaysWithinOne(test *testing.T) {
	state := input.NewState()
	Controls{Left: Hand{StickX: 1.25, StickY: -1.5}}.Apply(state)
	if x, y := state.Stick(); x != 1 || y != -1 {
		test.Fatalf("read as %v, %v", x, y)
	}
}

func TestTheFaceButtonsAreThePadsButtons(test *testing.T) {
	cases := []struct {
		controls Controls
		button   input.PadButton
	}{
		{Controls{Right: Hand{Lower: true}}, input.PadA},
		{Controls{Right: Hand{Upper: true}}, input.PadB},
		{Controls{Left: Hand{Lower: true}}, input.PadX},
		{Controls{Left: Hand{Upper: true}}, input.PadY},
		{Controls{Left: Hand{StickClick: true}}, input.PadLeftThumb},
		{Controls{Right: Hand{StickClick: true}}, input.PadRightThumb},
	}
	everyButton := []input.PadButton{input.PadA, input.PadB, input.PadX, input.PadY, input.PadLeftThumb, input.PadRightThumb}
	for _, held := range cases {
		state := input.NewState()
		held.controls.Apply(state)
		for _, button := range everyButton {
			if state.PadDown(button) != (button == held.button) {
				test.Fatalf("holding %d: button %d is down %v", held.button, button, state.PadDown(button))
			}
		}
		if !state.PadPressed(held.button) {
			test.Fatalf("button %d went down without a press", held.button)
		}
	}
}

func TestLettingGoReleasesTheButton(test *testing.T) {
	state := input.NewState()
	Controls{Right: Hand{Lower: true}}.Apply(state)
	state.Feed().EndFrame()
	Controls{}.Apply(state)
	if state.PadDown(input.PadA) {
		test.Fatal("A is still down after it was let go")
	}
}

func TestTheMenuButtonIsTheSystemsBack(test *testing.T) {
	state := input.NewState()
	Controls{}.Apply(state)
	if state.TakeBack() {
		test.Fatal("a back without the menu button")
	}
	Controls{Menu: true}.Apply(state)
	if !state.TakeBack() {
		test.Fatal("the menu button did not go back")
	}
}
