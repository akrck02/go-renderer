package draw2d

import (
	"image"
	"math"
)

// A path is a line of straight pieces and curves, in a program's own units, that a list draws as
// a stroke of some width: an icon's outline, a drawing in lines. A list only knows rectangles,
// text and pictures, so the stroke is drawn into a picture of its own (a mask: white, with the
// line's coverage as opacity, its edges smoothed) and the list shows the picture tinted. The
// picture is kept in the path and drawn again only when the stroke's size changes, so a path
// kept from frame to frame costs a picture each frame, like any other.

// Point is a place in a path's units.
type Point struct{ X, Y float32 }

// Path is a line made of straight pieces and curves. Build it with MoveTo, LineTo, CurveTo and
// Close, or with Circle and RoundedRectangle; keep it, and draw it with List.Stroke.
type Path struct {
	runs    [][]Point // each run of joined points; curves are already split into short lines
	picture *Image    // the stroke last drawn
	drawn   strokeSize
}

// strokeSize is what a stroke's picture depends on: the scale from the path's units to layout
// pixels, and the line's width.
type strokeSize struct{ scale, width float32 }

// curveSteps is how many straight pieces a curve is drawn with: enough for a curve on a screen to
// look smooth.
const curveSteps = 24

// strokeDensity is how many pixels of a stroke's picture a layout pixel holds: two, so that a
// stroke stays sharp on a screen of twice the density.
const strokeDensity = 2

// MoveTo starts a new run of the line at a point.
func (path *Path) MoveTo(x, y float32) {
	path.runs = append(path.runs, []Point{{x, y}})
	path.picture = nil
}

// LineTo draws a straight piece from the last point to another; with no point before, it starts
// there.
func (path *Path) LineTo(x, y float32) {
	if len(path.runs) == 0 {
		path.MoveTo(x, y)
		return
	}
	last := len(path.runs) - 1
	path.runs[last] = append(path.runs[last], Point{x, y})
	path.picture = nil
}

// CurveTo draws a cubic Bézier curve from the last point to (x, y), pulled towards two control
// points, as SVG's C does.
func (path *Path) CurveTo(firstX, firstY, secondX, secondY, x, y float32) {
	start := path.last()
	for step := 1; step <= curveSteps; step++ {
		at := float32(step) / curveSteps
		left := 1 - at
		path.LineTo(
			left*left*left*start.X+3*left*left*at*firstX+3*left*at*at*secondX+at*at*at*x,
			left*left*left*start.Y+3*left*left*at*firstY+3*left*at*at*secondY+at*at*at*y,
		)
	}
}

// Close draws a straight piece back to where the run began.
func (path *Path) Close() {
	if len(path.runs) == 0 {
		return
	}
	first := path.runs[len(path.runs)-1][0]
	path.LineTo(first.X, first.Y)
}

// last is the path's last point, or the origin when it has none.
func (path *Path) last() Point {
	if len(path.runs) == 0 {
		return Point{}
	}
	run := path.runs[len(path.runs)-1]
	return run[len(run)-1]
}

// circleHandle is how far a circle's quarter pulls its control points, as a share of its
// radius: the usual cubic Bézier approximation of a circle.
const circleHandle = 0.5523

// Circle adds a circle about a centre, as a run of its own.
func (path *Path) Circle(centreX, centreY, radius float32) {
	path.RoundedRectangle(centreX-radius, centreY-radius, 2*radius, 2*radius, radius)
}

// RoundedRectangle adds a rectangle with rounded corners, as a run of its own; a radius larger
// than half a side is made half of it.
func (path *Path) RoundedRectangle(x, y, width, height, radius float32) {
	radius = min(radius, width/2, height/2)
	handle := radius * (1 - circleHandle)
	right, bottom := x+width, y+height
	path.MoveTo(x+radius, y)
	path.LineTo(right-radius, y)
	path.CurveTo(right-handle, y, right, y+handle, right, y+radius)
	path.LineTo(right, bottom-radius)
	path.CurveTo(right, bottom-handle, right-handle, bottom, right-radius, bottom)
	path.LineTo(x+radius, bottom)
	path.CurveTo(x+handle, bottom, x, bottom-handle, x, bottom-radius)
	path.LineTo(x, y+radius)
	path.CurveTo(x, y+handle, x+handle, y, x+radius, y)
}

