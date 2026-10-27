//go:build !(js && wasm)

package overlay

import (
	"image"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
)

// systemEmojiAdvance is how far the fake platform's emoji move the pen, in pixels at size 16.
const systemEmojiAdvance = 20

// withSystemEmoji sets a fake platform drawer for one test, which has every emoji-first character
// and counts what it is asked.
func withSystemEmoji(test *testing.T) *int {
	test.Helper()
	asked := 0
	UseSystemEmoji(func(character rune, pixels int) (image.Image, float32, float32, float32, bool) {
		asked++
		if !emojiFirst(character) {
			return nil, 0, 0, 0, false
		}
		return image.NewNRGBA(image.Rect(0, 0, pixels, pixels)), 0, float32(pixels), systemEmojiAdvance, true
	})
	test.Cleanup(func() { UseSystemEmoji(nil) })
	return &asked
}

// Where a phone's font file has the emoji (Android 14 and earlier), it is used as before, without
// the cost of asking the platform.
func TestTheFontFilesEmojiWinsOverTheSystems(test *testing.T) {
	asked := withSystemEmoji(test)
	fonts := withFontFolder(test, map[string][]byte{"NotoColorEmojiLegacy.ttf": colourTestFont(test)})
	_, systemEmoji := systemFontsIn(fonts.Folders)
	glyph, found := systemEmoji.glyph(wave, 16)
	if !found || glyph.advance == systemEmojiAdvance {
		test.Fatal("the wave comes from the font file")
	}
	if *asked != 0 {
		test.Fatalf("the platform was asked %d times for an emoji the font file has", *asked)
	}
}

// Asking the platform costs a call into it: an emoji is asked once per size, and a frame that shows
// it again asks nothing and allocates nothing.
func TestAnEmojiTheFontsLackIsAskedOfTheSystemOnce(test *testing.T) {
	asked := withSystemEmoji(test)
	_, systemEmoji := systemFontsIn([]string{test.TempDir()})
	if _, found := systemEmoji.glyph('😀', 16); !found {
		test.Fatal("the platform draws the face")
	}
	allocations := testing.AllocsPerRun(10, func() { systemEmoji.glyph('😀', 16) })
	if *asked != 1 || allocations != 0 {
		test.Fatalf("asked %d times, %v allocations in a later frame; want once and none", *asked, allocations)
	}
}

// Text is laid out from its measure, so an emoji the platform draws is measured as it is drawn.
func TestATextWithASystemEmojiIsAsWideAsItsParts(test *testing.T) {
	withSystemEmoji(test)
	measurer, err := NewMeasurer(Fonts{Folders: []string{test.TempDir()}})
	if err != nil {
		test.Fatal(err)
	}
	width := func(text string) float32 { return measurer.MeasureText(text, 16, draw2d.Regular, 1).Width }
	if whole, parts := width("a😀b"), width("a")+systemEmojiAdvance+width("b"); whole != parts {
		test.Fatalf(`"a😀b" measures %v, its parts %v`, whole, parts)
	}
}

// With neither a font file nor the platform to draw it, the text keeps its question mark and
// nothing fails, as before.
func TestWithoutEitherTheCharacterIsDrawnAsAQuestionMark(test *testing.T) {
	UseSystemEmoji(func(rune, int) (image.Image, float32, float32, float32, bool) { return nil, 0, 0, 0, false })
	test.Cleanup(func() { UseSystemEmoji(nil) })
	measurer, err := NewMeasurer(Fonts{Folders: []string{test.TempDir()}})
	if err != nil {
		test.Fatal(err)
	}
	if emoji, question := measurer.MeasureText("😀", 16, draw2d.Regular, 1).Width, measurer.MeasureText("?", 16, draw2d.Regular, 1).Width; emoji != question {
		test.Fatalf("the face measures %v, a question mark %v", emoji, question)
	}
}
