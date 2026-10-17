// Package overlay lays out the scene's overlay (labels over the world, the header and the notices,
// see scene.Overlay) as textured rectangles in screen pixels, ready for a backend to draw with one
// texture: the Atlas, which holds the glyphs of the text, the Lucide icons and the corners of the
// rounded pills. It does no drawing itself and needs no graphics context, so it can be tested
// anywhere; the OpenGL scene renderer uploads its atlas and draws its quads.
package overlay

import (
	"image"
	"log"
	"math"
	"sort"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay/lucide"
	"github.com/akrck02/go-renderer/scene"
)

// Quad is a rectangle of the atlas drawn on screen with a colour: the atlas gives the coverage,
// the colour the paint. A quad of an image also reads its colours in the Pictures, which the
// colour then tints. Coordinates are pixels, with the origin at the top-left corner.
type Quad struct {
	Screen [4]float32 // left, top, right, bottom on screen
	Atlas  [4]float32 // left, top, right, bottom in the atlas
	Color  [4]float32 // straight RGBA; its alpha multiplies the coverage
	// Picture is where the quad reads its colours in the Pictures (left, top, right, bottom), when
	// Pictured.
	Picture  [4]float32
	Pictured bool
}

// Frame is what the layout needs to know about the frame being drawn.
type Frame struct {
	ViewProjection graphics.Mat4 // the camera's projection times its view
	Width, Height  int           // framebuffer size in pixels
	Scale          float32       // framebuffer pixels per layout pixel: 2 on a high-density screen; 0 = 1
	VerticalScale  float32       // the scene's vertical exaggeration; 0 = 1
}

// Painter lays out overlays frame after frame, keeping the atlas and the fonts between frames.
// It is used from one goroutine, the one that draws.
type Painter struct {
	atlas    *Atlas
	pictures *Pictures
	fonts    *fonts
	prebuilt map[faceKey]bool
	warned   map[string]bool
	// Warn reports a problem once (an unknown icon name); nil = the standard logger.
	Warn func(message string)

	quads    []Quad
	occupied []rectangle
	order    []int // the drawing's commands sorted by layer, reused every frame
	shown    []*draw2d.Image
	runes    []rune // a text's characters, reused: converting every text every frame is garbage
}

// NewPainter parses the embedded fonts and prepares an empty atlas.
func NewPainter() (*Painter, error) {
	loaded, err := loadFonts()
	if err != nil {
		return nil, err
	}
	return &Painter{atlas: NewAtlas(), pictures: NewPictures(), fonts: loaded, prebuilt: map[faceKey]bool{}, warned: map[string]bool{}}, nil
}

// Atlas returns the image the quads point into.
func (painter *Painter) Atlas() *Atlas { return painter.atlas }

// Pictures returns the image pictured quads read their colours from.
func (painter *Painter) Pictures() *Pictures { return painter.pictures }

// Paint lays out the whole overlay for a frame. anchors holds the world transform of the nodes
// labels are anchored to (scene.WorldTransforms); labels whose node is missing are not drawn.
// The returned slice is reused by the next call.
func (painter *Painter) Paint(overlay *scene.Overlay, anchors map[*scene.Node]graphics.Mat4, frame Frame) []Quad {
	painter.quads, painter.occupied = painter.quads[:0], painter.occupied[:0]
	if overlay == nil || frame.Width <= 0 || frame.Height <= 0 {
		return painter.quads
	}
	scale := frame.scale()
	bottom := painter.paintHeader(overlay.Header, scale)
	painter.paintNotices(overlay.VisibleNotices(), bottom, scale)
	painter.paintLabels(overlay.Labels, anchors, frame)
	painter.paintDrawing(overlay.Drawing, scale)
	return painter.quads
}

func (frame Frame) scale() float32 {
	if frame.Scale <= 0 {
		return 1
	}
	return frame.Scale
}

func (frame Frame) verticalScale() float32 {
	if frame.VerticalScale <= 0 {
		return 1
	}
	return frame.VerticalScale
}

// rectangle is an area of the screen in whole pixels.
type rectangle struct{ left, top, right, bottom int }

func (area rectangle) width() int  { return area.right - area.left }
func (area rectangle) height() int { return area.bottom - area.top }

