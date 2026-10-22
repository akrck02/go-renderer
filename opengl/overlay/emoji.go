package overlay

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/akrck02/go-renderer/draw2d"
	scaling "golang.org/x/image/draw"
)

// Emoji are drawn in their own colours from the system's colour emoji font (Android's
// NotoColorEmoji), as pictures in the picture atlas, in the same draw call as the text. Each is
// scaled once to each size it is drawn at, so it stays sharp and does not shimmer.

// emojiFont is the system's colour emoji fonts in some folders, read the first time an emoji is
// drawn (see systemFontsIn).
type emojiFont struct {
	mutex    sync.Mutex
	folders  []string
	fonts    []*colourFont // every colour font read, the plainest first
	decoded  map[rune]image.Image
	glyphs   map[emojiKey]*emojiGlyph // nil for a character the font has no picture of
	searched bool
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

// glyph returns an emoji's picture at a size in pixels, if the system's colour font has it.
func (emoji *emojiFont) glyph(character rune, size int) (*emojiGlyph, bool) {
	emoji.mutex.Lock()
	defer emoji.mutex.Unlock()
	emoji.find()
	if len(emoji.fonts) == 0 {
		return nil, false
	}
	key := emojiKey{character, size}
	if glyph, made := emoji.glyphs[key]; made {
		return glyph, glyph != nil
	}
	glyph := emoji.make(character, size)
	emoji.glyphs[key] = glyph
	return glyph, glyph != nil
}

// find reads the system's colour emoji fonts, the first time.
// Android 13 and later draw their own emoji from vector outlines (COLRv1), which are not read
// here; they keep the pictures too, in NotoColorEmojiLegacy, for programs like this one.
func (emoji *emojiFont) find() {
	if emoji.searched {
		return
	}
	emoji.searched = true
	emoji.fonts, emoji.decoded, emoji.glyphs = nil, map[rune]image.Image{}, map[emojiKey]*emojiGlyph{}
	for _, path := range colourEmojiFiles(emoji.folders) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if font, err := parseColourFont(data); err == nil {
			emoji.fonts = append(emoji.fonts, font)
		}
	}
}

// colourEmojiFiles are the font files named as colour emoji, the whole set first, the one for
// flags alone last.
func colourEmojiFiles(folders []string) []string {
	var found []string
	for _, folder := range folders {
		filepath.WalkDir(folder, func(path string, entry os.DirEntry, err error) error {
			name := plainName(path)
			if err == nil && !entry.IsDir() && strings.Contains(name, "emoji") && (strings.Contains(name, "color") || strings.Contains(name, "colour")) {
				found = append(found, path)
			}
			return nil
		})
	}
	slices.SortStableFunc(found, func(first, second string) int {
		return emojiFilePenalty(plainName(first)) - emojiFilePenalty(plainName(second))
	})
	return found
}

func emojiFilePenalty(name string) int {
	penalty := 0
	if strings.Contains(name, "flags") {
		penalty++
	}
	return penalty
}

// make decodes an emoji's PNG, once, from the first font that has it, and scales it to a size.
func (emoji *emojiFont) make(character rune, size int) *emojiGlyph {
	var picture colourPicture
	found := false
	for _, font := range emoji.fonts {
		if picture, found = font.picture(character); found {
			break
		}
	}
	if !found {
		return nil
	}
	decoded, known := emoji.decoded[character]
	if !known {
		var err error
		if decoded, err = picture.decode(); err != nil {
			return nil
		}
		emoji.decoded[character] = decoded
	}
	pixels := float32(size)
	width, height := max(int(picture.width*pixels+0.5), 1), max(int(picture.height*pixels+0.5), 1)
	scaled := image.NewNRGBA(image.Rect(0, 0, width, height))
	scaling.CatmullRom.Scale(scaled, scaled.Bounds(), decoded, decoded.Bounds(), scaling.Src, nil)
	return &emojiGlyph{picture: draw2d.NewImage(scaled), left: picture.left * pixels, top: picture.top * pixels,
		advance: picture.advance * pixels, pixelWidth: width, pixelHeight: height}
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
