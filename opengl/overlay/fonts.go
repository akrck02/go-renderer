package overlay

import (
	"image"
	"image/draw"
	"math"
	"strconv"
	"sync"

	"github.com/akrck02/go-renderer/failure"
	"github.com/akrck02/go-renderer/opengl/overlay/lucide"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// faceName says which font a text is written in.
type faceName int

const (
	textFace    faceName = iota // Go Medium: labels, badges, notices
	strongFace                  // Go Bold: the first line of the header
	iconFace                    // Lucide: icons
	regularFace                 // Go Regular: body text of drawings (interfaces)
	monoFace                    // Go Mono Bold: codes and keys (draw2d.Mono)
)

// Fonts are what a Painter or a Measurer draws text with: the TrueType or OpenType fonts, one per
// weight, and where the system's fonts are, for the characters they lack. The zero value is the
// default: the Go fonts and the system's usual folders.
type Fonts struct {
	// Regular, Medium, Bold and Mono are a font file's bytes each; an empty one keeps the Go font
	// of that weight.
	Regular, Medium, Bold, Mono []byte
	// Folders are where the system's fonts and colour emoji are looked for, for characters the
	// fonts lack. nil is DefaultFontFolders(); an empty, non-nil list looks nowhere, and such
	// characters are drawn as a question mark. In a browser the emoji come from the browser
	// itself, and Folders is not looked in for them.
	Folders []string
}

// folders returns the folders to look for system fonts in.
func (chosen Fonts) folders() []string {
	if chosen.Folders == nil {
		return DefaultFontFolders()
	}
	return chosen.Folders
}

func orDefault(chosen, fallback []byte) []byte {
	if len(chosen) > 0 {
		return chosen
	}
	return fallback
}

// prebuiltCharacters are drawn into the atlas as soon as a face is first used, so the first frames
// with text do not stop to draw them one by one: ASCII, Latin-1 and a few common signs. The list is
// made the first time it is needed, not when the package is imported.
var prebuiltCharacters = sync.OnceValue(listPrebuiltCharacters)

func listPrebuiltCharacters() []rune {
	var characters []rune
	for character := rune(0x20); character <= 0x7e; character++ {
		characters = append(characters, character)
	}
	for character := rune(0xa1); character <= 0xff; character++ {
		characters = append(characters, character)
	}
	return append(characters, '→', '←', '↑', '↓', '…', '–', '—', '€', '•')
}

// fonts parses the embedded fonts once and keeps a face for every size asked for.
type fonts struct {
	parsed map[faceName]*opentype.Font
	faces  map[faceKey]font.Face
	// fallbackFaces are faces of the system's fonts, for characters the program's fonts lack
	fallbackFaces map[fallbackKey]font.Face
	// system and emoji are the system's fonts and colour emoji in the chosen folders, shared with
	// every painter and measurer that looks in the same folders
	system *fallbackFonts
	emoji  *emojiFont
}

type fallbackKey struct {
	font *opentype.Font
	size int
}

type faceKey struct {
	name faceName
	size int
}

func loadFonts(chosen Fonts) (*fonts, error) {
	sources := map[faceName][]byte{textFace: orDefault(chosen.Medium, gomedium.TTF), strongFace: orDefault(chosen.Bold, gobold.TTF),
		iconFace: lucide.Font(), regularFace: orDefault(chosen.Regular, goregular.TTF), monoFace: orDefault(chosen.Mono, gomonobold.TTF)}
	system, emoji := systemFontsIn(chosen.folders())
	loaded := &fonts{parsed: map[faceName]*opentype.Font{}, faces: map[faceKey]font.Face{}, fallbackFaces: map[fallbackKey]font.Face{},
		system: system, emoji: emoji}
	for name, data := range sources {
		parsed, err := opentype.Parse(data)
		if err != nil {
			return nil, failure.Wrap(FontNotRead, "face "+strconv.Itoa(int(name)), err)
		}
		loaded.parsed[name] = parsed
	}
	return loaded, nil
}

// face returns a font at a size in pixels. Text is hinted to the pixel grid so it stays crisp at
// small sizes; icons are not, so their strokes keep their drawn proportions.
func (loaded *fonts) face(name faceName, size int) font.Face {
	key := faceKey{name, size}
	if face, found := loaded.faces[key]; found {
		return face
	}
	hinting := font.HintingFull
	if name == iconFace {
		hinting = font.HintingNone
	}
	face, err := opentype.NewFace(loaded.parsed[name], &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: hinting})
	if err != nil {
		// a parsed font accepts any positive size, so this only happens for a size of zero
		return nil
	}
	loaded.faces[key] = face
	return face
}

