package windowloop

import (
	"testing"

	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/akrck02/go-renderer/input"
)

func TestAGamePadsStickReadsUpAsPositiveAndRestsInItsDeadZone(test *testing.T) {
	if x, y := stickFromAxes(0.5, -1); x != 0.5 || y != 1 {
		test.Fatalf("a stick pushed up and right reads %v, %v", x, y)
	}
	if x, y := stickFromAxes(0.05, -0.08); x != 0 || y != 0 {
		test.Fatalf("a stick a little off centre reads %v, %v", x, y)
	}
}

func TestAPadsLeftStickIsTheMainOneAndItsRightTheSecond(test *testing.T) {
	state := input.NewState()
	var axes [6]float32
	axes[glfw.AxisLeftX], axes[glfw.AxisLeftY] = 1, 0
	axes[glfw.AxisRightX], axes[glfw.AxisRightY] = 0, -1
	sticksFromPad(state, axes)
	stickX, stickY := state.Stick()
	rightX, rightY := state.RightStick()
	if stickX != 1 || stickY != 0 || rightX != 0 || rightY != 1 {
		test.Fatalf("the left stick right and the right one up read %v, %v and %v, %v", stickX, stickY, rightX, rightY)
	}
}

func TestAPadsButtonsKeepGLFWsOrder(test *testing.T) {
	state := input.NewState()
	var buttons [15]glfw.Action
	buttons[glfw.ButtonStart], buttons[glfw.ButtonA] = glfw.Press, glfw.Press
	buttonsFromPad(state, buttons)
	if !state.PadDown(input.PadStart) || !state.PadDown(input.PadA) || state.PadDown(input.PadB) {
		test.Fatal("the pad's buttons are not read in GLFW's order")
	}
}
