package camera

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
)

func near(first, second float32) bool { return math.Abs(float64(first-second)) < 1e-4 }

func TestThePlanLooksStraightDownWithNorthUp(test *testing.T) {
	plan := NewPlan(graphics.Vector{2, 0, 3, 1}, 10)
	var view Camera
	plan.Update(nil, 0, &view)
	if !near(view.Position[0], 2) || !near(view.Position[2], 3) || view.Position[1] <= 0 {
		test.Errorf("camera at %v, want straight above the target (2, ·, 3)", view.Position)
	}
	if !near(view.Up[0], 0) || !near(view.Up[1], 0) || !near(view.Up[2], -1) {
		test.Errorf("screen up %v, want north (−z)", view.Up)
	}
	if !view.IsOrthographic() || view.OrthographicHeight != 10 {
		test.Errorf("orthographic height %v, want the span 10", view.OrthographicHeight)
	}
	if view.Far <= float32(plan.Distance) || view.Near <= 0 {
		test.Errorf("near %v and far %v must hold the target, %v away", view.Near, view.Far, plan.Distance)
	}
}

func TestTheIsometricViewIsAt35DegreesAnd45ToTheAxes(test *testing.T) {
	plan := NewPlan(graphics.Vector{0, 0, 0, 1}, 10)
	plan.Isometric = true
	var view Camera
	plan.Update(nil, 0, &view)
	horizontal := math.Hypot(float64(view.Position[0]), float64(view.Position[2]))
	elevation := math.Atan2(float64(view.Position[1]), horizontal) * 180 / math.Pi
	if math.Abs(elevation-35.264) > 0.01 {
		test.Errorf("elevation %.3f°, want 35.264°", elevation)
	}
	if !near(float32(math.Abs(float64(view.Position[0]))), float32(math.Abs(float64(view.Position[2])))) {
		test.Errorf("camera at %v, want it at 45° to the axes", view.Position)
	}
	forward := graphics.Vector{-view.Position[0], -view.Position[1], -view.Position[2], 0}
	if dot := forward[0]*view.Up[0] + forward[1]*view.Up[1] + forward[2]*view.Up[2]; !near(dot, 0) {
		test.Errorf("screen up %v is not square to the line of sight", view.Up)
	}
}

func TestDraggingMovesTheMapWithTheCursor(test *testing.T) {
	plan := NewPlan(graphics.Vector{0, 0, 0, 1}, 10)
	state := input.NewState()
	state.WindowHeight = 100
	state.Feed().SetButton(input.MouseLeft, true)
	state.CursorDeltaX = 10 // a tenth of the window to the right: the map follows, so the centre moves west
	var view Camera
	plan.Update(state, 0, &view)
	if !near(plan.Target[0], -1) || !near(plan.Target[2], 0) {
		test.Errorf("target %v, want 1 unit west", plan.Target)
	}
	state.CursorDeltaX, state.CursorDeltaY = 0, 10 // down the window: the centre moves north
	plan.Update(state, 0, &view)
	if !near(plan.Target[0], -1) || !near(plan.Target[2], -1) {
		test.Errorf("target %v, want 1 unit north as well", plan.Target)
	}
}

func TestScrollZoomsAndQuarterTurnsRotate(test *testing.T) {
	plan := NewPlan(graphics.Vector{0, 0, 0, 1}, 10)
	state := input.NewState()
	state.Feed().AddScroll(1)
	var view Camera
	plan.Update(state, 0, &view)
	if plan.Span >= 10 {
		test.Errorf("span %v after scrolling in, want less than 10", plan.Span)
	}
	plan.Span = 1e9
	plan.Update(nil, 0, &view)
	if plan.Span != plan.MaximumSpan {
		test.Errorf("span %v, want it clamped to %v", plan.Span, plan.MaximumSpan)
	}
	state = input.NewState()
	state.Feed().SetKey(input.KeyE, true)
	plan.Update(state, 0, &view)
	if plan.Turns != 1 || !near(view.Up[0], 1) || !near(view.Up[2], 0) {
		test.Errorf("after E: %d turns, screen up %v; want east up", plan.Turns, view.Up)
	}
}
