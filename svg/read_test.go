package svg

import (
	"errors"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
)

// document wraps shapes in an SVG of 24 units a side, as icon sets draw them.
func document(inside string) string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24">` + inside + `</svg>`
}

func mustRead(test *testing.T, source string) *Drawing {
	test.Helper()
	drawing, err := Read([]byte(source))
	if err != nil {
		test.Fatal(err)
	}
	return drawing
}

func TestEachShapeReadsIntoAPath(test *testing.T) {
	cases := map[string]string{
		"path":         `<path d="M2 2 L22 22"/>`,
		"rect":         `<rect x="2" y="2" width="20" height="20"/>`,
		"rounded rect": `<rect x="2" y="2" width="20" height="20" rx="4"/>`,
		"circle":       `<circle cx="12" cy="12" r="10"/>`,
		"ellipse":      `<ellipse cx="12" cy="12" rx="10" ry="5"/>`,
		"line":         `<line x1="2" y1="2" x2="22" y2="22"/>`,
		"polyline":     `<polyline points="2,2 22,2 22,22"/>`,
		"polygon":      `<polygon points="2 2 22 2 22 22"/>`,
	}
	for name, inside := range cases {
		drawing := mustRead(test, document(inside))
		if len(drawing.shapes) != 1 || len(drawing.shapes[0].steps) < 2 {
			test.Errorf("a %s reads into %d shapes, want one with its steps", name, len(drawing.shapes))
			continue
		}
		bounds := outlineBounds(drawing.shapes[0].steps)
		if bounds.x < 2-1e-6 || bounds.y < 2-1e-6 || bounds.x+bounds.width > 22+1e-6 || bounds.y+bounds.height > 22+1e-6 {
			test.Errorf("a %s spans %+v, outside where it was drawn", name, bounds)
		}
	}
	if drawing := mustRead(test, document("")); drawing.viewBox != (rectangle{0, 0, 24, 24}) {
		test.Errorf("the view box is %+v, want 0 0 24 24", drawing.viewBox)
	}
}

func TestAGroupPassesItsColoursAndPlaceDown(test *testing.T) {
	drawing := mustRead(test, document(
		`<g fill="none" stroke="#f00" stroke-width="2" stroke-linecap="round" transform="translate(10 0)">`+
			`<path d="M0 0 L1 0"/><path style="stroke: #00ff00; stroke-linejoin: bevel" d="M0 0 L1 0"/></g>`))
	first, second := drawing.shapes[0], drawing.shapes[1]
	if first.fill.visible() || first.stroke.colour != draw2d.MustHex("#ff0000") || first.strokeWidth != 2 || first.cap != roundCap {
		test.Errorf("the first path does not take its group's paint: %+v", first.look)
	}
	if second.stroke.colour != draw2d.MustHex("#00ff00") || second.join != bevelJoin {
		test.Errorf("a path's own style does not win over its group's: %+v", second.look)
	}
	if start := first.steps[0].end(); !near(start, point{10, 0}) {
		test.Errorf("the group's move does not place the path: it starts at %v", start)
	}
}

func TestEveryKindOfTransformPlacesAShape(test *testing.T) {
	cases := map[string]point{
		"translate(5)":              {6, 0},
		"translate(5 5)":            {6, 5},
		"scale(2)":                  {2, 0},
		"scale(2 3)":                {2, 0},
		"rotate(90)":                {0, 1},
		"rotate(90 10 10)":          {20, 1},
		"skewX(45)":                 {1, 0},
		"skewY(45)":                 {1, 1},
		"matrix(1 0 0 1 3 4)":       {4, 4},
		"translate(1,1) scale(2,2)": {3, 1},
	}
	for transform, want := range cases {
		drawing := mustRead(test, document(`<path transform="`+transform+`" d="M0 0 L1 0"/>`))
		if got := drawing.shapes[0].steps[1].end(); !near(got, want) {
			test.Errorf("%s puts (1,0) at %v, want %v", transform, got, want)
		}
	}
}

// Fills and strokes default as SVG says: a black fill, no stroke, a stroke one unit wide.
func TestColoursDefaultAndReadAsSVGWritesThem(test *testing.T) {
	plain := mustRead(test, document(`<path d="M0 0 L1 1"/>`)).shapes[0]
	if plain.fill.colour != draw2d.MustHex("#000000") || plain.stroke.visible() || plain.strokeWidth != 1 {
		test.Errorf("a path with no paint is %+v, want a black fill and no stroke", plain.look)
	}
	short := mustRead(test, document(`<path fill="#f80" fill-opacity="0.5" d="M0 0 L1 1"/>`)).shapes[0]
	if want := draw2d.MustHex("#ff8800").WithAlpha(0.5); short.fill.colour != want {
		test.Errorf("#f80 at half opacity reads as %v, want %v", short.fill.colour, want)
	}
}

// A gradient is drawn as the plain colour of its stops' average, wherever it is defined.
func TestAGradientIsTheAverageOfItsStops(test *testing.T) {
	drawing := mustRead(test, document(
		`<path fill="url(#warm)" d="M0 0 L1 1"/>`+
			`<defs><linearGradient id="warm"><stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/></linearGradient></defs>`))
	if want := draw2d.MustHex("#800080"); !closeColour(drawing.shapes[0].fill.colour, want) {
		test.Errorf("the gradient is drawn in %v, want about %v", drawing.shapes[0].fill.colour, want)
	}
}

