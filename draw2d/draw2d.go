// Package draw2d is a list of flat shapes to draw over the 3D view, in screen space: rounded
// rectangles with a border, soft shadows, lines of text and icons, each optionally clipped to a
// rectangle and placed on a layer. A program (or an interface library) fills a List every frame;
// a backend paints it on top of everything else.
//
// Coordinates are layout pixels with the origin at the top-left corner of the window: the same
// units as the cursor in input.State. The backend multiplies them by the screen's density (2 on a
// high-density screen), so a 13-pixel text has the same size on every screen and stays sharp.
//
// The package does no drawing and needs no graphics context: it is plain data, easy to build and
// to check in tests.
package draw2d

import (
	"strconv"
	"strings"

	"github.com/akrck02/go-renderer/failure"
)

// Rectangle is an area of the screen: its top-left corner and its size, in layout pixels.
type Rectangle struct {
	X, Y, Width, Height float32
}

// Right is the x of the rectangle's right edge.
func (rect Rectangle) Right() float32 { return rect.X + rect.Width }

// Bottom is the y of the rectangle's bottom edge.
func (rect Rectangle) Bottom() float32 { return rect.Y + rect.Height }

// IsEmpty tells whether the rectangle covers no area.
func (rect Rectangle) IsEmpty() bool { return rect.Width <= 0 || rect.Height <= 0 }

// Contains tells whether a point is inside the rectangle (left and top edges included).
func (rect Rectangle) Contains(x, y float32) bool {
	return x >= rect.X && y >= rect.Y && x < rect.Right() && y < rect.Bottom()
}

// Intersect returns the area two rectangles share; it is empty when they do not overlap.
func (rect Rectangle) Intersect(other Rectangle) Rectangle {
	left, top := max(rect.X, other.X), max(rect.Y, other.Y)
	right, bottom := min(rect.Right(), other.Right()), min(rect.Bottom(), other.Bottom())
	if right <= left || bottom <= top {
		return Rectangle{X: left, Y: top}
	}
	return Rectangle{X: left, Y: top, Width: right - left, Height: bottom - top}
}

// Inset shrinks the rectangle by an amount on every side.
func (rect Rectangle) Inset(amount float32) Rectangle {
	return Rectangle{X: rect.X + amount, Y: rect.Y + amount, Width: max(rect.Width-2*amount, 0), Height: max(rect.Height-2*amount, 0)}
}

// Color is a straight (not premultiplied) RGBA colour, each channel from 0 to 1, as seen on
// screen: it is not gamma-corrected like the lit scene.
type Color [4]float32

// Transparent paints nothing.
var Transparent = Color{}

// Hex reads a colour written as "#rrggbb" or "#rrggbbaa". It panics on a malformed colour, so it
// is meant for constants such as a theme's; use ParseHex for colours read from files.
func MustHex(text string) Color {
	color, err := ParseHex(text)
	if err != nil {
		panic(err)
	}
	return color
}

// ParseHex reads a colour written as on a web page, "#rrggbb", "#rrggbbaa" or the short "#rgb";
// InvalidColour otherwise.
func ParseHex(text string) (Color, error) {
	digits := strings.TrimPrefix(strings.TrimSpace(text), "#")
	if len(digits) == 3 {
		digits = string([]byte{digits[0], digits[0], digits[1], digits[1], digits[2], digits[2]})
	}
	if len(digits) != 6 && len(digits) != 8 {
		return Color{}, failure.Wrap(InvalidColour, strconv.Quote(text), nil)
	}
	if len(digits) == 6 {
		digits += "ff"
	}
	value, err := strconv.ParseUint(digits, 16, 32)
	if err != nil {
		return Color{}, failure.Wrap(InvalidColour, strconv.Quote(text), err)
	}
	channel := func(shift uint) float32 { return float32((value>>shift)&0xff) / 255 }
	return Color{channel(24), channel(16), channel(8), channel(0)}, nil
}

// WithAlpha returns the colour with its alpha multiplied by an opacity.
func (color Color) WithAlpha(opacity float32) Color {
	color[3] *= opacity
	return color
}

// Mix blends towards another colour: 0 keeps this one, 1 gives the other.
func (color Color) Mix(other Color, amount float32) Color {
	for channel := range color {
		color[channel] += (other[channel] - color[channel]) * amount
	}
	return color
}

// IsVisible tells whether the colour paints anything.
func (color Color) IsVisible() bool { return color[3] > 0 }

// Weight is how heavy a text is drawn.
type Weight int

const (
	Regular Weight = iota // body text
	Medium                // labels that should stand out a little
	Bold                  // titles
	Mono                  // codes and keys, in a monospaced face, so they stand apart from words
)

// TextMetrics are the measures of a line of text, in layout pixels.
type TextMetrics struct {
	Width   float32 // how far the line reaches from its start
	Ascent  float32 // from the baseline up to the top of the tallest letters
	Descent float32 // from the baseline down to the bottom of the lowest letters
}

// Height is the line's height: ascent plus descent.
func (metrics TextMetrics) Height() float32 { return metrics.Ascent + metrics.Descent }

// Measurer measures text the way the backend draws it, so that what is laid out is what is
// painted. scale is the screen's density (framebuffer pixels per layout pixel): text is drawn at
// size·scale pixels, whose widths do not always scale exactly.
type Measurer interface {
	MeasureText(text string, size float32, weight Weight, scale float32) TextMetrics
	// HasIcon tells whether an icon name is known to the backend.
	HasIcon(name string) bool
}