func (area rectangle) overlaps(other rectangle) bool {
	return area.left < other.right && other.left < area.right && area.top < other.bottom && other.top < area.bottom
}

func (area rectangle) grown(margin int) rectangle {
	return rectangle{area.left - margin, area.top - margin, area.right + margin, area.bottom + margin}
}

func (area rectangle) moved(x, y int) rectangle {
	return rectangle{area.left + x, area.top + y, area.right + x, area.bottom + y}
}

// Projection is where a point of the world falls on screen.
type Projection struct {
	X, Y    float32 // pixels from the top-left corner
	Depth   float32 // -1 at the near plane, 1 at the far plane: smaller is nearer
	Visible bool    // in front of the camera and inside the screen
}

// Project places a world point on screen, applying the vertical exaggeration as the scene
// shaders do.
func Project(point graphics.Vec4, frame Frame) Projection {
	point[1] *= frame.verticalScale()
	point[3] = 1
	clip := frame.ViewProjection.Apply(point)
	if clip[3] <= 1e-6 {
		return Projection{}
	}
	x, y, depth := clip[0]/clip[3], clip[1]/clip[3], clip[2]/clip[3]
	projection := Projection{
		X:     (x + 1) / 2 * float32(frame.Width),
		Y:     (1 - y) / 2 * float32(frame.Height),
		Depth: depth,
	}
	projection.Visible = x >= -1 && x <= 1 && y >= -1 && y <= 1 && depth >= -1 && depth <= 1
	return projection
}

// anchorPoint returns the world point a label stands on: its node's origin or its point, lifted
// by its height.
func anchorPoint(label *scene.Label, anchors map[*scene.Node]graphics.Mat4) (graphics.Vec4, bool) {
	point := label.Point
	if label.Node != nil {
		transform, found := anchors[label.Node]
		if !found {
			return graphics.Vec4{}, false
		}
		point = transform.Apply(graphics.Vec4{0, 0, 0, 1})
	}
	point[1] += label.Height
	return point, true
}

// placedLabel is a label that is on screen this frame.
type placedLabel struct {
	label      *scene.Label
	projection Projection
}

// paintLabels draws the labels nearest first, skipping any that would cover one already drawn
// (or the header and the notices), so a crowded view stays readable.
func (painter *Painter) paintLabels(labels []*scene.Label, anchors map[*scene.Node]graphics.Mat4, frame Frame) {
	placed := make([]placedLabel, 0, len(labels))
	for _, label := range labels {
		if label == nil || label.Hidden || label.IsEmpty() {
			continue
		}
		point, found := anchorPoint(label, anchors)
		if !found {
			continue
		}
		projection := Project(point, frame)
		if projection.Visible {
			placed = append(placed, placedLabel{label, projection})
		}
	}
	sort.SliceStable(placed, func(first, second int) bool { return placed[first].projection.Depth < placed[second].projection.Depth })
	for _, candidate := range placed {
		painter.paintLabel(candidate, frame.scale())
	}
}

// labelGap is the space between a label's anchor and the bottom of the label, in layout pixels.
const labelGap = 6

// badgeGap is the space between a label and its badge, in layout pixels.
const badgeGap = 3

// paintLabel draws a label centred over its anchor: the label pill, and the badge under it.
func (painter *Painter) paintLabel(placed placedLabel, scale float32) {
	label := placed.label
	main := painter.measurePill(pillContent{icon: label.Icon, text: label.Text, tint: label.Tint}, labelStyle, scale)
	badge := painter.measurePill(pillContent{text: label.Badge, tint: label.Tint}, badgeStyle, scale)
	stackHeight := main.height + badge.height
	if main.height > 0 && badge.height > 0 {
		stackHeight += pixels(badgeGap, scale)
	}
	if stackHeight == 0 {
		return
	}
	centerX := int(math.Round(float64(placed.projection.X)))
	bottom := int(math.Round(float64(placed.projection.Y))) - pixels(labelGap, scale)
	top := bottom - stackHeight
	stackWidth := max(main.width, badge.width)
	stack := rectangle{centerX - stackWidth/2, top, centerX - stackWidth/2 + stackWidth, bottom}
	if !painter.claim(stack, pixels(2, scale)) {
		return
	}
	if main.height > 0 {
		painter.paintPill(main, centerX-main.width/2, top, 1)
	}
	if badge.height > 0 {
		painter.paintPill(badge, centerX-badge.width/2, bottom-badge.height, 1)
	}
}

