package input

import (
	"math"
	"testing"
)

func TestATapIsAClickWhereTheFingerWas(test *testing.T) {
	state := NewState()
	var pointer TouchPointer
	pointer.Begin(state, 1, 300, 600, 3)
	if !state.ButtonPressed(MouseLeft) || state.CursorX != 100 || state.CursorY != 200 {
		test.Fatalf("pressed %v at %v, %v; want a press at 100, 200 layout pixels", state.ButtonPressed(MouseLeft), state.CursorX, state.CursorY)
	}
	pointer.End(state, 1, 303, 600, 3)
	if !state.ButtonReleased(MouseLeft) || state.CursorX != 101 {
		test.Fatalf("released %v at %v; want a release at 101", state.ButtonReleased(MouseLeft), state.CursorX)
	}
}

func TestASecondFingerLetsThePointerGo(test *testing.T) {
	state := NewState()
	var pointer TouchPointer
	pointer.Begin(state, 1, 100, 100, 1)
	pointer.Begin(state, 2, 500, 500, 1)
	if state.Button(MouseLeft) {
		test.Fatal("two fingers are a gesture: the pointer should not stay pressed")
	}
	if len(state.Touches()) != 2 {
		test.Fatalf("touches = %v, want both fingers", state.Touches())
	}
	pointer.End(state, 2, 500, 500, 1)
	pointer.Move(state, 1, 140, 100, 1)
	if state.Button(MouseLeft) || state.CursorX != 100 {
		test.Fatal("the first finger left alone should not drag: the pointer comes back with the next finger")
	}
	pointer.End(state, 1, 140, 100, 1)
	pointer.Begin(state, 3, 50, 60, 1)
	if !state.Button(MouseLeft) || state.CursorX != 50 {
		test.Fatal("a new finger after all were lifted should press again")
	}
}

func TestACancelledGestureReleasesEverything(test *testing.T) {
	state := NewState()
	var pointer TouchPointer
	pointer.Begin(state, 1, 100, 100, 1)
	pointer.Cancel(state)
	if state.Button(MouseLeft) || len(state.Touches()) != 0 {
		test.Fatal("the button or a finger stays after the system took the gesture")
	}
}

func TestTwoFingersPinchTwistAndPan(test *testing.T) {
	state := NewState()
	var follower TwoFingers
	state.Feed().SetTouch(1, 100, 100)
	state.Feed().SetTouch(2, 200, 100)
	if _, ok := follower.Step(state); ok {
		test.Fatal("a gesture starts at the step after the fingers came down")
	}
	// apart to twice the distance, turned a quarter counterclockwise, the middle moved right by 10
	state.Feed().SetTouch(1, 160, 200)
	state.Feed().SetTouch(2, 160, 0)
	gesture, ok := follower.Step(state)
	if !ok {
		test.Fatal("no gesture with two fingers")
	}
	if math.Abs(gesture.Spread-2) > 1e-9 || math.Abs(gesture.Turn-math.Pi/2) > 1e-9 || gesture.PanX != 10 || gesture.PanY != 0 {
		test.Fatalf("gesture = %+v, want spread 2, a quarter turn, pan 10, 0", gesture)
	}
	state.Feed().LiftTouch(2)
	if _, ok := follower.Step(state); ok {
		test.Fatal("one finger is no gesture")
	}
}

func TestTheUnclaimedCopyKeepsItsOwnTouches(test *testing.T) {
	state := NewState()
	state.Feed().SetTouch(1, 10, 10)
	copied := state.Unclaimed()
	copied.Feed().SetTouch(1, 99, 99)
	if state.Touches()[0].X != 10 {
		test.Fatal("changing the copy's touches changed the state's")
	}
}

func TestALiftedFingerLeavesNothingUnderIt(test *testing.T) {
	state := NewState()
	var pointer TouchPointer
	pointer.Begin(state, 1, 100, 200, 1)
	state.Feed().EndFrame()
	pointer.End(state, 1, 100, 200, 1)
	if state.CursorX != 100 || !state.ButtonReleased(MouseLeft) {
		test.Fatal("the release does not land where the finger was")
	}
	state.Feed().EndFrame()
	if state.CursorX > -1000 {
		test.Fatalf("cursor still at %v after the frame: a control there would stay lit", state.CursorX)
	}
	pointer.Begin(state, 2, 50, 60, 1)
	if state.CursorX != 50 || state.CursorDeltaX != 0 {
		test.Fatalf("next touch at %v with a movement of %v: want it there, with no jump", state.CursorX, state.CursorDeltaX)
	}
	state.Feed().EndFrame()
	if state.CursorX != 50 {
		test.Fatal("a finger down was taken off the window")
	}
}

func TestTheBackIsAnsweredOnceOrLeftUnanswered(test *testing.T) {
	state := NewState()
	state.Feed().PressBack()
	if !state.TakeBack() || state.TakeBack() {
		test.Fatal("a back is taken once, by the first part of the program that asks")
	}
	state.Feed().EndFrame()
	if state.Feed().BackUnanswered() {
		test.Fatal("a taken back was left unanswered")
	}
	state.Feed().PressBack()
	state.Feed().EndFrame()
	if !state.Feed().BackUnanswered() || state.Feed().BackUnanswered() {
		test.Fatal("a back nobody took is told once")
	}
	if state.TakeBack() {
		test.Fatal("a back is answered in its own step only")
	}
}
