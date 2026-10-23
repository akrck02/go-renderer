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

func TestLettersThatFitTogetherAreMeasuredCloser(t *testing.T) {
	fonts, err := SystemFonts()
	if err != nil {
		t.Skip("no system font to try kerning with")
	}
	measurer, err := NewMeasurer(fonts)
	if err != nil {
		t.Fatal(err)
	}
	loaded := measurer.fonts
	face := loaded.face(regularFace, 40)
	kern := toPixels(face.Kern('A', 'V'))
	if kern == 0 {
		t.Skip("the system font has no kerning for «AV»")
	}
	together := measurer.MeasureText("AV", 40, 0, 1).Width
	apart := measurer.MeasureText("A", 40, 0, 1).Width + measurer.MeasureText("V", 40, 0, 1).Width
	if together != apart+kern {
		t.Errorf("«AV» is %v wide; its letters apart are %v, and the font kerns them by %v", together, apart, kern)
	}
}
