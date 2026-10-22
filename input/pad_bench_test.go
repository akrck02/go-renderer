package input

import "testing"

// BenchmarkReadingAPadForAFrame measures a frame of a game reading a pad: every button set by the
// backend, then asked whether it is held and whether it was just pressed.
func BenchmarkReadingAPadForAFrame(benchmark *testing.B) {
	state := NewState()
	answered := 0
	benchmark.ReportAllocs()
	for range benchmark.N {
		for button := range padButtonCount {
			state.Feed().SetPadButton(button, button%2 == 0)
		}
		for button := range padButtonCount {
			if state.PadDown(button) && state.PadPressed(button) {
				answered++
			}
		}
		state.Feed().EndFrame()
	}
	if answered == 0 {
		benchmark.Fatal("no button was read as pressed")
	}
}
