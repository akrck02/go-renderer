package svg

import "math"

// runsOf cuts an outline into its runs of straight pieces: curves cut as a fill cuts them, and a
// point repeated in a row kept once.
func runsOf(steps []step) []run {
	var runs []run
	var current run
	drew := false // a move alone draws nothing; a line or a close, even of no length, draws a dot
	keep := func() {
		if drew && len(current.points) > 0 {
			runs = append(runs, current)
		}
		current, drew = run{}, false
	}
	var reached point
	for _, piece := range steps {
		switch piece.kind {
		case moveStep:
			keep()
			current.points = []point{piece.points[0]}
		case curveStep:
			drew = true
			for _, each := range curveEdges(reached, piece.points) {
				current.points = appendPoint(current.points, each.to)
			}
		case lineStep:
			drew = true
			current.points = appendPoint(current.points, piece.points[0])
		case closeStep:
			drew, current.closed = true, true
			if count := len(current.points); count > 1 && current.points[count-1] == current.points[0] {
				current.points = current.points[:count-1] // its last point is its first: kept once
			}
			start := current.points[0]
			keep()
			current.points = []point{start} // what follows a close starts where the run did
		}
		reached = piece.end()
	}
	keep()
	return runs
}

// appendPoint adds a point to a run, unless it is the one before again.
func appendPoint(points []point, next point) []point {
	if len(points) > 0 && points[len(points)-1] == next {
		return points
	}
	return append(points, next)
}

// circle is a polygon close to a circle.
func circle(centre point, radius float64) []point {
	points := make([]point, roundSides)
	for side := range roundSides {
		angle := 2 * math.Pi * float64(side) / roundSides
		points[side] = point{centre.x + radius*math.Cos(angle), centre.y + radius*math.Sin(angle)}
	}
	return points
}

// turnedClockwise is a polygon going round clockwise as the screen shows it (its signed area
// positive, y growing down); one going the other way is turned round.
func turnedClockwise(polygon []point) []point {
	area := 0.0
	for index, at := range polygon {
		next := polygon[(index+1)%len(polygon)]
		area += at.x*next.y - next.x*at.y
	}
	if area >= 0 {
		return polygon
	}
	turned := make([]point, len(polygon))
	for index, at := range polygon {
		turned[len(polygon)-1-index] = at
	}
	return turned
}

// polygonSteps is a polygon as a closed run of an outline.
func polygonSteps(polygon []point) []step {
	steps := []step{{kind: moveStep, points: [3]point{polygon[0]}}}
	for _, at := range polygon[1:] {
		steps = append(steps, step{kind: lineStep, points: [3]point{at}})
	}
	return append(steps, step{kind: closeStep, points: [3]point{polygon[0]}})
}

// meeting is where two lines cross, each through a point along a direction; none when parallel.
func meeting(first, firstDirection, second, secondDirection point) (point, bool) {
	across := firstDirection.x*secondDirection.y - firstDirection.y*secondDirection.x
	if math.Abs(across) < 1e-12 {
		return point{}, false
	}
	gap := subtract(second, first)
	along := (gap.x*secondDirection.y - gap.y*secondDirection.x) / across
	return add(first, scaled(firstDirection, along)), true
}

// direction is the unit vector from one point towards another.
func direction(from, to point) point {
	length := distance(from, to)
	if length == 0 {
		return point{}
	}
	return point{(to.x - from.x) / length, (to.y - from.y) / length}
}

// normal is the unit vector square to the way from one point to another.
func normal(from, to point) point {
	along := direction(from, to)
	return point{-along.y, along.x}
}

func add(one, other point) point            { return point{one.x + other.x, one.y + other.y} }
func subtract(one, other point) point       { return point{one.x - other.x, one.y - other.y} }
func scaled(vector point, by float64) point { return point{vector.x * by, vector.y * by} }
