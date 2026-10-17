package overlay

import (
	"image"
	"math"
	"sort"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/opengl/overlay/lucide"
)

// solidCornerRadius is the disc whose centre pixel paints square corners and plain rectangles:
// large enough that its centre is fully covered.
const solidCornerRadius = 4

// paintDrawing lays out a draw list (see draw2d) as quads over everything painted so far, layer by
// layer, cutting each command to its clip rectangle.
func (painter *Painter) paintDrawing(list *draw2d.List, scale float32) {
	if list.IsEmpty() {
		return
	}
	painter.preparePictures(list, scale)
	for _, index := range painter.drawingOrder(list) {
		command := &list.Commands[index]
		first := len(painter.quads)
		painter.paintCommand(command, scale)
		if command.HasClip {
			painter.clipQuads(first, screenArea(command.Clip, scale))
		}
	}
}

// drawingOrder lists the commands by layer, keeping their order within a layer.
func (painter *Painter) drawingOrder(list *draw2d.List) []int {
	order := painter.order[:0]
	sorted := true
	for index := range list.Commands {
		order = append(order, index)
		if index > 0 && list.Commands[index].Layer < list.Commands[index-1].Layer {
			sorted = false
		}
	}
	painter.order = order
	if sorted {
		return order // most lists are drawn layer by layer already: no sort, no closure
	}
	sort.SliceStable(order, func(first, second int) bool {
		return list.Commands[order[first]].Layer < list.Commands[order[second]].Layer
	})
	return order
}

func (painter *Painter) paintCommand(command *draw2d.Command, scale float32) {
	switch command.Kind {
	case draw2d.RectangleKind:
		painter.paintRectangle(command, scale)
	case draw2d.ShadowKind:
		painter.paintShadow(command, scale)
	case draw2d.TextKind:
		painter.paintText(command, scale)
	case draw2d.IconKind:
		painter.paintIcon(command, scale)
	case draw2d.ImageKind:
		painter.paintImage(command, scale)
	}
}

// preparePictures puts the pictures the list shows in the picture atlas before anything is laid
// out, so none moves while the frame's quads point at it: images, and the emoji in its texts.
func (painter *Painter) preparePictures(list *draw2d.List, scale float32) {
	shown := painter.shown[:0]
	for index := range list.Commands {
		command := &list.Commands[index]
		switch {
		case command.Kind == draw2d.ImageKind && command.Picture != nil:
			shown = append(shown, command.Picture)
		case command.Kind == draw2d.TextKind:
			shown = painter.emojiPictures(command, scale, shown)
		}
	}
	painter.shown = shown
	if len(shown) > 0 {
		painter.pictures.prepare(shown)
	}
}

// emojiPictures adds the pictures of a text's emoji to a list.
func (painter *Painter) emojiPictures(command *draw2d.Command, scale float32, shown []*draw2d.Image) []*draw2d.Image {
	if isASCII(command.Text) {
		return shown // no emoji in it: most texts
	}
	face, size := faceForWeight(command.Weight), max(pixels(command.Size, scale), 1)
	text := painter.runesOf(command.Text)
	for index, character := range text {
		if emoji, coloured := painter.fonts.colourFor(face, size, character, askedAsEmoji(text, index)); coloured {
			shown = append(shown, emoji.picture)
		}
	}
	return shown
}

// paintImage fills a rounded rectangle with a picture: the corners' coverage from the grey atlas
// as for any fill, the colours from the picture atlas, cropped to cover the rectangle.
func (painter *Painter) paintImage(command *draw2d.Command, scale float32) {
	area := screenArea(command.Area, scale)
	if area.width() <= 0 || area.height() <= 0 {
		return
	}
	bounds, found := painter.pictures.bounds(command.Picture)
	if !found {
		return
	}
	first := len(painter.quads)
	painter.paintFill(area, cornerPixels(command.Radius, area, scale), colourOf(command.Fill))
	painter.pictureQuads(first, area, coverCrop(bounds, area))
}

// pictureQuads gives the quads from first on the colours of a part of the picture atlas, source,
// stretched over an area.
func (painter *Painter) pictureQuads(first int, area rectangle, source [4]float32) {
	for index := first; index < len(painter.quads); index++ {
		quad := &painter.quads[index]
		for axis := range 2 {
			start, length := float32(area.left), float32(area.width())
			if axis == 1 {
				start, length = float32(area.top), float32(area.height())
			}
			for _, edge := range [2]int{axis, axis + 2} {
				along := (quad.Screen[edge] - start) / length
				quad.Picture[edge] = source[axis] + along*(source[axis+2]-source[axis])
			}
		}
		quad.Pictured = true
	}
}

