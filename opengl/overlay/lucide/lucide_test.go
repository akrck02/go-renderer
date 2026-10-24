package lucide

import (
	"testing"

	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Lucide keeps the names of icons it has withdrawn (the brands' logos) with an empty drawing; an
// icon the table offers must draw something, or a program asking HasIcon is told a lie.
func TestEveryIconTheTableNamesHasADrawing(test *testing.T) {
	parsed, err := sfnt.Parse(Font())
	if err != nil {
		test.Fatal(err)
	}
	var buffer sfnt.Buffer
	for name, character := range codepoints {
		index, err := parsed.GlyphIndex(&buffer, character)
		if err != nil || index == 0 {
			test.Errorf("%q has no character in the font", name)
			continue
		}
		segments, err := parsed.LoadGlyph(&buffer, index, fixed.I(drawingSize), nil)
		if err != nil || len(segments) == 0 {
			test.Errorf("%q is drawn empty", name)
		}
	}
}

const drawingSize = 24 // pixels; any size shows whether a drawing is empty
