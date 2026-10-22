package ctr

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
)

func TestPICAMatrixGoesRowByRowWithComponentsReversed(test *testing.T) {
	translation := graphics.Translate(graphics.Vector{10, 20, 30, 1})
	laidOut := PICAMatrix(translation)
	// The first row is (1, 0, 0, 10): reversed, w first.
	if laidOut[0] != 10 || laidOut[3] != 1 {
		test.Fatalf("first row %v, want [10 0 0 1]", laidOut[0:4])
	}
	if laidOut[4] != 20 || laidOut[6] != 1 || laidOut[8] != 30 || laidOut[9] != 1 || laidOut[12] != 1 {
		test.Fatalf("laid out %v", laidOut)
	}
}

func TestOrthographicTiltPutsTheScreenCornersOnTheClipCorners(test *testing.T) {
	projection := OrthographicTilt(0, 400, 0, 240, 0, 1)
	cases := []struct {
		point      graphics.Vector
		clipX      float32
		clipY      float32
		clipZ      float32
		screenName string
	}{
		// The quarter turn: screen x runs along clip y, and screen y along clip x. Depth goes
		// from -1 at the near plane to 0 at the far one (looking down -z).
		{graphics.Vector{0, 0, 0, 1}, -1, 1, -1, "bottom left, near"},
		{graphics.Vector{400, 240, 0, 1}, 1, -1, -1, "top right, near"},
		{graphics.Vector{200, 120, -1, 1}, 0, 0, 0, "centre, far"},
	}
	for _, testCase := range cases {
		clip := projection.Apply(testCase.point)
		if !near(clip[0], testCase.clipX) || !near(clip[1], testCase.clipY) || !near(clip[2], testCase.clipZ) || clip[3] != 1 {
			test.Errorf("%s: clip %v, want (%v, %v, %v, 1)", testCase.screenName, clip, testCase.clipX, testCase.clipY, testCase.clipZ)
		}
	}
}

func near(value, want float32) bool {
	difference := value - want
	return difference < 1e-5 && difference > -1e-5
}
