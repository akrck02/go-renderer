package input

// Feed is the side of a State that backends write: the keys, buttons, pointer, touches and pads
// they hear of, and the end of each frame. Programs read the State itself and only need Feed to
// play input to their own code in tests. It is a small value that holds the state, so calling
// state.Feed() for each event costs nothing.
type Feed struct {
	state *State
}

// Feed returns the side of the state that backends write.
func (state *State) Feed() Feed { return Feed{state: state} }

// SetStick sets a game pad's main stick (a console's circle pad, a pad's left stick), from -1 to
// 1 each way, right and up positive.
func (feed Feed) SetStick(x, y float64) { feed.state.stickX, feed.state.stickY = x, y }

// SetRightStick sets a game pad's second stick (a New 3DS's C-stick, a pad's right one).
func (feed Feed) SetRightStick(x, y float64) {
	feed.state.rightStickX, feed.state.rightStickY = x, y
}

// Stick is a game pad's main stick where there is one (a console's circle pad, a pad's left
// stick), from -1 to 1 each way, right and up positive; 0 at rest or without one.
func (state *State) Stick() (x, y float64) { return state.stickX, state.stickY }

// RightStick is a game pad's second stick (a New 3DS's C-stick, a pad's right one), as Stick.
func (state *State) RightStick() (x, y float64) { return state.rightStickX, state.rightStickY }
