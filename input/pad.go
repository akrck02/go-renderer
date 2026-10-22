package input

// A game pad's buttons, in the one layout every pad is read as (an Xbox pad's, as GLFW maps them):
// A is the bottom face button, Y the top one. The backend sets them every frame.

// PadButton is a game pad's button.
type PadButton int

// The buttons of a game pad, in that one layout.
const (
	PadA PadButton = iota
	PadB
	PadX
	PadY
	PadLeftShoulder
	PadRightShoulder
	PadBack
	PadStart
	PadGuide
	PadLeftThumb // pressing the left stick in
	PadRightThumb
	PadUp
	PadRight
	PadDown
	PadLeft
	padButtonCount
)

// SetPadButton records a game pad's button as held or let go; going down presses it this frame.
func (feed Feed) SetPadButton(button PadButton, isDown bool) {
	state := feed.state
	if button < 0 || button >= padButtonCount {
		return
	}
	if isDown && !state.padHeld[button] {
		state.padPressedNow[button] = true
	}
	state.padHeld[button] = isDown
}

// PadDown tells whether a game pad's button is held.
func (state *State) PadDown(button PadButton) bool {
	return button >= 0 && button < padButtonCount && state.padHeld[button]
}

// PadPressed tells whether a game pad's button went down this frame.
func (state *State) PadPressed(button PadButton) bool {
	return button >= 0 && button < padButtonCount && state.padPressedNow[button]
}
