package openxr

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
)

// Hand is one controller at a frame's display time.
type Hand struct {
	Grip, Aim      graphics.Pose // where it is held, and where it points (forward is -z)
	Tracked        bool          // the runtime knows where it is: Grip and Aim mean something
	Trigger        float32       // 0 to 1
	Squeeze        float32       // the grip button, 0 to 1
	StickX, StickY float32       // -1 to 1, right and up positive
	StickClick     bool          // the stick pressed in
	Upper, Lower   bool          // the face buttons: B and A on the right controller, Y and X on the left
}

// Side is a hand, to ask something of one controller (Session.Vibrate).
type Side int

const (
	LeftHand Side = iota
	RightHand
)

// Controls are both controllers and the menu button.
type Controls struct {
	Left, Right Hand
	Menu        bool // the menu button went down since the controls were last read
}

// stickDeadZone is how far from the centre a stick is still read as at rest: a thumbstick let go
// rarely reports exactly 0, and a camera following it would drift.
const stickDeadZone = 0.1

// faceButtons are the pad buttons the face buttons stand for, named for the letters printed on
// them, which are where a pad's A, B, X and Y are.
var faceButtons = [...]struct {
	held   func(controls Controls) bool
	button input.PadButton
}{
	{func(controls Controls) bool { return controls.Right.Lower }, input.PadA},
	{func(controls Controls) bool { return controls.Right.Upper }, input.PadB},
	{func(controls Controls) bool { return controls.Left.Lower }, input.PadX},
	{func(controls Controls) bool { return controls.Left.Upper }, input.PadY},
	{func(controls Controls) bool { return controls.Left.StickClick }, input.PadLeftThumb},
	{func(controls Controls) bool { return controls.Right.StickClick }, input.PadRightThumb},
}

// Apply puts the controllers into the input state as a game pad, so cameras and interfaces
// written for one work in a headset: the left stick is the main stick, the right stick the second,
// A, B, X and Y and the sticks' clicks are the pad's buttons, and the menu button is the system's
// back. The poses, triggers and grips are not input.State's: a program reads them from the Hands.
func (controls Controls) Apply(state *input.State) {
	feed := state.Feed()
	feed.SetStick(stickAxes(controls.Left.StickX, controls.Left.StickY))
	feed.SetRightStick(stickAxes(controls.Right.StickX, controls.Right.StickY))
	for _, mapping := range faceButtons {
		feed.SetPadButton(mapping.button, mapping.held(controls))
	}
	if controls.Menu {
		feed.PressBack()
	}
}

// stickAxes is a stick as input.State keeps it: 0 inside the dead zone, else each axis kept
// within -1 and 1.
func stickAxes(x, y float32) (float64, float64) {
	if math.Hypot(float64(x), float64(y)) < stickDeadZone {
		return 0, 0
	}
	return clampUnit(float64(x)), clampUnit(float64(y))
}

func clampUnit(value float64) float64 { return math.Max(-1, math.Min(1, value)) }
