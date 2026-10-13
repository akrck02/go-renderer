package input

import "testing"

func TestATapIsAClickWhereTheFingerWas(t *testing.T) {
	state := NewState()
	var pointer TouchPointer
	pointer.Begin(state, 1, 300, 600, 3)
	if !state.ButtonPressed(MouseLeft) || state.CursorX != 100 || state.CursorY != 200 {
		t.Fatalf("pressed %v at %v, %v; want a press at 100, 200 layout pixels", state.ButtonPressed(MouseLeft), state.CursorX, state.CursorY)
	}
	pointer.End(state, 1, 303, 600, 3)
	if !state.ButtonReleased(MouseLeft) || state.CursorX != 101 {
		t.Fatalf("released %v at %v; want a release at 101", state.ButtonReleased(MouseLeft), state.CursorX)
	}
}

func TestASecondFingerIsIgnored(t *testing.T) {
	state := NewState()
	var pointer TouchPointer
	pointer.Begin(state, 1, 100, 100, 1)
	pointer.Begin(state, 2, 500, 500, 1)
	pointer.Move(state, 2, 600, 600, 1)
	if state.CursorX != 100 || state.CursorY != 100 {
		t.Fatalf("cursor at %v, %v; the second finger should not move it", state.CursorX, state.CursorY)
	}
	pointer.End(state, 2, 600, 600, 1)
	if !state.Button(MouseLeft) {
		t.Fatal("lifting the second finger released the first one's press")
	}
	pointer.Move(state, 1, 140, 100, 1)
	if state.CursorX != 140 {
		t.Fatalf("cursor at %v; the first finger should still drag", state.CursorX)
	}
}

func TestACancelledGestureReleasesThePointer(t *testing.T) {
	state := NewState()
	var pointer TouchPointer
	pointer.Begin(state, 1, 100, 100, 1)
	pointer.Cancel(state)
	if state.Button(MouseLeft) {
		t.Fatal("the button stays pressed after the system took the gesture")
	}
	pointer.Begin(state, 2, 50, 50, 1)
	if !state.Button(MouseLeft) {
		t.Fatal("a new finger after a cancel should press again")
	}
}