func TestADocumentWithWhatIsNotDrawnStillGivesItsShapes(test *testing.T) {
	drawing := mustRead(test, document(
		`<defs><clipPath id="clip"><rect width="1" height="1"/></clipPath></defs>`+
			`<mask id="hole"><rect width="24" height="24" fill="#fff"/></mask>`+
			`<text x="1" y="1">words</text>`+
			`<g clip-path="url(#clip)" mask="url(#hole)"><path d="M0 0 L1 1"/></g>`))
	if len(drawing.shapes) != 1 {
		test.Errorf("the document gives %d shapes, want only its one path", len(drawing.shapes))
	}
}

func TestADocumentThatIsNotSVGHasItsCode(test *testing.T) {
	for _, source := range []string{"not xml at all <", `<html><body/></html>`, ""} {
		if _, err := Read([]byte(source)); !errors.Is(err, NotAnSVG) {
			test.Errorf("%q: %v, not %s", source, err, NotAnSVG.Name())
		}
	}
}

// A path that cannot be read is told by its code, and the rest of the drawing is kept, as a
// browser draws what it could read.
func TestABadPathIsToldAndTheRestIsKept(test *testing.T) {
	drawing, err := Read([]byte(document(`<path d="M0 0 L1 1 X"/><path d="M0 0 L2 2"/>`)))
	if !errors.Is(err, BadPath) {
		test.Errorf("the bad path gives %v, not %s", err, BadPath.Name())
	}
	if drawing == nil || len(drawing.shapes) != 2 {
		test.Errorf("the drawing keeps %v, want both paths, the first up to its error", drawing)
	}
}

// closeColour tells whether two colours are the same to within a step of 8 bits.
func closeColour(one, other draw2d.Color) bool {
	for channel := range one {
		if difference := one[channel] - other[channel]; difference > 1.0/255 || difference < -1.0/255 {
			return false
		}
	}
	return true
}

// A use of a shape defined elsewhere paints that shape, placed by the use: icon sets draw the
// same piece several times this way.
func TestAUseOfAShapePaintsItWhereTheUseSays(test *testing.T) {
	drawing := mustRead(test, document(
		`<defs><path id="piece" fill="#00ff00" d="M0 0 L1 0"/></defs>`+
			`<use href="#piece" transform="translate(5 0)"/><use xlink:href="#piece" x="0" y="3"/><use href="#nothing"/>`))
	if len(drawing.shapes) != 2 {
		test.Fatalf("two uses give %d shapes, want 2", len(drawing.shapes))
	}
	if start := drawing.shapes[0].steps[0].end(); !near(start, point{5, 0}) {
		test.Errorf("the first use starts at %v, want (5,0)", start)
	}
	if start := drawing.shapes[1].steps[0].end(); !near(start, point{0, 3}) {
		test.Errorf("the second use starts at %v, want (0,3)", start)
	}
	if drawing.shapes[0].fill.colour != draw2d.MustHex("#00ff00") {
		test.Errorf("a use does not keep its shape's own colour: %v", drawing.shapes[0].fill.colour)
	}
}

// tenUnits is a document of 10 units a side, drawn at a pixel a unit in a picture of 10.
func tenUnits(inside string) string {
	return `<svg viewBox="0 0 10 10">` + inside + `</svg>`
}

// What a group sets is kept by what it holds unless they say otherwise, as CSS inherits it:
// a shape shown inside a hidden group shows, and "inherit" keeps the group's value.
func TestAShapeSaysOtherwiseThanItsGroupOrInheritsIt(test *testing.T) {
	shown := tenUnits(`<g visibility="hidden"><rect width="10" height="10" visibility="visible"/></g>`)
	if got := opacityAt(test, shown, 10, 10, 5, 5); got < 0.99 {
		test.Errorf("a visible shape in a hidden group is not painted: %.2f", got)
	}
	inherited := `<svg viewBox="0 0 10 10"><g stroke-linecap="round"><path fill="none" stroke="#000" stroke-width="2" ` +
		`stroke-linecap="inherit" d="M2 5 H8"/></g></svg>`
	if got := opacityAt(test, inherited, 10, 10, 1, 4); got < 0.5 {
		test.Errorf("inherit drops the group's round caps: %.2f at the cap's corner", got)
	}
}

func TestNumbersOfATransformMayRunTogether(test *testing.T) {
	moved := tenUnits(`<rect width="2" height="2" transform="translate(4-0)"/>`)
	if got := opacityAt(test, moved, 10, 10, 5, 1); got < 0.99 {
		test.Errorf("translate(4-0) does not move the square 4 across: %.2f", got)
	}
}

func TestAColourIsReadOnlyWithItsHash(test *testing.T) {
	for _, value := range []string{"add", "beef00"} {
		if _, _, found := readPaint(value); found {
			test.Errorf("%q, with no #, reads as a colour", value)
		}
	}
}

// Opacity fades what it is on, and a group's fades what the group holds.
func TestOpacityFadesAShapeAndWhatAGroupHolds(test *testing.T) {
	alone := tenUnits(`<rect width="10" height="10" opacity="0.5"/>`)
	inGroup := tenUnits(`<g opacity="0.5"><rect width="10" height="10" fill-opacity="0.5"/></g>`)
	if got := opacityAt(test, alone, 10, 10, 5, 5); got < 0.48 || got > 0.52 {
		test.Errorf("a shape at half opacity covers %.2f", got)
	}
	if got := opacityAt(test, inGroup, 10, 10, 5, 5); got < 0.23 || got > 0.27 {
		test.Errorf("a half-faded fill in a half-faded group covers %.2f, want a quarter", got)
	}
}