// claim reserves an area of the screen, and refuses when another label or the header is there.
func (painter *Painter) claim(area rectangle, margin int) bool {
	padded := area.grown(margin)
	for _, taken := range painter.occupied {
		if padded.overlaps(taken) {
			return false
		}
	}
	painter.occupied = append(painter.occupied, area)
	return true
}

// screenMargin is the space between the header and the edges of the screen, in layout pixels.
const screenMargin = 16

// noticeGap separates the header from the notices and the notices from each other.
const noticeGap = 6

// paintHeader draws the header lines in a card at the top-left corner and returns the bottom of
// what it drew (the top margin when there is no header).
func (painter *Painter) paintHeader(lines []scene.HeaderLine, scale float32) int {
	margin := pixels(screenMargin, scale)
	if len(lines) == 0 {
		return margin - pixels(noticeGap, scale)
	}
	rows := make([]measuredContent, len(lines))
	widest := 0
	for index, line := range lines {
		lineStyle := headerStyle
		if index == 0 {
			lineStyle = headerFirstLineStyle
		}
		rows[index] = painter.measureContent(pillContent{icon: line.Icon, text: line.Text, tint: line.Tint}, lineStyle, scale)
		widest = max(widest, rows[index].width)
	}
	padding, rowHeight := pixels(headerPadding, scale), pixels(headerRowHeight, scale)
	card := rectangle{margin, margin, margin + widest + 2*padding, margin + len(rows)*rowHeight + 2*padding}
	painter.paintPanel(card, pixels(headerRadius, scale), 1, scale)
	for index, row := range rows {
		painter.paintContent(row, card.left+padding, card.top+padding+index*rowHeight, rowHeight, 1)
	}
	painter.occupied = append(painter.occupied, card)
	return card.bottom
}

// paintNotices draws the notices under the header, newest first, each fading as it ends.
func (painter *Painter) paintNotices(notices []scene.VisibleNotice, top int, scale float32) {
	left, gap := pixels(screenMargin, scale), pixels(noticeGap, scale)
	for _, notice := range notices {
		pill := painter.measurePill(pillContent{icon: notice.Icon, text: notice.Text, tint: notice.Tint}, noticeStyle, scale)
		if pill.height == 0 {
			continue
		}
		top += gap
		painter.paintPill(pill, left, top, notice.Opacity)
		painter.occupied = append(painter.occupied, rectangle{left, top, left + pill.width, top + pill.height})
		top += pill.height
	}
}

// pillContent is what goes in a pill or a header row: an icon, a text, and the icon's colour.
type pillContent struct {
	icon, text string
	tint       graphics.Vec4
}

// measuredContent is an icon and a text measured in screen pixels, ready to draw.
type measuredContent struct {
	style      pillStyle
	scale      float32
	icon       rune // 0 = none
	text       []rune
	textWidth  int
	iconSize   int
	gap        int
	width      int
	tint       graphics.Vec4
	textFace   faceName
	textSize   int
	iconColour [4]float32
}

// measuredPill is a pill measured in screen pixels.
type measuredPill struct {
	content       measuredContent
	width, height int
}

// measureContent measures an icon and a text side by side, cutting the text with an ellipsis
// when it is too long.
func (painter *Painter) measureContent(content pillContent, style pillStyle, scale float32) measuredContent {
	measured := measuredContent{style: style, scale: scale, tint: content.tint, textFace: style.face,
		textSize: pixels(style.textSize, scale), iconSize: pixels(style.iconSize, scale)}
	measured.icon = painter.iconCharacter(content.icon)
	painter.prebuild(style.face, measured.textSize)
	measured.text = painter.fitText(content.text, style.face, measured.textSize, pixels(style.maximumTextWidth, scale))
	measured.textWidth = painter.textWidth(measured.text, style.face, measured.textSize)
	measured.iconColour = iconColour(content.tint)
	if measured.icon == 0 {
		measured.iconSize = 0
	}
	if measured.icon != 0 && len(measured.text) > 0 {
		measured.gap = pixels(style.gap, scale)
	}
	measured.width = measured.iconSize + measured.gap + measured.textWidth
	return measured
}

