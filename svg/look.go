package svg

import (
	"strconv"
	"strings"

	"github.com/akrck02/go-renderer/draw2d"
)

// lineCap is how a stroke's open ends are drawn.
type lineCap int

const (
	buttCap   lineCap = iota // cut square at the end
	roundCap                 // a half circle past the end
	squareCap                // a half square past the end
)

// lineJoin is how a stroke turns a corner.
type lineJoin int

const (
	miterJoin lineJoin = iota // the two edges carried on to a point, cut off past the miter limit
	roundJoin                 // a round corner
	bevelJoin                 // the corner cut straight across
)

// paint is what fills or strokes a shape: a colour, or a gradient met by its id, drawn plain.
type paint struct {
	colour    draw2d.Color
	reference string // the id in a url(#id), until it is found
}

// visible tells whether the paint draws anything.
func (fill paint) visible() bool { return fill.colour[3] > 0 || fill.reference != "" }

// look is how a shape is painted, passed from a group to what it holds.
type look struct {
	fill, stroke                        paint
	strokeWidth, miterLimit             float64
	cap                                 lineCap
	join                                lineJoin
	evenOdd                             bool
	opacity, fillOpacity, strokeOpacity float64
	// dashes are the lengths of a dashed stroke's dashes and gaps, one after the other; none for
	// a whole stroke. dashOffset is how far into them the stroke starts.
	dashes     []float64
	dashOffset float64
	// invisible is visibility hidden: unlike display none, what it holds may show itself again
	invisible bool
}

// startingLook is how SVG paints a shape that says nothing: filled black, not stroked.
var startingLook = look{
	fill: paint{colour: draw2d.Color{0, 0, 0, 1}}, strokeWidth: 1, miterLimit: 4,
	opacity: 1, fillOpacity: 1, strokeOpacity: 1,
}

// with is the look changed by an element's attributes and its style; unknown values are ignored,
// so the look from its group stays.
func (shown look) with(all attributes) look {
	for name, value := range all {
		shown.set(name, strings.TrimSpace(value))
	}
	return shown
}

// set changes one property of the look.
func (shown *look) set(name, value string) {
	switch name {
	case "fill", "stroke":
		if colour, reference, found := readPaint(value); found {
			*shown.paintNamed(name) = paint{colour: colour, reference: reference}
		}
	case "stroke-width":
		shown.strokeWidth = length(value, shown.strokeWidth)
	case "stroke-miterlimit":
		shown.miterLimit = length(value, shown.miterLimit)
	case "stroke-linecap":
		if cap, found := lineCaps[value]; found {
			shown.cap = cap
		}
	case "stroke-linejoin":
		if join, found := lineJoins[value]; found {
			shown.join = join
		}
	case "stroke-dasharray":
		shown.dashes = dashes(value)
	case "stroke-dashoffset":
		shown.dashOffset = length(value, shown.dashOffset)
	case "fill-rule":
		if value == "evenodd" || value == "nonzero" {
			shown.evenOdd = value == "evenodd"
		}
	case "visibility":
		if value == "hidden" || value == "collapse" || value == "visible" {
			shown.invisible = value != "visible"
		}
	case "opacity":
		shown.opacity *= length(value, 1) // a group's opacity fades what it holds
	case "fill-opacity":
		shown.fillOpacity = length(value, shown.fillOpacity)
	case "stroke-opacity":
		shown.strokeOpacity = length(value, shown.strokeOpacity)
	}
}

func (shown *look) paintNamed(name string) *paint {
	if name == "stroke" {
		return &shown.stroke
	}
	return &shown.fill
}

// lineCaps and lineJoins are the caps and joins by their names in SVG; a word not here, as
// "inherit", leaves the one before.
var (
	lineCaps  = map[string]lineCap{"butt": buttCap, "round": roundCap, "square": squareCap}
	lineJoins = map[string]lineJoin{"miter": miterJoin, "round": roundJoin, "bevel": bevelJoin}
)

// colourNames are the few colours written by name that icons use; the rest of CSS's are left out.
var colourNames = map[string]draw2d.Color{
	"black": {0, 0, 0, 1}, "white": {1, 1, 1, 1}, "none": {}, "transparent": {},
	"currentColor": {0, 0, 0, 1}, // the colour of the text around, which an icon on its own has not: black
}

// readPaint reads a fill or a stroke: a colour by name, #rgb, #rrggbb, #rrggbbaa, or url(#id).
func readPaint(value string) (draw2d.Color, string, bool) {
	if colour, found := colourNames[value]; found {
		return colour, "", true
	}
	if strings.HasPrefix(value, "url(") {
		reference := strings.TrimSuffix(strings.TrimPrefix(value, "url("), ")")
		return draw2d.Color{}, strings.Trim(strings.TrimSpace(reference), `"'#`), true
	}
	if !strings.HasPrefix(value, "#") { // ParseHex would take "add" without one
		return draw2d.Color{}, "", false
	}
	colour, err := draw2d.ParseHex(value)
	return colour, "", err == nil
}

// length reads a number, with "px" after it or not; what is not one keeps the value before.
func length(value string, otherwise float64) float64 {
	number, err := strconv.ParseFloat(strings.TrimSuffix(value, "px"), 64)
	if err != nil || !isFinite(number) { // ParseFloat takes "NaN" and "Inf", which no length is
		return otherwise
	}
	return number
}

// dashes reads a stroke's dashes: none, or lengths that are not all zero. An odd count is
// repeated, as SVG says, so that dashes and gaps take turns.
func dashes(value string) []float64 {
	lengths, total := numbers(value), 0.0
	for _, each := range lengths {
		if each < 0 {
			return nil
		}
		total += each
	}
	if total == 0 {
		return nil
	}
	if len(lengths)%2 == 1 {
		lengths = append(lengths, lengths...)
	}
	return lengths
}
