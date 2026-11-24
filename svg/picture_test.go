package svg

import (
	"math"
	"testing"
)

// opacityAt is how covered a pixel of a drawing's picture is, from 0 to 1.
func opacityAt(test *testing.T, source string, width, height, x, y int) float64 {
	test.Helper()
	picture := mustRead(test, source).Picture(width, height)
	return float64(picture.Pixels.NRGBAAt(x, y).A) / 255
}

func TestASquareHalfOverAPixelCoversHalfOfIt(test *testing.T) {
	source := `<svg viewBox="0 0 4 4"><rect width="1.5" height="4"/></svg>`
	for x, want := range []float64{1, 0.5, 0} {
		if got := opacityAt(test, source, 4, 4, x, 1); math.Abs(got-want) > 0.04 {
			test.Errorf("pixel %d is %.2f covered, want %.2f", x, got, want)
		}
	}
}

func TestARingIsFilledAsEachRuleSays(test *testing.T) {
	// two squares drawn the same way round: non-zero fills the inner one too, even-odd leaves a hole
	ring := `M0 0 H8 V8 H0 Z M2 2 H6 V6 H2 Z`
	nonZero := `<svg viewBox="0 0 8 8"><path d="` + ring + `"/></svg>`
	evenOdd := `<svg viewBox="0 0 8 8"><path fill-rule="evenodd" d="` + ring + `"/></svg>`
	if opacityAt(test, nonZero, 8, 8, 4, 4) < 0.99 || opacityAt(test, nonZero, 8, 8, 1, 1) < 0.99 {
		test.Error("non-zero leaves part of the two squares empty")
	}
	if opacityAt(test, evenOdd, 8, 8, 4, 4) > 0.01 || opacityAt(test, evenOdd, 8, 8, 1, 1) < 0.99 {
		test.Error("even-odd does not leave the inner square as a hole")
	}
	// the inner square drawn the other way round: now non-zero leaves the hole too
	turned := `<svg viewBox="0 0 8 8"><path d="M0 0 H8 V8 H0 Z M2 2 V6 H6 V2 Z"/></svg>`
	if opacityAt(test, turned, 8, 8, 4, 4) > 0.01 || opacityAt(test, turned, 8, 8, 1, 1) < 0.99 {
		test.Error("non-zero fills an inner square drawn the other way round")
	}
}

// The view box fills the picture as SVG's default says: scaled to fit, its middle in the middle.
func TestTheViewBoxIsScaledToFitAndCentred(test *testing.T) {
	quarter := `<svg viewBox="0 0 24 24"><rect width="12" height="12"/></svg>`
	if opacityAt(test, quarter, 48, 48, 23, 23) < 0.99 || opacityAt(test, quarter, 48, 48, 24, 24) > 0.01 {
		test.Error("a quarter of the view box does not cover a quarter of a picture twice its size")
	}
	whole := `<svg viewBox="0 0 10 10"><rect width="10" height="10"/></svg>`
	if opacityAt(test, whole, 40, 20, 5, 10) > 0.01 || opacityAt(test, whole, 40, 20, 20, 10) < 0.99 {
		test.Error("a square drawing in a wide picture is not drawn in its middle")
	}
}

// Shapes are painted in order, each over the ones before, in their own colours.
func TestShapesArePaintedInOrderInTheirColours(test *testing.T) {
	source := `<svg viewBox="0 0 2 2"><rect width="2" height="2" fill="#ff0000"/>` +
		`<rect width="1" height="2" fill="#0000ff" fill-opacity="0.5"/></svg>`
	picture := mustRead(test, source).Picture(2, 2)
	red, mixed := picture.Pixels.NRGBAAt(1, 0), picture.Pixels.NRGBAAt(0, 0)
	if red.R != 255 || red.B != 0 || red.A != 255 {
		test.Errorf("the red square alone is %v", red)
	}
	if mixed.A != 255 || math.Abs(float64(mixed.R)-127.5) > 2 || math.Abs(float64(mixed.B)-127.5) > 2 {
		test.Errorf("half blue over red is %v, want an even mix", mixed)
	}
}
