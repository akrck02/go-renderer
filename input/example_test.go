package input_test

import (
	"fmt"

	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/window"
)

// In a program the window loop (or the browser's) fills the state; these examples press the keys
// and buttons themselves, as a backend would, to show what a program reads.

// ExampleState_Pressed reads the keyboard in an update step: F11 toggles the whole screen.
func ExampleState_Pressed() {
	state := input.NewState()
	state.Feed().SetKey(input.KeyF11, true) // the backend: F11 went down

	if state.Pressed(input.KeyF11) {
		state.SetFullscreen(!state.Fullscreen())
	}
	fmt.Println(state.Down(input.KeyF11), state.Fullscreen())

	state.Feed().EndFrame() // the backend, after the step: a press lasts one step
	fmt.Println(state.Down(input.KeyF11), state.Pressed(input.KeyF11))
	// Output:
	// true true
	// true false
}

// ExampleState_PadPressed moves a boat with a pad's left stick and jumps with its bottom button.
func ExampleState_PadPressed() {
	state := input.NewState()
	state.Feed().SetStick(0.5, 1) // the backend: half right, all the way up
	state.Feed().SetPadButton(input.PadA, true)

	const metresPerSecond = 4
	const seconds = 0.5
	stickX, stickY := state.Stick()
	forward := stickY * metresPerSecond * seconds
	sideways := stickX * metresPerSecond * seconds
	fmt.Println(forward, sideways, state.PadPressed(input.PadA))
	// Output: 2 1 true
}

// ExamplePadLayoutFromName names a pad's bottom button the way its maker prints it.
func ExamplePadLayoutFromName() {
	bottomButton := map[input.PadLayout]string{
		input.XboxLayout:        "A",
		input.PlayStationLayout: "✕",
		input.NintendoLayout:    "B",
	}
	state := input.NewState()
	// a browser's name for a pad; a window's backend reads the GUID instead (PadLayoutFromGUID)
	state.Feed().SetPadLayout(input.PadLayoutFromName("Wireless Controller (STANDARD GAMEPAD Vendor: 054c Product: 09cc)"))
	fmt.Println("Press", bottomButton[state.PadLayout()], "to jump")
	// Output: Press ✕ to jump
}

// ExampleState_SetFullscreen toggles the whole screen with F11 in an application's update step.
func ExampleState_SetFullscreen() {
	app := &window.Application{Input: input.NewState()}
	app.Update = func(app *window.Application, seconds float64) error {
		if app.Input.Pressed(input.KeyF11) {
			app.Input.SetFullscreen(!app.Input.Fullscreen())
		}
		return nil
	}

	app.Input.Feed().SetKey(input.KeyF11, true) // the backend: F11 went down, then a step runs
	if err := app.Update(app, 1.0/60); err != nil {
		fmt.Println(err)
	}
	fmt.Println(app.Input.Fullscreen())
	// Output: true
}
