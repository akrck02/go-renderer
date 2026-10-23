package overlay

import (
	"math"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func TestALineOfTextIsAsTallAsABrowserMakesIt(t *testing.T) {
	loaded, err := loadFonts(Fonts{})
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{13, 14, 17, 19, 34} {
		exact, err := loaded.parsed[regularFace].Metrics(nil, fixed.I(size), font.HintingNone)
		if err != nil {
			t.Fatal(err)
		}
		measured := loaded.metrics(regularFace, size)
		wantAscent, wantDescent := math.Round(float64(exact.Ascent)/64), math.Round(float64(exact.Descent)/64)
		if float64(measured.ascent) != wantAscent || float64(measured.descent) != wantDescent {
			t.Errorf("at %d px the ascent and descent are %v and %v, not the font's own rounded: %v and %v",
				size, measured.ascent, measured.descent, wantAscent, wantDescent)
		}
	}
}
