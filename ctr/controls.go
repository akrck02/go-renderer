package ctr

// The console's buttons, circle pad and touch screen, turned into input.State, so cameras and
// interfaces written for a keyboard and a mouse work on it. Reading the hardware is built only for
// the nintendo3ds target (controls_read.go); turning a reading into input is plain Go, tested on
// any computer.

import (
	"math"

	"github.com/akrck02/go-renderer/input"
)

// Buttons are held buttons, one bit each, numbered as libctru numbers them (KEY_A, KEY_B, …).
type Buttons uint32

const (
	ButtonA      Buttons = 1 << 0
	ButtonB      Buttons = 1 << 1
	ButtonSelect Buttons = 1 << 2
	ButtonStart  Buttons = 1 << 3
	ButtonRight  Buttons = 1 << 4
	ButtonLeft   Buttons = 1 << 5
	ButtonUp     Buttons = 1 << 6
	ButtonDown   Buttons = 1 << 7
	ButtonR      Buttons = 1 << 8
	ButtonL      Buttons = 1 << 9
	ButtonX      Buttons = 1 << 10
	ButtonY      Buttons = 1 << 11
	ButtonZL     Buttons = 1 << 14 // New 3DS only
	ButtonZR     Buttons = 1 << 15 // New 3DS only
	ButtonTouch  Buttons = 1 << 20 // a finger or the stylus on the bottom screen
)

// buttonKeys are the keys the buttons stand for: the cross is the arrow keys, L and R zoom out
// and in (minus and plus), A is Enter, B is Escape, X and Y are the X and Y keys.
var buttonKeys = [...]struct {
	button Buttons
	key    input.Key
}{
	{ButtonUp, input.KeyUp}, {ButtonDown, input.KeyDown}, {ButtonLeft, input.KeyLeft}, {ButtonRight, input.KeyRight},
	{ButtonL, input.KeyMinus}, {ButtonR, input.KeyPlus},
	{ButtonA, input.KeyEnter}, {ButtonB, input.KeyEscape}, {ButtonX, input.KeyX}, {ButtonY, input.KeyY},
}

// circlePadReach is how far libctru reports the circle pad at its edge.
const circlePadReach = 156

// circlePadDeadZone is the share of the reach that counts as the pad at rest.
const circlePadDeadZone = 0.12

// Reading is the hardware at one moment.
type Reading struct {
	Held             Buttons
	TouchX, TouchY   int // on the bottom screen, 320 × 240, when ButtonTouch is held
	CircleX, CircleY int // the circle pad, ±156; up is positive
	CStickX, CStickY int // the C-stick, about ±146, once StartMotionControls started it
	Gyroscope        Gyroscope
}

// Gyroscope is how fast the console turns, in degrees a second, once StartMotionControls started
// it: about the axis across its screens (Pitch), up through them (Yaw) and out of them (Roll).
type Gyroscope struct {
	Roll, Pitch, Yaw float64
}

// gyroscopeFrom turns the gyroscope's raw steps into degrees a second. libctru's coefficient
// divides the steps: multiplying by it read a slight turn as thousands of degrees a second.
func gyroscopeFrom(roll, pitch, yaw int, stepsPerDegree float64) Gyroscope {
	return Gyroscope{Roll: float64(roll) / stepsPerDegree, Pitch: float64(pitch) / stepsPerDegree, Yaw: float64(yaw) / stepsPerDegree}
}

// The C-stick's reach and the share of it that counts as at rest: it reads about ±146 at its edge,
// and a little off centre when let go.
const (
	cStickReach    = 146
	cStickDeadZone = 0.15
)

// CStick is the C-stick from -1 to 1 on each axis (up positive), 0 inside its dead zone.
func (reading Reading) CStick() (x, y float64) {
	return stickAxes(reading.CStickX, reading.CStickY, cStickReach, cStickDeadZone)
}

// Circle is the circle pad from -1 to 1 on each axis (up positive), 0 inside its dead zone.
func (reading Reading) Circle() (x, y float64) {
	return stickAxes(reading.CircleX, reading.CircleY, circlePadReach, circlePadDeadZone)
}

// stickAxes is a stick's raw position as -1 to 1 on each axis, 0 inside its dead zone (a share of
// its reach).
func stickAxes(rawX, rawY int, reach, deadZone float64) (x, y float64) {
	x, y = float64(rawX)/reach, float64(rawY)/reach
	if math.Hypot(x, y) < deadZone {
		return 0, 0
	}
	return math.Max(-1, math.Min(1, x)), math.Max(-1, math.Min(1, y))
}

// Controls follows the hardware from reading to reading, to tell presses and releases apart.
type Controls struct {
	previous       Reading
	pointer        input.TouchPointer
	touchX, touchY float32 // where the finger was last: libctru reports nothing once it is lifted
}

// Pressed are the buttons held now that were not held at the previous reading.
func (controls *Controls) Pressed(reading Reading) Buttons {
	return reading.Held &^ controls.previous.Held
}

// Apply puts a reading into the input state: the buttons as keys (B also as the system's back),
// the circle pad as the stick, and the touch screen as the pointer, so dragging works like dragging
// with a mouse.
func (controls *Controls) Apply(state *input.State, reading Reading) {
	changed := reading.Held ^ controls.previous.Held
	for _, mapping := range buttonKeys {
		if changed&mapping.button != 0 {
			state.SetKey(mapping.key, reading.Held&mapping.button != 0)
		}
	}
	if controls.Pressed(reading)&ButtonB != 0 {
		state.PressBack()
	}
	state.StickX, state.StickY = reading.Circle()
	state.RightStickX, state.RightStickY = reading.CStick() // at rest until StartMotionControls
	controls.applyTouch(state, reading)
	controls.previous = reading
}

func (controls *Controls) applyTouch(state *input.State, reading Reading) {
	const finger = 0
	touching, wasTouching := reading.Held&ButtonTouch != 0, controls.previous.Held&ButtonTouch != 0
	switch {
	case touching && !wasTouching:
		controls.touchX, controls.touchY = float32(reading.TouchX), float32(reading.TouchY)
		controls.pointer.Begin(state, finger, controls.touchX, controls.touchY, 1)
	case touching:
		controls.touchX, controls.touchY = float32(reading.TouchX), float32(reading.TouchY)
		controls.pointer.Move(state, finger, controls.touchX, controls.touchY, 1)
	case wasTouching:
		controls.pointer.End(state, finger, controls.touchX, controls.touchY, 1)
	}
}
