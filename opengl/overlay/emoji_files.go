package overlay

import (
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/akrck02/go-renderer/draw2d"
	scaling "golang.org/x/image/draw"
)

// newFontFileEmoji reads the colour emoji fonts in these folders, when the first emoji is drawn.
func newFontFileEmoji(folders []string) *fontFileEmoji {
	return &fontFileEmoji{folders: slices.Clone(folders)}
}

// fontFileEmoji is the system's colour emoji fonts in some folders, read the first time an emoji
// is drawn.
type fontFileEmoji struct {
	folders  []string
	fonts    []*colourFont // every colour font read, the plainest first
	decoded  map[rune]image.Image
	searched bool
}

// emoji decodes an emoji's picture from the first colour font that has it, at a size in pixels.
func (files *fontFileEmoji) emoji(character rune, size int) (*emojiGlyph, bool) {
	files.find()
	glyph := files.make(character, size)
	return glyph, glyph != nil
}

// find reads the system's colour emoji fonts, the first time.
// Android 13 and later draw their own emoji from vector outlines (COLRv1), which are not read
// here; they keep the pictures too, in NotoColorEmojiLegacy, for programs like this one.
func (files *fontFileEmoji) find() {
	if files.searched {
		return
	}
	files.searched = true
	files.fonts, files.decoded = nil, map[rune]image.Image{}
	for _, path := range colourEmojiFiles(files.folders) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if font, err := parseColourFont(data); err == nil {
			files.fonts = append(files.fonts, font)
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
			if err == nil && !entry.IsDir() && hasFontExtension(path) && strings.Contains(name, "emoji") && (strings.Contains(name, "color") || strings.Contains(name, "colour")) {
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
func (files *fontFileEmoji) make(character rune, size int) *emojiGlyph {
	var picture colourPicture
	found := false
	for _, font := range files.fonts {
		if picture, found = font.picture(character); found {
			break
		}
	}
	if !found {
		return nil
	}
	decoded, known := files.decoded[character]
	if !known {
		var err error
		if decoded, err = picture.decode(); err != nil {
			return nil
		}
		files.decoded[character] = decoded
	}
	pixels := float32(size)
	width, height := max(int(picture.width*pixels+0.5), 1), max(int(picture.height*pixels+0.5), 1)
	scaled := image.NewNRGBA(image.Rect(0, 0, width, height))
	scaling.CatmullRom.Scale(scaled, scaled.Bounds(), decoded, decoded.Bounds(), scaling.Src, nil)
	return &emojiGlyph{picture: draw2d.NewImage(scaled), left: picture.left * pixels, top: picture.top * pixels,
		advance: picture.advance * pixels, pixelWidth: width, pixelHeight: height}
}