// Bounds is the smallest rectangle, in the path's units, that holds every point of it.
func (path *Path) Bounds() Rect {
	first := true
	var left, top, right, bottom float32
	for _, run := range path.runs {
		for _, point := range run {
			if first {
				left, top, right, bottom, first = point.X, point.Y, point.X, point.Y, false
				continue
			}
			left, top = min(left, point.X), min(top, point.Y)
			right, bottom = max(right, point.X), max(bottom, point.Y)
		}
	}
	return Rect{X: left, Y: top, Width: right - left, Height: bottom - top}
}

// Stroke draws a path's line, width layout pixels wide, in a colour: the path's point (0, 0) goes
// at (x, y) of the screen and each of its units is scale layout pixels.
func (list *List) Stroke(path *Path, x, y, scale, width float32, color Color) {
	if path == nil || len(path.runs) == 0 || width <= 0 || scale <= 0 || !color.IsVisible() {
		return
	}
	bounds := path.Bounds()
	area := Rect{X: x + bounds.X*scale - width, Y: y + bounds.Y*scale - width,
		Width: bounds.Width*scale + 2*width, Height: bounds.Height*scale + 2*width}
	size := strokeSize{scale: scale, width: width}
	if path.picture == nil || path.drawn != size {
		path.picture, path.drawn = path.strokePicture(bounds, size), size
	}
	list.Image(path.picture, area, 0, color)
}

// strokePicture draws the path's line into a mask: each piece as a capsule of the line's width,
// so pieces join and end round.
func (path *Path) strokePicture(bounds Rect, size strokeSize) *Image {
	pixelsPerUnit := size.scale * strokeDensity
	margin := size.width * strokeDensity
	width := int(math.Ceil(float64(bounds.Width*pixelsPerUnit + 2*margin)))
	height := int(math.Ceil(float64(bounds.Height*pixelsPerUnit + 2*margin)))
	picture := NewBlankImage(max(width, 1), max(height, 1))
	place := func(point Point) Point {
		return Point{(point.X-bounds.X)*pixelsPerUnit + margin, (point.Y-bounds.Y)*pixelsPerUnit + margin}
	}
	half := size.width * strokeDensity / 2
	for _, run := range path.runs {
		for index := range run {
			from := place(run[max(index-1, 0)])
			strokePiece(picture.Pixels, from, place(run[index]), half)
		}
	}
	return picture
}

// strokePiece covers the pixels of a mask within half a line's width of a straight piece, its
// edge smoothed over a pixel; a pixel already covered more keeps its coverage.
func strokePiece(mask *image.NRGBA, from, to Point, half float32) {
	reach := half + 1
	left, right := int(min(from.X, to.X)-reach), int(math.Ceil(float64(max(from.X, to.X)+reach)))
	top, bottom := int(min(from.Y, to.Y)-reach), int(math.Ceil(float64(max(from.Y, to.Y)+reach)))
	bounds := mask.Rect
	for row := max(top, bounds.Min.Y); row < min(bottom, bounds.Max.Y); row++ {
		for column := max(left, bounds.Min.X); column < min(right, bounds.Max.X); column++ {
			centre := Point{float32(column) + 0.5, float32(row) + 0.5}
			coverage := min(max(half+0.5-distanceToPiece(centre, from, to), 0), 1)
			offset := mask.PixOffset(column, row)
			if alpha := uint8(coverage * 255); alpha > mask.Pix[offset+3] {
				mask.Pix[offset], mask.Pix[offset+1], mask.Pix[offset+2], mask.Pix[offset+3] = 255, 255, 255, alpha
			}
		}
	}
}

// distanceToPiece is how far a point is from a straight piece between two others.
func distanceToPiece(point, from, to Point) float32 {
	alongX, alongY := to.X-from.X, to.Y-from.Y
	length := alongX*alongX + alongY*alongY
	share := float32(0)
	if length > 0 {
		share = min(max(((point.X-from.X)*alongX+(point.Y-from.Y)*alongY)/length, 0), 1)
	}
	nearestX, nearestY := from.X+share*alongX-point.X, from.Y+share*alongY-point.Y
	return float32(math.Sqrt(float64(nearestX*nearestX + nearestY*nearestY)))
}
