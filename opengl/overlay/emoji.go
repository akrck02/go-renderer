package overlay

import (
	"sync"
	"unicode"

	"github.com/akrck02/go-renderer/draw2d"
)

// Emoji are drawn in their own colours, as pictures in the picture atlas, in the same draw call as
// the text: from the system's colour emoji font (Android's NotoColorEmoji), or drawn by the browser
// in a page. Each is made once at each size it is drawn at, so it stays sharp and does not shimmer.

// emojiFont keeps the emoji drawn so far, each at each size it was drawn at, and asks its source
// for the ones it has not made yet (see systemFontsIn).
type emojiFont struct {
	mutex  sync.Mutex
	source emojiSource              // where the pictures come from; nil when nowhere
	glyphs map[emojiKey]*emojiGlyph // nil for a character the source has no picture of
}

// emojiSource makes an emoji's picture at a size in pixels: from the system's colour font files,
// or drawn by a browser. It returns false when it has no picture of the character. It is only
// called with the emojiFont's mutex held, so it needs no lock of its own.
type emojiSource interface {
	emoji(character rune, size int) (*emojiGlyph, bool)
}

type emojiKey struct {
	character rune
	size      int
}

// emojiGlyph is an emoji at one size: its picture, and where it sits in pixels: left of the pen,
// top above the baseline, and the pen's move after it.
type emojiGlyph struct {
	picture                 *draw2d.Image
	left, top, advance      float32
	pixelWidth, pixelHeight int
}

// newEmojiFont keeps the emoji a source makes.
func newEmojiFont(source emojiSource) *emojiFont {
	return &emojiFont{source: source, glyphs: map[emojiKey]*emojiGlyph{}}
}

// glyph returns an emoji's picture at a size in pixels, if the source has it. Each is made once:
// a frame that shows emoji already made allocates nothing.
func (emoji *emojiFont) glyph(character rune, size int) (*emojiGlyph, bool) {
	emoji.mutex.Lock()
	defer emoji.mutex.Unlock()
	if emoji.source == nil {
		return nil, false
	}
	key := emojiKey{character, size}
	if glyph, made := emoji.glyphs[key]; made {
		return glyph, glyph != nil
	}
	glyph, found := emoji.source.emoji(character, size)
	if !found {
		glyph = nil
	}
	emoji.glyphs[key] = glyph
	return glyph, found
}

// emojiFirst tells whether a character is shown as a colour emoji before any font's outline of
// it: the pictographs, faces, transport and the newer symbols, which are emoji by default.
func emojiFirst(character rune) bool {
	switch {
	case character >= 0x1f300 && character <= 0x1f6ff, // pictographs, faces, transport
		character >= 0x1f900 && character <= 0x1faff, // supplemental and newer
		character >= 0x1f1e6 && character <= 0x1f1ff: // regional indicators
		return true
	}
	return false
}

// emojiPresentation is the variation selector asking for the character before it as an emoji.
const emojiPresentation = 0xfe0f

// askedAsEmoji tells whether the character at an index is followed by the selector asking for it
// as an emoji (❤️).
func askedAsEmoji(text []rune, index int) bool {
	return index+1 < len(text) && text[index+1] == emojiPresentation
}

// invisible characters steer how emoji and scripts join (a variation selector asking for the
// emoji picture, a zero width joiner) and are not drawn themselves.
func invisible(character rune) bool {
	return character == 0x200d || unicode.Is(unicode.Variation_Selector, character)
}