// measurePill measures a pill; an empty one (no known icon, no text) has no size.
func (painter *Painter) measurePill(content pillContent, style pillStyle, scale float32) measuredPill {
	measured := painter.measureContent(content, style, scale)
	if measured.width == 0 {
		return measuredPill{}
	}
	height := evenPixels(style.height, scale)
	width := measured.width + 2*pixels(style.padding, scale)
	if len(measured.text) == 0 {
		width = height // an icon alone sits in a circle
	}
	return measuredPill{content: measured, width: max(width, height), height: height}
}

// evenPixels rounds a length to an even number of pixels, so a pill's radius is whole.
func evenPixels(length, scale float32) int {
	return 2 * int(math.Round(float64(length*scale/2)))
}

// iconCharacter returns the character of an icon in the icon font, or 0 (and a warning, once)
// for an unknown name.
func (painter *Painter) iconCharacter(name string) rune {
	if name == "" {
		return 0
	}
	character, found := lucide.Codepoint(name)
	if found {
		return character
	}
	if !painter.warned[name] {
		painter.warned[name] = true
		painter.warn("overlay: no icon named " + quote(name) + " (see https://lucide.dev/icons)")
	}
	return 0
}

func (painter *Painter) warn(message string) {
	if painter.Warn != nil {
		painter.Warn(message)
		return
	}
	log.Print(message)
}

func quote(text string) string { return "\"" + text + "\"" }

// textWidth adds up the advances of the characters, in pixels.
func (painter *Painter) textWidth(text []rune, face faceName, size int) int {
	width := float32(0)
	for _, character := range painter.drawable(text, face, size) {
		width += painter.glyph(face, size, character).Advance
	}
	return int(math.Ceil(float64(width)))
}

// drawable replaces the characters the font lacks with a question mark.
func (painter *Painter) drawable(text []rune, face faceName, size int) []rune {
	for index, character := range text {
		if !painter.fonts.hasGlyph(face, size, character) {
			text[index] = '?'
		}
	}
	return text
}

// fitText cuts a text so it fits a width, ending it with an ellipsis.
func (painter *Painter) fitText(text string, face faceName, size, maximumWidth int) []rune {
	characters := painter.drawable([]rune(text), face, size)
	if maximumWidth <= 0 || painter.textWidth(characters, face, size) <= maximumWidth {
		return characters
	}
	for length := len(characters) - 1; length > 0; length-- {
		shortened := append(append([]rune{}, characters[:length]...), '…')
		if painter.textWidth(shortened, face, size) <= maximumWidth {
			return shortened
		}
	}
	return []rune{'…'}
}

// paintPill draws a pill with its shadow, fill, border and content, at its top-left corner.
func (painter *Painter) paintPill(pill measuredPill, left, top int, opacity float32) {
	area := rectangle{left, top, left + pill.width, top + pill.height}
	style, scale := pill.content.style, pill.content.scale
	radius := pill.height / 2
	fill, border := style.colours(pill.content.tint)
	painter.paintRoundedBox(area, radius, fill, border, opacity, scale)
	contentLeft := left + (pill.width-pill.content.width)/2
	painter.paintContent(pill.content, contentLeft, top, pill.height, opacity)
}

// paintPanel draws a light card with rounded corners (the header).
func (painter *Painter) paintPanel(area rectangle, radius int, opacity, scale float32) {
	painter.paintRoundedBox(area, radius, panelFill, panelBorder, opacity, scale)
}

// paintRoundedBox draws a soft shadow, the fill and the border of a rounded rectangle.
func (painter *Painter) paintRoundedBox(area rectangle, radius int, fill, border [4]float32, opacity, scale float32) {
	blur, drop := max(pixels(shadowBlur, scale), 1), pixels(shadowDrop, scale)
	shadowArea := area.grown(blur).moved(0, drop)
	painter.ninePatch(painter.atlas.shadow(radius, blur), shadowArea, radius+blur, faded(shadowColour, opacity), true)
	painter.ninePatch(painter.atlas.disc(radius), area, radius, faded(fill, opacity), true)
	painter.ninePatch(painter.atlas.ring(radius, max(pixels(1, scale), 1)), area, radius, faded(border, opacity), false)
}

