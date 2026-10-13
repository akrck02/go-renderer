package camera

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
)

func TestSpreadingTwoFingersComesCloser(t *testing.T) {
	orbit := NewOrbit(graphics.Vec4{0, 0, 0, 1}, 100)
	before := orbit.Distance
	orbit.Gesture(input.Gesture{Spread: 2, CenterX: 400, CenterY: 400}, 800, 800)
	if orbit.Distance != before/2 {
		t.Fatalf("distance = %v, want half of %v", orbit.Distance, before)
	}
	orbit.Gesture(input.Gesture{Spread: 1, Turn: 0.5, CenterX: 400, CenterY: 400}, 800, 800)
	if math.Abs(orbit.Yaw-(-0.1)) > 1e-9 {
		t.Fatalf("yaw = %v, want 0.4 - 0.5", orbit.Yaw)
	}
}

func TestPinchingInTheMiddleKeepsTheTarget(t *testing.T) {
	orbit := NewOrbit(graphics.Vec4{0, 0, 0, 1}, 100)
	orbit.Gesture(input.Gesture{Spread: 2, CenterX: 400, CenterY: 300}, 800, 600)
	if orbit.Target[0] != 0 || orbit.Target[2] != 0 {
		t.Fatalf("target moved to %v pinching at the window's middle", orbit.Target)
	}
}

// What is under the fingers stays under them: pinching on the right comes closer to the right.
func TestPinchingComesCloserToWhatIsBetweenTheFingers(t *testing.T) {
	orbit := NewOrbit(graphics.Vec4{0, 0, 0, 1}, 100)
	orbit.Yaw = 0 // the screen's right is +X
	const width, height, fingersRight = 800, 600, 200.0
	worldPerPixel := func() float64 { return orbit.Distance / height * 0.9 }
	underFingersBefore := fingersRight * worldPerPixel()
	orbit.Gesture(input.Gesture{Spread: 2, CenterX: width/2 + fingersRight, CenterY: height / 2}, width, height)
	underFingersAfter := float64(orbit.Target[0]) + fingersRight*worldPerPixel()
	if math.Abs(underFingersAfter-underFingersBefore) > 1e-3 {
		t.Fatalf("under the fingers: x %v before, %v after; want the same point", underFingersBefore, underFingersAfter)
	}
}

// Fingers turning a quarter counterclockwise around the window's middle: what was on its right
// is now above it, as when turning a paper map.
func TestTurningTwoFingersTurnsThePlantWithThem(t *testing.T) {
	orbit := NewOrbit(graphics.Vec4{0, 0, 0, 1}, 100)
	rightX, rightZ := orbit.groundOffset(100, 0, 600)
	orbit.Gesture(input.Gesture{Spread: 1, Turn: math.Pi / 2, CenterX: 400, CenterY: 300}, 800, 600)
	aboveX, aboveZ := orbit.groundOffset(0, -100, 600)
	if math.Abs(aboveX-rightX) > 1e-3 || math.Abs(aboveZ-rightZ) > 1e-3 {
		t.Fatalf("above the middle is %v, %v; want what was on its right, %v, %v", aboveX, aboveZ, rightX, rightZ)
	}
}

// Pinching, turning and moving all at once, away from the middle: the ground under the fingers
// stays under them.
func TestTheGroundStaysBetweenTheFingers(t *testing.T) {
	orbit := NewOrbit(graphics.Vec4{0, 0, 0, 1}, 100)
	const width, height = 800, 600
	gesture := input.Gesture{Spread: 1.3, Turn: 0.4, PanX: 30, PanY: -20, CenterX: 600, CenterY: 150}
	underX, underZ := orbit.groundOffset(gesture.CenterX-gesture.PanX-width/2, gesture.CenterY-gesture.PanY-height/2, height)
	groundX, groundZ := float64(orbit.Target[0])+underX, float64(orbit.Target[2])+underZ
	orbit.Gesture(gesture, width, height)
	underX, underZ = orbit.groundOffset(gesture.CenterX-width/2, gesture.CenterY-height/2, height)
	nowX, nowZ := float64(orbit.Target[0])+underX, float64(orbit.Target[2])+underZ
	if math.Abs(nowX-groundX) > 1e-3 || math.Abs(nowZ-groundZ) > 1e-3 {
		t.Fatalf("under the fingers: %v, %v before, %v, %v after", groundX, groundZ, nowX, nowZ)
	}
}

func TestMovingTwoFingersMovesTheTarget(t *testing.T) {
	orbit := NewOrbit(graphics.Vec4{0, 0, 0, 1}, 100)
	orbit.Gesture(input.Gesture{Spread: 1, PanX: 40}, 800, 800)
	if orbit.Target[0] == 0 && orbit.Target[2] == 0 {
		t.Fatal("the target did not move with the fingers")
	}
}

func TestAStickWalksAhead(t *testing.T) {
	walk := NewWalk(0, 0, 1.7, 2)
	walk.Steer(1, 0, 1) // full ahead for a second, facing -Z
	if math.Abs(float64(walk.Z+2)) > 1e-5 || math.Abs(float64(walk.X)) > 1e-5 {
		t.Fatalf("at %v, %v; want two units towards -Z", walk.X, walk.Z)
	}
	walk.Turn(100, 0)
	if walk.Yaw <= 0 {
		t.Fatal("a finger moving right should turn the walker right")
	}
}
