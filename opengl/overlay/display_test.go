package overlay

import (
	"testing"

	"github.com/akrck02/go-renderer/scene"
	"golang.org/x/image/font/gofont/gomonobold"
)

// A label asks for the display font; the others keep the labels' own.
func TestALabelCanBeWrittenInTheDisplayFont(test *testing.T) {
	if style := labelStyleFor(&scene.Label{Text: "North wing"}); style.face != textFace {
		test.Fatalf("a plain label is written in face %v, want the labels' own %v", style.face, textFace)
	}
	if style := labelStyleFor(&scene.Label{Text: "North wing", Display: true}); style.face != displayFace {
		test.Fatalf("a display label is written in face %v, want the display face %v", style.face, displayFace)
	}
}

// The display font is the one chosen, and without one the labels' own font. A monospaced font
// tells them apart: its i is as wide as its M.
func TestTheDisplayFontFallsBackToTheLabelsFont(test *testing.T) {
	plain, err := loadFonts(Fonts{})
	if err != nil {
		test.Fatal(err)
	}
	if isMonospaced(plain, displayFace) {
		test.Fatal("without a display font, the display face should be the labels' own, proportional font")
	}
	chosen, err := loadFonts(Fonts{Display: gomonobold.TTF})
	if err != nil {
		test.Fatal(err)
	}
	if !isMonospaced(chosen, displayFace) || isMonospaced(chosen, textFace) {
		test.Fatal("the chosen (monospaced) display font should be the display face, and only it")
	}
}

func isMonospaced(loaded *fonts, name faceName) bool {
	face := loaded.face(name, 14)
	narrow, _ := face.GlyphAdvance('i')
	wide, _ := face.GlyphAdvance('M')
	return narrow == wide
}