// paintContent draws an icon and a text side by side, centred vertically in a row.
func (painter *Painter) paintContent(content measuredContent, left, top, rowHeight int, opacity float32) {
	pen := left
	if content.icon != 0 {
		iconTop := top + (rowHeight-content.iconSize)/2
		// Lucide glyphs fill the em square above the baseline: the dot sits at its bottom-left
		painter.paintGlyph(iconFace, content.iconSize, content.icon, pen, iconTop+content.iconSize, faded(content.iconColour, opacity))
		pen += content.iconSize + content.gap
	}
	metrics := painter.fonts.metrics(content.textFace, content.textSize)
	baseline := top + int(math.Round(float64(float32(rowHeight)+metrics.capHeight)/2))
	position := float32(pen)
	for _, character := range content.text {
		entry := painter.paintGlyph(content.textFace, content.textSize, character, int(math.Round(float64(position))), baseline, faded(content.style.textColour, opacity))
		position += entry.Advance
	}
}

// paintGlyph draws one character with its dot at (x, y) and returns its entry.
func (painter *Painter) paintGlyph(face faceName, size int, character rune, x, y int, colour [4]float32) Entry {
	entry := painter.glyph(face, size, character)
	if entry.Bounds.Empty() {
		return entry
	}
	left, top := float32(x+entry.Offset.X), float32(y+entry.Offset.Y)
	bounds := entry.Bounds
	painter.quads = append(painter.quads, Quad{
		Screen: [4]float32{left, top, left + float32(bounds.Dx()), top + float32(bounds.Dy())},
		Atlas:  rectangleOf(bounds),
		Color:  colour,
	})
	return entry
}

func rectangleOf(bounds image.Rectangle) [4]float32 {
	return [4]float32{float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Max.X), float32(bounds.Max.Y)}
}

// ninePatch stretches a square picture over a rectangle: its four corners (corner pixels wide)
// keep their size, the middle row and column of the picture stretch along the edges, and its
// centre pixel fills the middle when fillCentre is set.
func (painter *Painter) ninePatch(entry Entry, area rectangle, corner int, colour [4]float32, fillCentre bool) {
	if entry.Bounds.Empty() || colour[3] <= 0 {
		return
	}
	corner = min(corner, area.width()/2, area.height()/2)
	pictureCorner := float32(entry.Bounds.Dx() / 2)
	atlasLeft, atlasTop := float32(entry.Bounds.Min.X), float32(entry.Bounds.Min.Y)
	middle := [2]float32{atlasLeft + pictureCorner - 0.5, atlasTop + pictureCorner - 0.5}
	columns := [3]span{
		{float32(area.left), float32(area.left + corner), atlasLeft, atlasLeft + float32(corner)},
		{float32(area.left + corner), float32(area.right - corner), middle[0], middle[0]},
		{float32(area.right - corner), float32(area.right), atlasLeft + 2*pictureCorner - float32(corner), atlasLeft + 2*pictureCorner},
	}
	rows := [3]span{
		{float32(area.top), float32(area.top + corner), atlasTop, atlasTop + float32(corner)},
		{float32(area.top + corner), float32(area.bottom - corner), middle[1], middle[1]},
		{float32(area.bottom - corner), float32(area.bottom), atlasTop + 2*pictureCorner - float32(corner), atlasTop + 2*pictureCorner},
	}
	for rowIndex, row := range rows {
		for columnIndex, column := range columns {
			if rowIndex == 1 && columnIndex == 1 && !fillCentre {
				continue
			}
			painter.addSpanQuad(column, row, colour)
		}
	}
}

// span is a stretch of the screen and the stretch of the atlas drawn on it, along one axis.
type span struct{ screenStart, screenEnd, atlasStart, atlasEnd float32 }

func (painter *Painter) addSpanQuad(column, row span, colour [4]float32) {
	if column.screenEnd <= column.screenStart || row.screenEnd <= row.screenStart {
		return
	}
	painter.quads = append(painter.quads, Quad{
		Screen: [4]float32{column.screenStart, row.screenStart, column.screenEnd, row.screenEnd},
		Atlas:  [4]float32{column.atlasStart, row.atlasStart, column.atlasEnd, row.atlasEnd},
		Color:  colour,
	})
}

func faded(colour [4]float32, opacity float32) [4]float32 {
	colour[3] *= opacity
	return colour
}
