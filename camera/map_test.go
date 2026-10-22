package camera

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
)

func near(first, second float32) bool { return math.Abs(float64(first-second)) < 1e-4 }

func TestTheMapLooksStraightDownWithNorthUp(test *testing.T) {
	view := NewMap(graphics.Vector{2, 0, 3, 1}, 10)
	var seen Camera
	view.Update(nil, 0, &seen)
	if !near(seen.Position[0], 2) || !near(seen.Position[2], 3) || seen.Position[1] <= 0 {
		test.Errorf("camera at %v, want straight above the target (2, ·, 3)", seen.Position)
	}
	if !near(seen.Up[0], 0) || !near(seen.Up[1], 0) || !near(seen.Up[2], -1) {
		test.Errorf("screen up %v, want north (−z)", seen.Up)
	}
	if !seen.IsOrthographic() || seen.OrthographicHeight != 10 {
		test.Errorf("orthographic height %v, want the span 10", seen.OrthographicHeight)
	}
	if seen.Far <= float32(view.Distance) || seen.Near <= 0 {
		test.Errorf("near %v and far %v must hold the target, %v away", seen.Near, seen.Far, view.Distance)
	}
}

func TestTheIsometricViewIsAt35DegreesAnd45ToTheAxes(test *testing.T) {
	view := NewMap(graphics.Vector{0, 0, 0, 1}, 10)
	view.Isometric = true
	var seen Camera
	view.Update(nil, 0, &seen)
	horizontal := math.Hypot(float64(seen.Position[0]), float64(seen.Position[2]))
	elevation := math.Atan2(float64(seen.Position[1]), horizontal) * 180 / math.Pi
	if math.Abs(elevation-35.264) > 0.01 {
		test.Errorf("elevation %.3f°, want 35.264°", elevation)
	}
	if !near(float32(math.Abs(float64(seen.Position[0]))), float32(math.Abs(float64(seen.Position[2])))) {
		test.Errorf("camera at %v, want it at 45° to the axes", seen.Position)
	}
	forward := graphics.Vector{-seen.Position[0], -seen.Position[1], -seen.Position[2], 0}
	if dot := forward[0]*seen.Up[0] + forward[1]*seen.Up[1] + forward[2]*seen.Up[2]; !near(dot, 0) {
		test.Errorf("screen up %v is not square to the line of sight", seen.Up)
	}
}

func TestDraggingMovesTheMapWithTheCursor(test *testing.T) {
	view := NewMap(graphics.Vector{0, 0, 0, 1}, 10)
	state := input.NewState()
	state.WindowHeight = 100
	state.Feed().SetButton(input.MouseLeft, true)
	state.CursorDeltaX = 10 // a tenth of the window to the right: the map follows, so the centre moves west
	var seen Camera
	view.Update(state, 0, &seen)
	if !near(view.Target[0], -1) || !near(view.Target[2], 0) {
		test.Errorf("target %v, want 1 unit west", view.Target)
	}
	state.CursorDeltaX, state.CursorDeltaY = 0, 10 // down the window: the centre moves north
	view.Update(state, 0, &seen)
	if !near(view.Target[0], -1) || !near(view.Target[2], -1) {
		test.Errorf("target %v, want 1 unit north as well", view.Target)
	}
}

func TestScrollZoomsAndQuarterTurnsRotate(test *testing.T) {
	view := NewMap(graphics.Vector{0, 0, 0, 1}, 10)
	state := input.NewState()
	state.Feed().AddScroll(1)
	var seen Camera
	view.Update(state, 0, &seen)
	if view.Span >= 10 {
		test.Errorf("span %v after scrolling in, want less than 10", view.Span)
	}
	view.Span = 1e9
	view.Update(nil, 0, &seen)
	if view.Span != view.MaximumSpan {
		test.Errorf("span %v, want it clamped to %v", view.Span, view.MaximumSpan)
	}
	state = input.NewState()
	state.Feed().SetKey(input.KeyE, true)
	view.Update(state, 0, &seen)
	if view.Turns != 1 || !near(seen.Up[0], 1) || !near(seen.Up[2], 0) {
		test.Errorf("after E: %d turns, screen up %v; want east up", view.Turns, seen.Up)
	}
}
