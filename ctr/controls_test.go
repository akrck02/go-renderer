package ctr

import (
	"testing"

	"github.com/akrck02/go-renderer/input"
)

func TestTheCrossIsTheArrowKeys(test *testing.T) {
	state := input.NewState()
	var controls Controls
	controls.Apply(state, Reading{Held: ButtonUp | ButtonL})
	if !state.Down(input.KeyUp) || !state.Pressed(input.KeyUp) || !state.Down(input.KeyMinus) {
		test.Fatal("up and L should hold the up arrow and minus")
	}
	state.Feed().EndFrame()
	controls.Apply(state, Reading{})
	if state.Down(input.KeyUp) || state.Down(input.KeyMinus) {
		test.Fatal("letting go should release the keys")
	}
}

func TestBIsTheSystemsBack(test *testing.T) {
	state := input.NewState()
	var controls Controls
	controls.Apply(state, Reading{Held: ButtonB})
	if !state.Down(input.KeyEscape) {
		test.Fatal("B should hold Escape")
	}
	state.Feed().EndFrame()
	if !state.Feed().BackUnanswered() {
		test.Fatal("B should be the system's back")
	}
}

func TestDraggingOnTheTouchScreenDragsThePointer(test *testing.T) {
	state := input.NewState()
	var controls Controls
	controls.Apply(state, Reading{Held: ButtonTouch, TouchX: 100, TouchY: 50})
	if !state.Button(input.MouseLeft) || state.CursorX != 100 || state.CursorY != 50 {
		test.Fatalf("a touch should press the pointer at 100, 50; it is at %v, %v", state.CursorX, state.CursorY)
	}
	state.Feed().EndFrame()
	controls.Apply(state, Reading{Held: ButtonTouch, TouchX: 130, TouchY: 40})
	if state.CursorDeltaX != 30 || state.CursorDeltaY != -10 {
		test.Fatalf("moved by %v, %v; want 30, -10", state.CursorDeltaX, state.CursorDeltaY)
	}
	state.Feed().EndFrame()
	controls.Apply(state, Reading{}) // lifted: libctru reports no position
	if state.Button(input.MouseLeft) || !state.ButtonReleased(input.MouseLeft) {
		test.Fatal("lifting the finger should release the pointer")
	}
}

func TestTheCirclePadRestsInItsDeadZone(test *testing.T) {
	if x, y := (Reading{CircleX: 10, CircleY: -8}).Circle(); x != 0 || y != 0 {
		test.Fatalf("a slight touch gives %v, %v; want 0, 0", x, y)
	}
	if x, y := (Reading{CircleX: 156, CircleY: -200}).Circle(); x != 1 || y != -1 {
		test.Fatalf("the edge gives %v, %v; want 1, -1", x, y)
	}
}

func TestPressedIsOnlyTheNewButtons(test *testing.T) {
	var controls Controls
	controls.Apply(input.NewState(), Reading{Held: ButtonA})
	if pressed := controls.Pressed(Reading{Held: ButtonA | ButtonStart}); pressed != ButtonStart {
		test.Fatalf("pressed %b, want START only", pressed)
	}
}

func TestTheCirclePadIsTheStick(test *testing.T) {
	state := input.NewState()
	var controls Controls
	controls.Apply(state, Reading{CircleX: 156, CircleY: 0})
	if x, y := state.Stick(); x != 1 || y != 0 {
		test.Fatalf("the pad at its right edge is the stick right: %v, %v", x, y)
	}
	controls.Apply(state, Reading{CircleX: 5, CircleY: -5})
	if x, y := state.Stick(); x != 0 || y != 0 {
		test.Fatalf("the pad at rest is no stick: %v, %v", x, y)
	}
}

func TestTheCStickRestsInItsDeadZoneAndReachesOne(test *testing.T) {
	if x, y := (Reading{CStickX: 10, CStickY: -8}).CStick(); x != 0 || y != 0 {
		test.Fatalf("a C-stick a little off centre reads %v, %v", x, y)
	}
	if x, y := (Reading{CStickX: 146, CStickY: -200}).CStick(); x != 1 || y != -1 {
		test.Fatalf("a C-stick at its edge reads %v, %v", x, y)
	}
}

func TestTheGyroscopesStepsAreDividedIntoDegreesASecond(test *testing.T) {
	if turn := gyroscopeFrom(140, -70, 14, 14); turn != (Gyroscope{Roll: 10, Pitch: -5, Yaw: 1}) {
		test.Fatalf("140, -70 and 14 steps at 14 a degree read %+v", turn)
	}
}

func TestTheCStickIsTheRightStick(test *testing.T) {
	var controls Controls
	state := input.NewState()
	controls.Apply(state, Reading{CStickX: 146, CStickY: 146})
	if x, y := state.RightStick(); x != 1 || y != 1 {
		test.Fatalf("the C-stick up and right reads as the right stick at %v, %v", x, y)
	}
}
