package windowloop

import (
	"math"

	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/akrck02/go-renderer/input"
)

// A game pad's two sticks, read every frame from the first pad GLFW knows. GLFW maps every pad it
// recognises to one layout (an Xbox pad's), so a game reads any of them the same way. The window
// loop is the only writer of the sticks on a computer.

// gamePadDeadZone is the share of a stick's reach that counts as at rest: a pad let go reads a
// little off centre.
const gamePadDeadZone = 0.15

// gamePad follows the first game pad from frame to frame, to let its sticks go when it is
// unplugged.
type gamePad struct{ present bool }

// read copies the first game pad's sticks into the input state, and leaves them at rest once when
// the pad goes. GLFW's reading of a pad allocates (its binding returns a new state each time), so
// it is read only while a pad is there.
func (pad *gamePad) read(state *input.State) {
	present := glfw.Joystick1.IsGamepad()
	if !present {
		if pad.present {
			state.StickX, state.StickY, state.RightStickX, state.RightStickY = 0, 0, 0, 0
		}
		pad.present = false
		return
	}
	pad.present = true
	if reading := glfw.Joystick1.GetGamepadState(); reading != nil { // nil: the pad went between the calls
		sticksFromPad(state, reading.Axes)
	}
}

// sticksFromPad sets both sticks from a pad's axes, in GLFW's layout.
func sticksFromPad(state *input.State, axes [6]float32) {
	state.StickX, state.StickY = stickFromAxes(axes[glfw.AxisLeftX], axes[glfw.AxisLeftY])
	state.RightStickX, state.RightStickY = stickFromAxes(axes[glfw.AxisRightX], axes[glfw.AxisRightY])
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
