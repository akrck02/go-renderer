package windowloop

import (
	"testing"

	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/akrck02/go-renderer/input"
)

func TestAGamePadsStickReadsUpAsPositiveAndRestsInItsDeadZone(t *testing.T) {
	if x, y := stickFromAxes(0.5, -1); x != 0.5 || y != 1 {
		t.Fatalf("a stick pushed up and right reads %v, %v", x, y)
	}
	if x, y := stickFromAxes(0.05, -0.08); x != 0 || y != 0 {
		t.Fatalf("a stick a little off centre reads %v, %v", x, y)
	}
}

func TestAPadsLeftStickIsTheMainOneAndItsRightTheSecond(t *testing.T) {
	state := input.NewState()
	var axes [6]float32
	axes[glfw.AxisLeftX], axes[glfw.AxisLeftY] = 1, 0
	axes[glfw.AxisRightX], axes[glfw.AxisRightY] = 0, -1
	sticksFromPad(state, axes)
	if state.StickX != 1 || state.StickY != 0 || state.RightStickX != 0 || state.RightStickY != 1 {
		t.Fatalf("the left stick right and the right one up read %v, %v and %v, %v", state.StickX, state.StickY, state.RightStickX, state.RightStickY)
	}
}

func TestAPadsButtonsKeepGLFWsOrder(t *testing.T) {
	state := input.NewState()
	var buttons [15]glfw.Action
	buttons[glfw.ButtonStart], buttons[glfw.ButtonA] = glfw.Press, glfw.Press
	buttonsFromPad(state, buttons)
	if !state.PadDown(input.PadStart) || !state.PadDown(input.PadA) || state.PadDown(input.PadB) {
		t.Fatal("the pad's buttons are not read in GLFW's order")
	}
}
