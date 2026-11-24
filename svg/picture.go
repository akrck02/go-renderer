package svg

import (
	"image"
	"slices"

	"github.com/akrck02/go-renderer/draw2d"
)

// Picture draws the drawing into a picture of a size in pixels, in its own colours. Its view box
// is scaled to fit and centred, as SVG draws a picture by default. It is drawn anew on each call:
// keep the picture for as long as the size stays.
func (drawing *Drawing) Picture(width, height int) *draw2d.Image {
	canvas := newCanvas(width, height)
	toPixels := drawing.fitting(width, height)
	for _, each := range drawing.shapes {
		outline := toPixels.placed(slices.Clone(each.steps))
		if each.fill.visible() {
			canvas.lay(coverage(outline, width, height, each.evenOdd), each.fill.colour)
		}
		if each.stroke.visible() && each.strokeWidth > 0 {
			scale := toPixels.scale()
			drawn := pen{half: each.strokeWidth * scale / 2, cap: each.cap, join: each.join, miterLimit: each.miterLimit,
				dashes: scaledLengths(each.dashes, scale), dashOffset: each.dashOffset * scale}
			canvas.lay(coverage(strokeOutline(outline, drawn), width, height, false), each.stroke.colour)
		}
	}
	return draw2d.NewImage(canvas.straight())
}

// fitting is the transform from the view box to a picture's pixels: scaled to fit, centred.
func (drawing *Drawing) fitting(width, height int) transform {
	box := drawing.viewBox
	scale := min(float64(width)/box.width, float64(height)/box.height)
	left := (float64(width)-box.width*scale)/2 - box.x*scale
	top := (float64(height)-box.height*scale)/2 - box.y*scale
	return transform{scale, 0, 0, scale, left, top}
}

// canvas is a picture being painted: its colours premultiplied by their opacity, as laying one
// colour over another wants them.
type canvas struct {
	width, height int
	colours       []float32 // red, green, blue and opacity of each pixel
}

func newCanvas(width, height int) *canvas {
	return &canvas{width: width, height: height, colours: make([]float32, width*height*4)}
}

// lay paints a colour over the canvas where a coverage says, as much as it says.
func (painted *canvas) lay(covered []float32, colour draw2d.Color) {
	for pixel, amount := range covered {
		opacity := amount * colour[3]
		if opacity == 0 {
			continue
		}
		at := painted.colours[pixel*4 : pixel*4+4]
		for channel := range 3 {
			at[channel] = colour[channel]*opacity + at[channel]*(1-opacity)
		}
		at[3] = opacity + at[3]*(1-opacity)
	}
}

// straight is the canvas as a picture of straight colours, as draw2d keeps them.
func (painted *canvas) straight() *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, painted.width, painted.height))
	for pixel := range painted.width * painted.height {
		at := painted.colours[pixel*4 : pixel*4+4]
		if at[3] == 0 {
			continue
		}
		for channel := range 3 {
			picture.Pix[pixel*4+channel] = toByte(at[channel] / at[3])
		}
		picture.Pix[pixel*4+3] = toByte(at[3])
	}
	return picture
}

// toByte is a part of a colour, from 0 to 1, as a byte, rounded.
func toByte(value float32) uint8 { return uint8(min(max(value, 0), 1)*255 + 0.5) }
