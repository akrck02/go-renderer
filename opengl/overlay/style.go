package overlay

import "github.com/akrck02/go-renderer/graphics"

// The look of the overlay: light pills with a thin border and a soft shadow, and dark text, so
// labels read well over any scene, by day or by night. Lengths are layout pixels: on a
// high-density screen they are multiplied by the frame's scale.

// pillStyle is the size and the colours of one kind of pill.
type pillStyle struct {
	face             faceName
	height           float32
	textSize         float32
	iconSize         float32
	padding          float32 // between the edge and the content, on each side
	gap              float32 // between the icon and the text
	maximumTextWidth float32 // longer texts end with an ellipsis
	textColour       [4]float32
	tinted           bool // the fill and the border take the tint (badges)
}

var (
	ink            = [4]float32{0.094, 0.129, 0.180, 1} // #18212e
	pillFill       = [4]float32{1, 1, 1, 0.92}
	pillBorder     = [4]float32{0.094, 0.129, 0.180, 0.14}
	panelFill      = [4]float32{1, 1, 1, 0.94}
	panelBorder    = [4]float32{0.094, 0.129, 0.180, 0.12}
	badgeFill      = [4]float32{0.933, 0.945, 0.961, 0.95} // #eef1f5
	shadowColour   = [4]float32{0.04, 0.06, 0.10, 0.22}
	tintedBadgeMix = float32(0.2) // share of the tint in a tinted badge's fill
)

var labelStyle = pillStyle{face: textFace, height: 24, textSize: 13, iconSize: 15, padding: 9, gap: 5,
	maximumTextWidth: 240, textColour: ink}

var badgeStyle = pillStyle{face: textFace, height: 18, textSize: 11, padding: 7,
	maximumTextWidth: 120, textColour: ink, tinted: true}

var noticeStyle = pillStyle{face: textFace, height: 28, textSize: 13, iconSize: 16, padding: 11, gap: 7,
	maximumTextWidth: 420, textColour: ink}

var headerStyle = pillStyle{face: textFace, textSize: 14, iconSize: 16, gap: 8, maximumTextWidth: 420, textColour: ink}

var headerFirstLineStyle = pillStyle{face: strongFace, textSize: 16, iconSize: 18, gap: 8, maximumTextWidth: 420, textColour: ink}

const (
	headerPadding   = 12 // inside the header card
	headerRowHeight = 26
	headerRadius    = 10
	shadowBlur      = 3
	shadowDrop      = 1 // the shadow falls this far below the pill
)

// hasTint tells whether a colour was given (a zero alpha means "use the default").
func hasTint(tint graphics.Vec4) bool { return tint[3] > 0 }

// iconColour is the tint, or the text colour when there is none.
func iconColour(tint graphics.Vec4) [4]float32 {
	if !hasTint(tint) {
		return ink
	}
	return [4]float32{tint[0], tint[1], tint[2], 1}
}

// colours returns the fill and the border of a pill of this style with a tint.
func (style pillStyle) colours(tint graphics.Vec4) (fill, border [4]float32) {
	if !style.tinted {
		return pillFill, pillBorder
	}
	if !hasTint(tint) {
		return badgeFill, pillBorder
	}
	fill = mix([4]float32{1, 1, 1, 0.96}, [4]float32{tint[0], tint[1], tint[2], 0.96}, tintedBadgeMix)
	border = [4]float32{tint[0], tint[1], tint[2], 0.7}
	return fill, border
}

func mix(from, to [4]float32, amount float32) [4]float32 {
	var mixed [4]float32
	for channel := range mixed {
		mixed[channel] = from[channel] + (to[channel]-from[channel])*amount
	}
	return mixed
}
