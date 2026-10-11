package scene

import (
	"math"
	"testing"
)

func TestLevelVariationFollowsTheHarmonic(t *testing.T) {
	variation := &LevelVariation{MinimumX: 0, MinimumZ: 0, MaximumX: 2, MaximumZ: 1, Columns: 2, Rows: 1,
		InPhase: []float32{1, 3}, InQuadrature: []float32{0, 0}}
	water := &Water{Level: 10, Variation: variation}
	if level := water.LevelAt(0.5, 0.5); math.Abs(float64(level-11)) > 1e-5 {
		t.Fatalf("at the first cell centre with angle 0 the level should be 11, got %v", level)
	}
	if level := water.LevelAt(1, 0.5); math.Abs(float64(level-12)) > 1e-5 {
		t.Fatalf("between the cells the offset should be interpolated, got %v", level)
	}
	variation.Angle = math.Pi
	if level := water.LevelAt(-50, 0.5); math.Abs(float64(level-9)) > 1e-5 {
		t.Fatalf("outside the area the edge applies, and half a turn inverts it: got %v", level)
	}
}
