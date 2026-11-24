package overlay

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// emboldeningPerEm is how much wider a synthetic bold makes each stroke, as a part of the font's
// size: FreeType's FT_GlyphSlot_Embolden widens by a 24th of the em, and browsers draw a bold
// the font lacks much the same way.
const emboldeningPerEm = 1.0 / 24

// emboldened is a face drawn bolder than it is: each glyph's strokes widened to the right, and
// its advance by as much, so the letters keep their spacing. It stands in for a bold weight the
// font file cannot give, as a variable font's, which x/image draws only at its default weight.
type emboldened struct {
	font.Face
	// strength is how many pixels wider each stroke is drawn
	strength fixed.Int26_6
}

// newEmboldened is face drawn bolder, for a font of size pixels.
func newEmboldened(face font.Face, size int) *emboldened {
	return &emboldened{Face: face, strength: fixed.Int26_6(float64(size) * emboldeningPerEm * 64)}
}

// Glyph is the face's glyph, its mask widened by the strength.
func (face *emboldened) Glyph(dot fixed.Point26_6, character rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	bounds, mask, maskPoint, advance, found := face.Face.Glyph(dot, character)
	if !found || bounds.Empty() {
		return bounds, mask, maskPoint, advance + face.strength, found
	}
	widened := widen(mask, maskPoint, bounds.Size(), face.strength)
	bounds.Max.X = bounds.Min.X + widened.Bounds().Dx()
	return bounds, widened, image.Point{}, advance + face.strength, true
}

// GlyphBounds are the glyph's bounds, as wide as Glyph draws it.
func (face *emboldened) GlyphBounds(character rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	bounds, advance, found := face.Face.GlyphBounds(character)
	bounds.Max.X += face.strength
	return bounds, advance + face.strength, found
}

// GlyphAdvance is the glyph's advance, wider by the strength.
func (face *emboldened) GlyphAdvance(character rune) (fixed.Int26_6, bool) {
	advance, found := face.Face.GlyphAdvance(character)
	return advance + face.strength, found
}

// widen copies a glyph's mask, each pixel covering as much of the strength's pixels to its right
// as it covers itself: whole pixels fully, the last one in part.
func widen(mask image.Image, from image.Point, size image.Point, strength fixed.Int26_6) *image.Alpha {
	whole := int(strength >> 6)
	part := float64(strength&63) / 64
	extra := whole
	if part > 0 {
		extra++
	}
	widened := image.NewAlpha(image.Rect(0, 0, size.X+extra, size.Y))
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			_, _, _, alpha := mask.At(from.X+x, from.Y+y).RGBA()
			covering := uint8(alpha >> 8)
			for step := 0; step <= extra; step++ {
				amount := covering
				if step == extra && part > 0 {
					amount = uint8(float64(covering) * part)
				}
				index := widened.PixOffset(x+step, y)
				widened.Pix[index] = max(widened.Pix[index], amount)
			}
		}
	}
	return widened
}
