package overlay

import (
	"image"
	"image/draw"
	"math"
)

// Atlas is one grey image holding everything the overlay draws: the glyphs of the text, the
// icons and the rounded corners of the pills. Every picture is a coverage mask (0 = empty,
// 255 = covered) that the backend tints when it draws it, so a single texture and a single draw
// call paint the whole overlay.
//
// Pictures are added when first needed and never move; when the image is full it doubles in size
// (keeping what it had in the same place). Version changes whenever pixels change, so the backend
// knows to upload the image again.
type Atlas struct {
	Image   *image.Alpha
	Version int

	cursorX, cursorY, rowHeight int
	entries                     map[entryKey]Entry
}

// Entry is a picture in the atlas: where its pixels are, and where to draw them relative to a
// pen position (for glyphs, the dot on the baseline).
type Entry struct {
	Bounds  image.Rectangle // pixels in the atlas
	Offset  image.Point     // top-left corner of the picture relative to the pen
	Advance float32         // how far the pen moves after a glyph, in pixels
}

// entryKey names a picture: a glyph of a face at a size, or a shape of a size.
type entryKey struct {
	kind      entryKind
	face      faceName
	size      int // pixels
	character rune
	extra     int // ring width, shadow blur
}

type entryKind int

const (
	glyphEntry entryKind = iota
	discEntry
	ringEntry
	shadowEntry
)

// entryPadding keeps one empty pixel around every picture, so linear filtering never mixes
// neighbours.
const entryPadding = 1

const initialAtlasSize = 512
const maximumAtlasSize = 8192

// NewAtlas returns an empty atlas.
func NewAtlas() *Atlas {
	return &Atlas{Image: image.NewAlpha(image.Rect(0, 0, initialAtlasSize, initialAtlasSize)), entries: map[entryKey]Entry{}, Version: 1}
}

// Size returns the side of the square image, in pixels.
func (atlas *Atlas) Size() int { return atlas.Image.Bounds().Dx() }

// lookup returns a picture, drawing it with paint the first time. paint fills a mask of the
// size it asks for; the entry it returns says how to place it.
func (atlas *Atlas) lookup(key entryKey, paint func() (*image.Alpha, Entry)) Entry {
	if entry, found := atlas.entries[key]; found {
		return entry
	}
	mask, entry := paint()
	if mask != nil && !mask.Bounds().Empty() {
		entry.Bounds = atlas.place(mask)
	}
	atlas.entries[key] = entry
	return entry
}

// place copies a mask into free space (a shelf packer: rows of pictures, left to right) and
// returns where it went.
func (atlas *Atlas) place(mask *image.Alpha) image.Rectangle {
	width, height := mask.Bounds().Dx(), mask.Bounds().Dy()
	for !atlas.fits(width, height) {
		if !atlas.grow() {
			return image.Rectangle{}
		}
	}
	if atlas.cursorX+width+2*entryPadding > atlas.Size() {
		atlas.cursorX, atlas.cursorY, atlas.rowHeight = 0, atlas.cursorY+atlas.rowHeight+entryPadding, 0
	}
	origin := image.Pt(atlas.cursorX+entryPadding, atlas.cursorY+entryPadding)
	bounds := image.Rectangle{Min: origin, Max: origin.Add(image.Pt(width, height))}
	draw.Draw(atlas.Image, bounds, mask, mask.Bounds().Min, draw.Src)
	atlas.cursorX += width + entryPadding
	atlas.rowHeight = max(atlas.rowHeight, height+entryPadding)
	atlas.Version++
	return bounds
}

// fits tells whether a picture fits on the current row or on a new one.
func (atlas *Atlas) fits(width, height int) bool {
	size := atlas.Size()
	if width+2*entryPadding > size {
		return false
	}
	if atlas.cursorX+width+2*entryPadding <= size {
		return atlas.cursorY+height+2*entryPadding <= size
	}
	return atlas.cursorY+atlas.rowHeight+height+2*entryPadding <= size
}

// grow doubles the image, keeping every picture where it was.
func (atlas *Atlas) grow() bool {
	size := atlas.Size() * 2
	if size > maximumAtlasSize {
		return false
	}
	larger := image.NewAlpha(image.Rect(0, 0, size, size))
	draw.Draw(larger, atlas.Image.Bounds(), atlas.Image, image.Point{}, draw.Src)
	atlas.Image = larger
	atlas.Version++
	return true
}

// disc returns a filled circle of a radius, anti-aliased; its quarters are the corners of pills.
func (atlas *Atlas) disc(radius int) Entry {
	key := entryKey{kind: discEntry, size: radius}
	if entry, found := atlas.entries[key]; found {
		return entry // without making the closure below, which would allocate (see Painter.glyph)
	}
	return atlas.drawDisc(key)
}

func (atlas *Atlas) drawDisc(key entryKey) Entry {
	radius := key.size
	return atlas.lookup(key, func() (*image.Alpha, Entry) {
		return circleMask(radius, func(distance float64) float64 { return clamp01(float64(radius) - distance + 0.5) }), Entry{}
	})
}

// ring returns the outline of a circle, width pixels thick inside the radius: the pill borders.
func (atlas *Atlas) ring(radius, width int) Entry {
	key := entryKey{kind: ringEntry, size: radius, extra: width}
	if entry, found := atlas.entries[key]; found {
		return entry
	}
	return atlas.drawRing(key)
}

func (atlas *Atlas) drawRing(key entryKey) Entry {
	radius, width := key.size, key.extra
	return atlas.lookup(key, func() (*image.Alpha, Entry) {
		return circleMask(radius, func(distance float64) float64 {
			outer := clamp01(float64(radius) - distance + 0.5)
			inner := clamp01(float64(radius-width) - distance + 0.5)
			return outer - inner
		}), Entry{}
	})
}

// shadow returns a soft disc: solid up to radius minus blur, fading out to radius plus blur. Its
// side is 2·(radius+blur), so its corners are radius+blur wide.
func (atlas *Atlas) shadow(radius, blur int) Entry {
	key := entryKey{kind: shadowEntry, size: radius, extra: blur}
	if entry, found := atlas.entries[key]; found {
		return entry
	}
	return atlas.drawShadow(key)
}

func (atlas *Atlas) drawShadow(key entryKey) Entry {
	radius, blur := key.size, key.extra
	return atlas.lookup(key, func() (*image.Alpha, Entry) {
		inner, outer := float64(radius-blur), float64(radius+blur)
		return circleMask(radius+blur, func(distance float64) float64 { return 1 - smoothStep(inner, outer, distance) }), Entry{}
	})
}

// circleMask paints a square of side 2·radius with coverage as a function of the distance of
// each pixel centre to the centre of the square.
func circleMask(radius int, coverage func(distance float64) float64) *image.Alpha {
	side := 2 * radius
	mask := image.NewAlpha(image.Rect(0, 0, side, side))
	for row := 0; row < side; row++ {
		for column := 0; column < side; column++ {
			distance := math.Hypot(float64(column)+0.5-float64(radius), float64(row)+0.5-float64(radius))
			mask.Pix[row*mask.Stride+column] = uint8(math.Round(255 * clamp01(coverage(distance))))
		}
	}
	return mask
}

func clamp01(value float64) float64 { return math.Max(0, math.Min(1, value)) }

func smoothStep(edgeLow, edgeHigh, value float64) float64 {
	if edgeHigh <= edgeLow {
		if value < edgeLow {
			return 0
		}
		return 1
	}
	amount := clamp01((value - edgeLow) / (edgeHigh - edgeLow))
	return amount * amount * (3 - 2*amount)
}
