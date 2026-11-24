package svg

import "math"

// roundSides is how many sides the polygon standing for a round cap or join has: enough to look
// round at the sizes icons are drawn.
const roundSides = 24

// roundJoinTolerance is, in pixels, how far a bevel may fall inside a round join's circle and be
// drawn instead: a curve cut into pieces turns a little at each, and a 24-sided polygon at every
// one of them is most of a stroked icon's cost.
const roundJoinTolerance = 0.01

// minimumDashPattern is, in pixels, the shortest dashes and gaps worth cutting a stroke into:
// shorter ones blur into the whole line, which is drawn instead of thousands of specks.
const minimumDashPattern = 0.5

// pen is how a stroke is drawn: half its width, and its caps and joins.
type pen struct {
	half       float64
	cap        lineCap
	join       lineJoin
	miterLimit float64
	dashes     []float64 // a dashed stroke's dashes and gaps, in pixels; none for a whole one
	dashOffset float64
}

// run is one stretch of an outline, cut into straight pieces: its points, and whether it is
// closed back to its start.
type run struct {
	points []point
	closed bool
}

// strokeOutline is the area a stroke paints, as an outline to fill by the non-zero rule: a
// quadrilateral along each straight piece, and a polygon for each cap and join. Each is turned the
// same way round, so where they overlap they add up instead of cancelling out.
func strokeOutline(steps []step, drawn pen) []step {
	var polygons [][]point
	runs := runsOf(steps)
	if drawn.dashes != nil && patternLength(drawn.dashes) >= minimumDashPattern {
		runs = dashed(runs, drawn.dashes, drawn.dashOffset)
	}
	for _, each := range runs {
		polygons = append(polygons, drawn.stretch(each)...)
	}
	var outline []step
	for _, polygon := range polygons {
		outline = append(outline, polygonSteps(turnedClockwise(polygon))...)
	}
	return outline
}

// stretch is the polygons of one run's stroke.
func (drawn pen) stretch(outline run) [][]point {
	points := outline.points
	if len(points) == 1 {
		return drawn.dot(points[0])
	}
	var polygons [][]point
	for index := 1; index < len(points); index++ {
		polygons = append(polygons, drawn.piece(points[index-1], points[index]))
	}
	for index := 1; index < len(points)-1; index++ {
		polygons = append(polygons, drawn.corner(points[index-1], points[index], points[index+1])...)
	}
	if outline.closed { // two points closed go there and back: joined at both ends, never capped
		last := len(points) - 1
		polygons = append(polygons, drawn.piece(points[last], points[0]))
		polygons = append(polygons, drawn.corner(points[last-1], points[last], points[0])...)
		return append(polygons, drawn.corner(points[last], points[0], points[1])...)
	}
	polygons = append(polygons, drawn.end(points[1], points[0])...)
	return append(polygons, drawn.end(points[len(points)-2], points[len(points)-1])...)
}

// piece is the quadrilateral a straight piece paints: as wide as the stroke, along it.
func (drawn pen) piece(from, to point) []point {
	side := scaled(normal(from, to), drawn.half)
	return []point{add(from, side), add(to, side), subtract(to, side), subtract(from, side)}
}

// end is the cap at a run's end, coming from the point before it.
func (drawn pen) end(before, at point) [][]point {
	switch drawn.cap {
	case roundCap:
		return [][]point{circle(at, drawn.half)}
	case squareCap:
		ahead := scaled(direction(before, at), drawn.half)
		return [][]point{drawn.piece(at, add(at, ahead))}
	}
	return nil
}

// dot is what a run of one point paints: nothing, unless its caps give it a shape.
func (drawn pen) dot(at point) [][]point {
	switch drawn.cap {
	case roundCap:
		return [][]point{circle(at, drawn.half)}
	case squareCap:
		return [][]point{{{at.x - drawn.half, at.y - drawn.half}, {at.x + drawn.half, at.y - drawn.half},
			{at.x + drawn.half, at.y + drawn.half}, {at.x - drawn.half, at.y + drawn.half}}}
	}
	return nil
}

// corner is the join where a run turns at a point, on the outer side of the turn: the inner side
// is already painted by the two pieces.
func (drawn pen) corner(before, at, after point) [][]point {
	into, out := direction(before, at), direction(at, after)
	turn := into.x*out.y - into.y*out.x
	if math.Abs(turn) < 1e-9 && into.x*out.x+into.y*out.y > 0 {
		return nil // straight on: no corner
	}
	if drawn.join == roundJoin && drawn.bevelMiss(into, out) >= roundJoinTolerance {
		return [][]point{circle(at, drawn.half)}
	}
	outer := -math.Copysign(1, turn) // the side of the turn's outside
	first := add(at, scaled(normal(before, at), drawn.half*outer))
	second := add(at, scaled(normal(at, after), drawn.half*outer))
	bevel := []point{at, first, second}
	if drawn.join != miterJoin || math.Abs(turn) < 1e-9 {
		return [][]point{bevel}
	}
	tip, found := meeting(first, into, second, out)
	if !found || distance(at, tip) > drawn.miterLimit*drawn.half {
		return [][]point{bevel} // past the limit a miter is cut off, as SVG says
	}
	return [][]point{{at, first, tip, second}}
}

// bevelMiss is how far, in pixels, a bevel across a turn falls inside the round join's circle:
// the half width less the bevel's distance from the corner, which is half × cos(turn / 2).
func (drawn pen) bevelMiss(into, out point) float64 {
	cosine := into.x*out.x + into.y*out.y
	return drawn.half * (1 - math.Sqrt(max(0, (1+cosine)/2)))
}