// coverCrop is the part of a picture that covers an area of another shape: all of it along one
// axis, the middle of it along the other.
func coverCrop(bounds image.Rectangle, area rectangle) [4]float32 {
	left, top := float32(bounds.Min.X), float32(bounds.Min.Y)
	width, height := float32(bounds.Dx()), float32(bounds.Dy())
	areaShape := float32(area.width()) / float32(area.height())
	if width/height > areaShape {
		kept := height * areaShape
		left += (width - kept) / 2
		width = kept
	} else {
		kept := width / areaShape
		top += (height - kept) / 2
		height = kept
	}
	return [4]float32{left, top, left + width, top + height}
}

// screenArea turns a rectangle in layout pixels into whole screen pixels.
func screenArea(area draw2d.Rect, scale float32) rectangle {
	left, top := pixels(area.X, scale), pixels(area.Y, scale)
	return rectangle{left, top, pixels(area.Right(), scale), pixels(area.Bottom(), scale)}
}

// cornerPixels is a corner radius in screen pixels, no more than half the shorter side.
func cornerPixels(radius float32, area rectangle, scale float32) int {
	return max(0, min(pixels(radius, scale), area.width()/2, area.height()/2))
}

func (painter *Painter) paintRectangle(command *draw2d.Command, scale float32) {
	area := screenArea(command.Area, scale)
	if area.width() <= 0 || area.height() <= 0 {
		return
	}
	radius := cornerPixels(command.Radius, area, scale)
	painter.paintFill(area, radius, colourOf(command.Fill))
	if command.BorderWidth > 0 && command.Border.IsVisible() {
		painter.paintBorder(area, radius, max(pixels(command.BorderWidth, scale), 1), colourOf(command.Border))
	}
}

func (painter *Painter) paintFill(area rectangle, radius int, colour [4]float32) {
	if radius == 0 {
		painter.ninePatch(painter.atlas.disc(solidCornerRadius), area, 0, colour, true)
		return
	}
	painter.ninePatch(painter.atlas.disc(radius), area, radius, colour, true)
}

// paintBorder draws an outline inside the area: a ring's nine patches when the corners are round,
// four thin rectangles when they are square.
func (painter *Painter) paintBorder(area rectangle, radius, width int, colour [4]float32) {
	if radius > 0 {
		painter.ninePatch(painter.atlas.ring(radius, width), area, radius, colour, false)
		return
	}
	width = min(width, area.width()/2, area.height()/2)
	edges := [4]rectangle{ // an array: a slice here would be garbage every border
		{area.left, area.top, area.right, area.top + width},
		{area.left, area.bottom - width, area.right, area.bottom},
		{area.left, area.top + width, area.left + width, area.bottom - width},
		{area.right - width, area.top + width, area.right, area.bottom - width},
	}
	for _, edge := range edges {
		painter.paintFill(edge, 0, colour)
	}
}

func (painter *Painter) paintShadow(command *draw2d.Command, scale float32) {
	area := screenArea(command.Area, scale)
	blur := max(pixels(command.Blur, scale), 1)
	radius := max(pixels(command.Radius, scale), 1)
	painter.ninePatch(painter.atlas.shadow(radius, blur), area, radius+blur, colourOf(command.Fill), true)
}

func (painter *Painter) paintText(command *draw2d.Command, scale float32) {
	face := faceForWeight(command.Weight)
	size := max(pixels(command.Size, scale), 1)
	painter.prebuild(face, size)
	baseline := pixels(command.Baseline, scale)
	pen := command.Area.X * scale
	colour := colourOf(command.Fill)
	// an emoji keeps its own colours, faded as the text is
	emojiColour := colourOf(draw2d.Color{1, 1, 1, command.Fill[3]})
	text := painter.runesOf(command.Text)
	for index, character := range text {
		if invisible(character) {
			continue
		}
		if emoji, coloured := painter.fonts.colourFor(face, size, character, askedAsEmoji(text, index)); coloured {
			painter.paintEmoji(emoji, int(math.Round(float64(pen))), baseline, emojiColour)
			pen += emoji.advance
			continue
		}
		if !painter.fonts.hasGlyph(face, size, character) {
			character = '?'
		}
		entry := painter.paintGlyph(face, size, character, int(math.Round(float64(pen))), baseline, colour)
		pen += entry.Advance
	}
}

// paintEmoji draws an emoji's picture with its left edge and top placed from the pen on the
// baseline, as a glyph's would be.
func (painter *Painter) paintEmoji(emoji *emojiGlyph, pen, baseline int, colour [4]float32) {
	bounds, found := painter.pictures.bounds(emoji.picture)
	if !found {
		return
	}
	left := pen + int(math.Round(float64(emoji.left)))
	top := baseline - int(math.Round(float64(emoji.top)))
	area := rectangle{left, top, left + emoji.pixelWidth, top + emoji.pixelHeight}
	first := len(painter.quads)
	painter.paintFill(area, 0, colour)
	painter.pictureQuads(first, area, [4]float32{float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Max.X), float32(bounds.Max.Y)})
}

