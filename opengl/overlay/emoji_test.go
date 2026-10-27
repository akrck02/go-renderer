package overlay

import (
	"image"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
)

// countingEmoji is a source with a square picture of every emoji, which counts what it is asked.
type countingEmoji struct {
	made int
}

func (source *countingEmoji) emoji(character rune, size int) (*emojiGlyph, bool) {
	if !emojiFirst(character) {
		return nil, false
	}
	source.made++
	picture := draw2d.NewImage(image.NewNRGBA(image.Rect(0, 0, size, size)))
	return &emojiGlyph{picture: picture, top: float32(size), advance: float32(size), pixelWidth: size, pixelHeight: size}, true
}

// Making an emoji means drawing or decoding a picture: each must be made once per size, not in
// every frame that shows it.
func TestAnEmojiIsMadeOncePerSize(test *testing.T) {
	source := &countingEmoji{}
	emoji := newEmojiFont(source)
	for range 3 {
		if _, found := emoji.glyph(wave, 16); !found {
			test.Fatal("the source has the wave")
		}
	}
	if _, found := emoji.glyph(wave, 32); !found {
		test.Fatal("the source has the wave at every size")
	}
	if source.made != 2 {
		test.Fatalf("the wave was made %d times for two sizes", source.made)
	}
	if _, found := emoji.glyph('a', 16); found {
		test.Fatal("a letter is no emoji")
	}
}

// Without a source, nothing is an emoji, and nothing fails: the text keeps its question marks.
func TestWithoutASourceNoCharacterIsAnEmoji(test *testing.T) {
	if _, found := newEmojiFont(nil).glyph(wave, 16); found {
		test.Fatal("an emoji was found with nowhere to look")
	}
}
