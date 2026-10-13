package input

// TouchPointer turns fingers into the pointer every interface already understands: the first finger
// down moves the cursor and presses the left button, lifting it releases the button. Other fingers
// are ignored until it lifts. Positions come in screen pixels and go to the state in layout pixels,
// divided by the screen's scale.
type TouchPointer struct {
	finger int64 // the touch that is the pointer
	down   bool
}

// Begin starts a touch; the first finger becomes the pointer.
func (pointer *TouchPointer) Begin(state *State, finger int64, x, y, scale float32) {
	if pointer.down {
		return
	}
	pointer.finger, pointer.down = finger, true
	state.MoveCursor(float64(x/scale), float64(y/scale))
	state.SetButton(MouseLeft, true)
}

// Move follows the pointer's finger.
func (pointer *TouchPointer) Move(state *State, finger int64, x, y, scale float32) {
	if !pointer.down || finger != pointer.finger {
		return
	}
	state.MoveCursor(float64(x/scale), float64(y/scale))
}

// End lifts the pointer's finger: a tap is a click where the finger was.
func (pointer *TouchPointer) End(state *State, finger int64, x, y, scale float32) {
	if !pointer.down || finger != pointer.finger {
		return
	}
	pointer.down = false
	state.MoveCursor(float64(x/scale), float64(y/scale))
	state.SetButton(MouseLeft, false)
}

// Cancel lets go of the pointer where it is, when the system takes the gesture over (a swipe from
// the screen's edge): the button is released, so nothing stays pressed.
func (pointer *TouchPointer) Cancel(state *State) {
	if !pointer.down {
		return
	}
	pointer.down = false
	state.SetButton(MouseLeft, false)
}
