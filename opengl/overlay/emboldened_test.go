package overlay

import (
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

// A variable font has one file for every weight, and x/image draws only its default one: the bold
// must still look bold, or headings and strong words read as plain text.
func TestABoldThatIsTheRegularsFileIsDrawnBolder(test *testing.T) {
	const size = 24
	plain, err := loadFonts(Fonts{Regular: goregular.TTF, Bold: append([]byte(nil), goregular.TTF...), Folders: []string{}})
	if err != nil {
		test.Fatal(err)
	}
	regular := plain.face(regularFace, size)
	bold := plain.face(strongFace, size)
	regularAdvance, _ := regular.GlyphAdvance('H')
	boldAdvance, _ := bold.GlyphAdvance('H')
	if boldAdvance <= regularAdvance {
		test.Errorf("the made-up bold H advances %v, not more than the regular's %v", boldAdvance, regularAdvance)
	}
	regularMask, _ := rasteriseGlyph(regular, 'l')
	boldMask, _ := rasteriseGlyph(bold, 'l')
	if inked(boldMask.Pix) <= inked(regularMask.Pix) {
		test.Errorf("the made-up bold l is not inked more than the regular one")
	}
}

// A font's own bold file is drawn as it is: only a missing weight is made up.
func TestABoldOfItsOwnIsNotEmboldened(test *testing.T) {
	loaded, err := loadFonts(Fonts{Folders: []string{}})
	if err != nil {
		test.Fatal(err)
	}
	if _, madeUp := loaded.face(strongFace, 16).(*emboldened); madeUp {
		test.Error("Go Bold, a bold of its own, is drawn emboldened")
	}
}

// inked is how much ink a glyph's mask holds.
func inked(pixels []uint8) int {
	total := 0
	for _, pixel := range pixels {
		total += int(pixel)
	}
	return total
}
