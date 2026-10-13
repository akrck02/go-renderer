package input

import "math"

// Touch is a finger on the screen: which one, and where, in layout pixels.
type Touch struct {
	Finger int64
	X, Y   float64
}

// Touches returns the fingers on the screen now: for gestures of more than one finger (pinch,
// twist). One finger alone is also the pointer (TouchPointer).
func (state *State) Touches() []Touch { return state.touches }

// SetTouch puts a finger down, or moves it (called by the backend).
func (state *State) SetTouch(finger int64, x, y float64) {
	for index := range state.touches {
		if state.touches[index].Finger == finger {
			state.touches[index].X, state.touches[index].Y = x, y
			return
		}
	}
	state.touches = append(state.touches, Touch{Finger: finger, X: x, Y: y})
}

// LiftTouch takes a finger off the screen (called by the backend).
func (state *State) LiftTouch(finger int64) {
	for index := range state.touches {
		if state.touches[index].Finger == finger {
			state.touches = append(state.touches[:index], state.touches[index+1:]...)
			return
		}
	}
}

// ClearTouches takes every finger off, when the system takes a gesture over (called by the backend).
func (state *State) ClearTouches() { state.touches = state.touches[:0] }

// TouchPointer turns fingers into the pointer every interface already understands: one finger
// down moves the cursor and presses the left button, lifting it releases the button. A second
// finger lets the pointer go: two fingers are a gesture (pinch, twist), not a click or a drag; the
// pointer comes back with the next finger down once all are lifted. Positions come in screen
// pixels and go to the state in layout pixels, divided by the screen's scale; every finger is also
// in the state's touches.
type TouchPointer struct {
	finger  int64 // the touch that is the pointer
	down    bool  // the pointer's finger presses the button
	fingers int   // how many fingers are on the screen
}

// Begin puts a finger down: the first becomes the pointer, a second lets it go.
func (pointer *TouchPointer) Begin(state *State, finger int64, x, y, scale float32) {
	pointer.fingers++
	state.SetTouch(finger, float64(x/scale), float64(y/scale))
	if pointer.fingers > 1 {
		pointer.letGo(state)
		return
	}
	pointer.finger, pointer.down = finger, true
	state.MoveCursor(float64(x/scale), float64(y/scale))
	state.SetButton(MouseLeft, true)
}

// Move follows a finger; the pointer follows its own.
func (pointer *TouchPointer) Move(state *State, finger int64, x, y, scale float32) {
	state.SetTouch(finger, float64(x/scale), float64(y/scale))
	if !pointer.down || finger != pointer.finger {
		return
	}
	state.MoveCursor(float64(x/scale), float64(y/scale))
}

// End lifts a finger: lifting the pointer's is a click where it was.
func (pointer *TouchPointer) End(state *State, finger int64, x, y, scale float32) {
	pointer.fingers = max(pointer.fingers-1, 0)
	state.LiftTouch(finger)
	if !pointer.down || finger != pointer.finger {
		return
	}
	pointer.down = false
	state.MoveCursor(float64(x/scale), float64(y/scale))
	state.SetButton(MouseLeft, false)
}

// Cancel lets go of every finger, when the system takes the gesture over (a swipe from the
// screen's edge): the button is released, so nothing stays pressed.
func (pointer *TouchPointer) Cancel(state *State) {
	pointer.fingers = 0
	state.ClearTouches()
	pointer.letGo(state)
}

func (pointer *TouchPointer) letGo(state *State) {
	if !pointer.down {
		return
	}
	pointer.down = false
	state.SetButton(MouseLeft, false)
}

// TwoFingers follows a gesture of two fingers from one step to the next: how much they spread
// (pinch), turned (twist) and moved together (pan).
type TwoFingers struct {
	first, second Touch // where the two fingers were at the last step
	following     bool
}

// Gesture is what two fingers did since the last step.
type Gesture struct {
	Spread float64 // the distance between them now over before: above 1 they moved apart
	Turn   float64 // radians they turned around their middle, counterclockwise on screen
	PanX   float64 // how far their middle moved, in layout pixels
	PanY   float64
	// CenterX and CenterY are where their middle is now, in layout pixels: what a pinch comes
	// closer to
	CenterX, CenterY float64
}

// Step reads the state's touches and returns the gesture since the last step; false when there
// are not exactly two fingers, or they just came down (a gesture starts at the next step).
func (follower *TwoFingers) Step(state *State) (Gesture, bool) {
	touches := state.Touches()
	if len(touches) != 2 {
		follower.following = false
		return Gesture{}, false
	}
	first, second := touches[0], touches[1]
	same := follower.following && first.Finger == follower.first.Finger && second.Finger == follower.second.Finger
	before, after := [2]Touch{follower.first, follower.second}, [2]Touch{first, second}
	follower.first, follower.second, follower.following = first, second, true
	if !same {
		return Gesture{}, false
	}
	return gestureBetween(before, after), true
}

func gestureBetween(before, after [2]Touch) Gesture {
	spanBefore, spanAfter := span(before), span(after)
	gesture := Gesture{Spread: 1}
	if spanBefore > 1 {
		gesture.Spread = spanAfter / spanBefore
	}
	turn := angle(after) - angle(before)
	gesture.Turn = math.Atan2(math.Sin(turn), math.Cos(turn)) // within half a turn either way
	gesture.PanX = (after[0].X + after[1].X - before[0].X - before[1].X) / 2
	gesture.PanY = (after[0].Y + after[1].Y - before[0].Y - before[1].Y) / 2
	gesture.CenterX, gesture.CenterY = (after[0].X+after[1].X)/2, (after[0].Y+after[1].Y)/2
	return gesture
}

func span(pair [2]Touch) float64 { return math.Hypot(pair[1].X-pair[0].X, pair[1].Y-pair[0].Y) }

// angle is the direction from the first finger to the second, counterclockwise on screen (whose y
// goes down).
func angle(pair [2]Touch) float64 { return math.Atan2(-(pair[1].Y - pair[0].Y), pair[1].X-pair[0].X) }
