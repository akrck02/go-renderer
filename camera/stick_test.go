package camera

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
)

// groundDistance is how far apart two points are on the ground, from above.
func groundDistance(from, to graphics.Vec4) float64 {
	return math.Hypot(float64(to[0]-from[0]), float64(to[2]-from[2]))
}

func TestTheStickUpMovesTheOrbitAwayFromTheCamera(t *testing.T) {
	orbit := NewOrbit(graphics.Vec4{0, 0, 0, 1}, 100)
	var camera models.Camera
	state := input.NewState()
	state.WindowWidth, state.WindowHeight = 400, 240
	orbit.Update(state, 0.1, &camera)
	eye, before := camera.Position, orbit.Target
	state.StickY = 1
	orbit.Update(state, 0.5, &camera)
	if groundDistance(eye, orbit.Target) <= groundDistance(eye, before) {
		t.Fatalf("the target went from %v to %v: not away from the camera at %v", before, orbit.Target, eye)
	}
}

func TestTheStickRightMovesThePlanRight(t *testing.T) {
	plan := NewPlan(graphics.Vec4{0, 0, 0, 1}, 100)
	var camera models.Camera
	state := input.NewState()
	state.WindowWidth, state.WindowHeight = 400, 240
	plan.Update(state, 0.1, &camera)
	before := plan.Target
	state.StickX = 1
	plan.Update(state, 0.5, &camera)
	if plan.Target == before {
		t.Fatalf("the stick did not move the plan from %v", before)
	}
}