// faceFor returns the face that draws a character: the font's own, or, for a character it lacks,
// one of the system's fonts at the same size (not for icons). It is nil when none has it.
func (loaded *fonts) faceFor(name faceName, size int, character rune) font.Face {
	face := loaded.face(name, size)
	if face == nil {
		return nil
	}
	if _, found := face.GlyphAdvance(character); found || name == iconFace {
		return face
	}
	system := loaded.system.fontFor(character)
	if system == nil {
		return face
	}
	key := fallbackKey{system, size}
	if fallback, made := loaded.fallbackFaces[key]; made {
		return fallback
	}
	fallback, err := opentype.NewFace(system, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return face
	}
	loaded.fallbackFaces[key] = fallback
	return fallback
}

// colourFor returns the system's colour picture of an emoji, when it is drawn as one: an emoji by
// default (a face, a pictograph), or one asked for as an emoji (followed by U+FE0F), before any
// outline of it; another character only when no font has an outline of it. Icons never are.
func (loaded *fonts) colourFor(name faceName, size int, character rune, asked bool) (*emojiGlyph, bool) {
	if name == iconFace || (character < 0x2000 && !asked) {
		return nil, false // the common case, at no cost
	}
	if !asked && !emojiFirst(character) {
		face := loaded.faceFor(name, size, character)
		if face == nil {
			return nil, false
		}
		if _, drawn := face.GlyphAdvance(character); drawn {
			return nil, false
		}
	}
	return loaded.emoji.glyph(character, size)
}

// faceMetrics are the vertical measures of a face, in pixels.
type faceMetrics struct {
	ascent, descent, capHeight float32
}

func (loaded *fonts) metrics(name faceName, size int) faceMetrics {
	face := loaded.face(name, size)
	if face == nil {
		return faceMetrics{}
	}
	measures := face.Metrics()
	capHeight := toPixels(measures.CapHeight)
	if capHeight <= 0 {
		capHeight = toPixels(measures.Ascent) * 0.7
	}
	return faceMetrics{ascent: toPixels(measures.Ascent), descent: toPixels(measures.Descent), capHeight: capHeight}
}

// glyph returns the picture of a character in a face, drawing it into the atlas the first time.
// The entry's offset places the picture relative to the dot on the baseline.
// A glyph already drawn is found without making the closure that draws it: TinyGo keeps a
// closure's captured values on the heap, and text asks for every glyph every frame.
func (painter *Painter) glyph(name faceName, size int, character rune) Entry {
	key := entryKey{kind: glyphEntry, face: name, size: size, character: character}
	if entry, found := painter.atlas.entries[key]; found {
		return entry
	}
	return painter.drawGlyph(key)
}

func (painter *Painter) drawGlyph(key entryKey) Entry {
	return painter.atlas.lookup(key, func() (*image.Alpha, Entry) {
		return rasteriseGlyph(painter.fonts.faceFor(key.face, key.size, key.character), key.character)
	})
}

func rasteriseGlyph(face font.Face, character rune) (*image.Alpha, Entry) {
	if face == nil {
		return nil, Entry{}
	}
	bounds, mask, maskPoint, advance, found := face.Glyph(fixed.Point26_6{}, character)
	if !found {
		return nil, Entry{}
	}
	entry := Entry{Offset: bounds.Min, Advance: toPixels(advance)}
	if bounds.Empty() {
		return nil, entry
	}
	copied := image.NewAlpha(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(copied, copied.Bounds(), mask, maskPoint, draw.Src)
	return copied, entry
}

// prebuild draws the common characters of a text face at a size into the atlas.
func (painter *Painter) prebuild(name faceName, size int) {
	key := faceKey{name, size}
	if painter.prebuilt[key] {
		return
	}
	painter.prebuilt[key] = true
	for _, character := range prebuiltCharacters() {
		painter.glyph(name, size, character)
	}
}

// hasGlyph tells whether a face, or the system's fonts for it, can draw a character. The face's
// own glyphs, nearly every character drawn, are looked up once; only a character it lacks goes on
// to the system's fonts.
func (loaded *fonts) hasGlyph(name faceName, size int, character rune) bool {
	face := loaded.face(name, size)
	if face == nil {
		return false
	}
	if _, found := face.GlyphAdvance(character); found {
		return true
	}
	if name == iconFace {
		return false
	}
	fallback := loaded.faceFor(name, size, character)
	if fallback == nil || fallback == face {
		return false
	}
	_, found := fallback.GlyphAdvance(character)
	return found
}

func toPixels(value fixed.Int26_6) float32 { return float32(value) / 64 }

// pixels converts a length in layout pixels to whole screen pixels at a scale.
func pixels(length float32, scale float32) int {
	return int(math.Round(float64(length * scale)))
}
