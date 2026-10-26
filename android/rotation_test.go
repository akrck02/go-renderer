package android

import (
	"math"
	"testing"

	space "github.com/akrck02/go-renderer/graphics"
)

func near(got, want space.Vector) bool {
	for index := 0; index < 3; index++ {
		if math.Abs(float64(got[index]-want[index])) > 1e-5 {
			return false
		}
	}
	return true
}

// facing is where a camera turned by the screen's orientation looks, and which way is its up.
func facing(orientation space.Quaternion) (ahead, up space.Vector) {
	return orientation.Rotate(space.Vector{0, 0, -1, 0}), orientation.Rotate(space.Vector{0, 1, 0, 0})
}

func TestAPhoneFlatOnATableLooksDownIntoIt(test *testing.T) {
	ahead, up := facing(screenTurned(space.IdentityQuaternion(), 0))
	if !near(ahead, space.Vector{0, -1, 0, 0}) {
		test.Fatalf("a phone lying face up looks along %v, want straight down", ahead)
	}
	if !near(up, space.Vector{0, 0, -1, 0}) {
		test.Fatalf("its top points along %v, want ahead, as it lies", up)
	}
}

func TestAnUprightPhoneLooksAheadWithItsTopUp(test *testing.T) {
	// held up from the table: turned a quarter about its own x, its back to the north
	upright := aroundAxis(1, 0, 0, math.Pi/2)
	ahead, up := facing(screenTurned(upright, 0))
	if !near(ahead, space.Vector{0, 0, -1, 0}) || !near(up, space.Vector{0, 1, 0, 0}) {
		test.Fatalf("an upright phone looks along %v with its up along %v, want ahead and up", ahead, up)
	}
}

func TestAPhoneOnItsSideKeepsTheRoomUpright(test *testing.T) {
	upright := aroundAxis(1, 0, 0, math.Pi/2)
	for _, turn := range []struct {
		quarterTurns int
		radians      float64 // how the phone is turned about the way it looks, its top to the left
	}{{1, math.Pi / 2}, {2, math.Pi}, {3, -math.Pi / 2}} {
		// the phone turned on its back's axis, and the screen turned the other way to stay upright
		onItsSide := upright.Multiply(aroundAxis(0, 0, 1, turn.radians))
		ahead, up := facing(screenTurned(onItsSide, turn.quarterTurns))
		if !near(ahead, space.Vector{0, 0, -1, 0}) || !near(up, space.Vector{0, 1, 0, 0}) {
			test.Errorf("%d quarter turns: the screen looks along %v with its up along %v, want ahead and up", turn.quarterTurns, ahead, up)
		}
	}
}

func TestTurningThePhoneTurnsTheView(test *testing.T) {
	upright := aroundAxis(1, 0, 0, math.Pi/2)
	left := aroundAxis(0, 0, 1, math.Pi/2).Multiply(upright) // turned to the left about the room's up
	ahead, _ := facing(screenTurned(left, 0))
	if !near(ahead, space.Vector{-1, 0, 0, 0}) {
		test.Fatalf("turned to the left the phone looks along %v, want to the left", ahead)
	}
}
