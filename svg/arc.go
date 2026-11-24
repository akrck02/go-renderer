package svg

import "math"

// ellipse is the shape an arc is a part of: its radii, and how far it is turned.
type ellipse struct {
	radiusX, radiusY, turnDegrees float64
}

// arcSteps are the cubic curves that draw an arc from one point to another, following SVG's
// implementation notes (https://www.w3.org/TR/SVG11/implnote.html#ArcImplementationNotes): find
// the ellipse's centre from the two ends and the flags, then draw it in pieces of at most a
// quarter turn, each close enough to a cubic curve. No radius is a line; the same ends, nothing.
func arcSteps(from, to point, shape ellipse, large, sweep bool) []step {
	if from == to {
		return nil
	}
	radiusX, radiusY := math.Abs(shape.radiusX), math.Abs(shape.radiusY)
	if radiusX == 0 || radiusY == 0 {
		return []step{{kind: lineStep, points: [3]point{to}}}
	}
	turn := shape.turnDegrees * math.Pi / 180
	cosine, sine := math.Cos(turn), math.Sin(turn)
	// the start, seen from the middle of the two ends, in the ellipse's own axes
	halfX, halfY := (from.x-to.x)/2, (from.y-to.y)/2
	startX, startY := cosine*halfX+sine*halfY, -sine*halfX+cosine*halfY
	// radii too small to reach from one end to the other are grown until they just do
	if reach := startX*startX/(radiusX*radiusX) + startY*startY/(radiusY*radiusY); reach > 1 {
		radiusX, radiusY = radiusX*math.Sqrt(reach), radiusY*math.Sqrt(reach)
	}
	centreX, centreY := arcCentre(startX, startY, radiusX, radiusY, large == sweep)
	first := angleBetween(1, 0, (startX-centreX)/radiusX, (startY-centreY)/radiusY)
	swept := angleBetween((startX-centreX)/radiusX, (startY-centreY)/radiusY, (-startX-centreX)/radiusX, (-startY-centreY)/radiusY)
	if !isFinite(first, swept, centreX, centreY) || swept == 0 {
		// radii so huge or so tiny that their squares run out of range: as near a line as drawn
		return []step{{kind: lineStep, points: [3]point{to}}}
	}
	if !sweep && swept > 0 {
		swept -= 2 * math.Pi
	} else if sweep && swept < 0 {
		swept += 2 * math.Pi
	}
	placed := placedEllipse{
		centre:  point{cosine*centreX - sine*centreY + (from.x+to.x)/2, sine*centreX + cosine*centreY + (from.y+to.y)/2},
		radiusX: radiusX, radiusY: radiusY, cosine: cosine, sine: sine,
	}
	steps := placed.curves(first, swept)
	steps[len(steps)-1].points[2] = to // the end exactly where the path says, not a rounding off
	return steps
}

// arcCentre is the centre of the ellipse, in its own axes, seen from the middle of the two ends:
// on one side of the line between them or the other, as the flags say.
func arcCentre(startX, startY, radiusX, radiusY float64, otherSide bool) (float64, float64) {
	squaredX, squaredY := radiusX*radiusX, radiusY*radiusY
	spread := squaredX*startY*startY + squaredY*startX*startX
	factor := math.Sqrt(math.Max(0, (squaredX*squaredY-spread)/spread))
	if otherSide {
		factor = -factor
	}
	return factor * radiusX * startY / radiusY, -factor * radiusY * startX / radiusX
}

// angleBetween is the turn, in radians, from one direction to another, with its sign.
func angleBetween(fromX, fromY, toX, toY float64) float64 {
	return math.Atan2(fromX*toY-fromY*toX, fromX*toX+fromY*toY)
}

// placedEllipse is an ellipse where it is drawn: its centre, radii and turn.
type placedEllipse struct {
	centre                         point
	radiusX, radiusY, cosine, sine float64
}

// at is the point of the ellipse at an angle of its own circle.
func (shape placedEllipse) at(unitX, unitY float64) point {
	x, y := unitX*shape.radiusX, unitY*shape.radiusY
	return point{shape.centre.x + shape.cosine*x - shape.sine*y, shape.centre.y + shape.sine*x + shape.cosine*y}
}

// curves draws the part of the ellipse from an angle, turning by swept, in pieces of at most a
// quarter turn: each a cubic curve whose controls lie along the tangents at its ends, at the
// length that keeps the curve on the ellipse (4/3 of the tangent of a quarter of the piece).
func (shape placedEllipse) curves(first, swept float64) []step {
	pieces := int(math.Ceil(math.Abs(swept) / (math.Pi / 2)))
	turn := swept / float64(pieces)
	reach := 4.0 / 3 * math.Tan(turn/4)
	steps := make([]step, 0, pieces)
	for piece := range pieces {
		start, end := first+turn*float64(piece), first+turn*float64(piece+1)
		startCosine, startSine := math.Cos(start), math.Sin(start)
		endCosine, endSine := math.Cos(end), math.Sin(end)
		steps = append(steps, step{kind: curveStep, points: [3]point{
			shape.at(startCosine-reach*startSine, startSine+reach*startCosine),
			shape.at(endCosine+reach*endSine, endSine-reach*endCosine),
			shape.at(endCosine, endSine),
		}})
	}
	return steps
}

// isFinite tells whether every value is a number, and not an infinite one.
func isFinite(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}
