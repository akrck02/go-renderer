package input

import "testing"

func TestAPadButtonIsPressedOnceAndHeldUntilLetGo(t *testing.T) {
	state := NewState()
	state.SetPadButton(PadA, true)
	if !state.PadPressed(PadA) || !state.PadDown(PadA) {
		t.Fatal("a pad button going down is not pressed and held")
	}
	state.EndFrame()
	state.SetPadButton(PadA, true)
	if state.PadPressed(PadA) || !state.PadDown(PadA) {
		t.Fatal("a pad button held is pressed again")
	}
	state.SetPadButton(PadA, false)
	if state.PadDown(PadA) {
		t.Fatal("a pad button let go is still held")
	}
}