func (painter *Painter) paintIcon(command *draw2d.Command, scale float32) {
	character := painter.iconCharacter(command.Text)
	if character == 0 {
		return
	}
	size := max(pixels(command.Size, scale), 1)
	left, top := pixels(command.Area.X, scale), pixels(command.Area.Y, scale)
	// Lucide glyphs fill the em square above the baseline: the dot sits at its bottom-left
	painter.paintGlyph(iconFace, size, character, left, top+size, colourOf(command.Fill))
}

// clipQuads cuts the quads from first onwards to a rectangle of the screen, moving their atlas
// coordinates with their edges, and drops the ones left outside.
func (painter *Painter) clipQuads(first int, clip rectangle) {
	kept := first
	for index := first; index < len(painter.quads); index++ {
		quad, visible := clipQuad(painter.quads[index], clip)
		if visible {
			painter.quads[kept] = quad
			kept++
		}
	}
	painter.quads = painter.quads[:kept]
}

func clipQuad(quad Quad, clip rectangle) (Quad, bool) {
	limits := [4]float32{float32(clip.left), float32(clip.top), float32(clip.right), float32(clip.bottom)}
	for axis := 0; axis < 2; axis++ {
		low, high := axis, axis+2
		start, end := quad.Screen[low], quad.Screen[high]
		if end <= limits[low] || start >= limits[high] || end <= start {
			return quad, false
		}
		atlasPerPixel := (quad.Atlas[high] - quad.Atlas[low]) / (end - start)
		picturePerPixel := (quad.Picture[high] - quad.Picture[low]) / (end - start)
		if start < limits[low] {
			quad.Atlas[low] += (limits[low] - start) * atlasPerPixel
			quad.Picture[low] += (limits[low] - start) * picturePerPixel
			quad.Screen[low] = limits[low]
		}
		if end > limits[high] {
			quad.Atlas[high] -= (end - limits[high]) * atlasPerPixel
			quad.Picture[high] -= (end - limits[high]) * picturePerPixel
			quad.Screen[high] = limits[high]
		}
	}
	return quad, true
}

func colourOf(color draw2d.Color) [4]float32 { return [4]float32(color) }

func faceForWeight(weight draw2d.Weight) faceName {
	switch weight {
	case draw2d.Medium:
		return textFace
	case draw2d.Bold:
		return strongFace
	case draw2d.Mono:
		return monoFace
	default:
		return regularFace
	}
}

// Measurer measures text the way the painter draws it, without an atlas or a graphics context, so
// an interface library can lay out its controls before the frame is drawn. It implements
// draw2d.Measurer.
type Measurer struct {
	fonts *fonts
}

// NewMeasurer parses the embedded fonts.
func NewMeasurer() (*Measurer, error) {
	loaded, err := loadFonts()
	if err != nil {
		return nil, err
	}
	return &Measurer{fonts: loaded}, nil
}

// MeasureText measures a line at a size in layout pixels on a screen of a density (see
// draw2d.Measurer). A character the font lacks is measured in the system's font that has it
// (FontFolders), or as a question mark, as it is drawn.
func (measurer *Measurer) MeasureText(text string, size float32, weight draw2d.Weight, scale float32) draw2d.TextMetrics {
	if scale <= 0 {
		scale = 1
	}
	face, pixelSize := faceForWeight(weight), max(pixels(size, scale), 1)
	fontFace := measurer.fonts.face(face, pixelSize)
	if fontFace == nil {
		return draw2d.TextMetrics{}
	}
	width := float32(0)
	characters := []rune(text)
	for index, character := range characters {
		if invisible(character) {
			continue
		}
		if emoji, coloured := measurer.fonts.colourFor(face, pixelSize, character, askedAsEmoji(characters, index)); coloured {
			width += emoji.advance
			continue
		}
		advance, found := measurer.fonts.faceFor(face, pixelSize, character).GlyphAdvance(character)
		if !found {
			advance, _ = fontFace.GlyphAdvance('?')
		}
		width += toPixels(advance)
	}
	metrics := measurer.fonts.metrics(face, pixelSize)
	return draw2d.TextMetrics{Width: width / scale, Ascent: metrics.ascent / scale, Descent: metrics.descent / scale}
}

// HasIcon tells whether Lucide has an icon of this name.
func (measurer *Measurer) HasIcon(name string) bool {
	_, found := lucide.Codepoint(name)
	return found
}

// runesOf is a text's characters in a slice the painter keeps: valid until the next call.
func (painter *Painter) runesOf(text string) []rune {
	runes := painter.runes[:0]
	for _, character := range text {
		runes = append(runes, character)
	}
	painter.runes = runes
	return runes
}

// isASCII tells whether a text is plain ASCII, without looking at it character by character.
func isASCII(text string) bool {
	for index := 0; index < len(text); index++ {
		if text[index] >= 0x80 {
			return false
		}
	}
	return true
}
