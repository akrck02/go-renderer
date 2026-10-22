package camera

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
)

// groundDistance is how far apart two points are on the ground, from above.
func groundDistance(from, to graphics.Vector) float64 {
	return math.Hypot(float64(to[0]-from[0]), float64(to[2]-from[2]))
}

func TestTheStickUpMovesTheOrbitAwayFromTheCamera(test *testing.T) {
	orbit := NewOrbit(graphics.Vector{0, 0, 0, 1}, 100)
	var camera Camera
	state := input.NewState()
	state.WindowWidth, state.WindowHeight = 400, 240
	orbit.Update(state, 0.1, &camera)
	eye, before := camera.Position, orbit.Target
	state.Feed().SetStick(0, 1)
	orbit.Update(state, 0.5, &camera)
	if groundDistance(eye, orbit.Target) <= groundDistance(eye, before) {
		test.Fatalf("the target went from %v to %v: not away from the camera at %v", before, orbit.Target, eye)
	}
}

func TestTheStickRightMovesTheMapRight(test *testing.T) {
	view := NewMap(graphics.Vector{0, 0, 0, 1}, 100)
	var camera Camera
	state := input.NewState()
	state.WindowWidth, state.WindowHeight = 400, 240
	view.Update(state, 0.1, &camera)
	before := view.Target
	state.Feed().SetStick(1, 0)
	view.Update(state, 0.5, &camera)
	if view.Target == before {
		test.Fatalf("the stick did not move the map from %v", before)
	}
}
