package windowloop

import (
	"math"

	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/akrck02/go-renderer/input"
)

// A game pad's two sticks, read every frame from the first pad GLFW knows. GLFW maps every pad it
// recognises to one layout (an Xbox pad's), so a game reads any of them the same way.

// gamePadDeadZone is the share of a stick's reach that counts as at rest: a pad let go reads a
// little off centre.
const gamePadDeadZone = 0.15

// ReadGamePad copies the first game pad's sticks into the input state, or leaves them at rest
// without one. It reads the pad's state only when there is a pad: GLFW's reading allocates.
func ReadGamePad(state *input.State) {
	if !glfw.Joystick1.IsGamepad() {
		state.StickX, state.StickY, state.RightStickX, state.RightStickY = 0, 0, 0, 0
		return
	}
	pad := glfw.Joystick1.GetGamepadState()
	if pad == nil {
		return // the pad went between the two calls
	}
	state.StickX, state.StickY = stickFromAxes(pad.Axes[glfw.AxisLeftX], pad.Axes[glfw.AxisLeftY])
	state.RightStickX, state.RightStickY = stickFromAxes(pad.Axes[glfw.AxisRightX], pad.Axes[glfw.AxisRightY])
}

// stickFromAxes is a stick from GLFW's axes, which count down as positive, as -1 to 1 with up
// positive, and 0 inside the dead zone.
func stickFromAxes(axisX, axisY float32) (x, y float64) {
	x, y = float64(axisX), -float64(axisY)
	if math.Hypot(x, y) < gamePadDeadZone {
		return 0, 0
	}
	return x, y
}
