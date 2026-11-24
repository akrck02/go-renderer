package svg

import "math"

// maximumDashes is how many dashes a run is cut into at the most; past them the rest of the run
// stays as it falls, all dash or all gap. No picture shows that many, but a line far
// longer than the picture would ask for them, and take seconds.
const maximumDashes = 4096

// dashed cuts runs into their dashes: open runs for the dashes, nothing for the gaps. Each run
// starts the dashes again, at the offset, as SVG starts them again on each part of a path.
func dashed(runs []run, dashes []float64, offset float64) []run {
	var pieces []run
	for _, each := range runs {
		points := each.points
		if each.closed {
			points = append(points[:len(points):len(points)], points[0])
		}
		pieces = append(pieces, dashesAlong(points, newDashWalk(dashes, offset))...)
	}
	return pieces
}

// dashWalk is where a walk along a line is in its dashes: which one, how much of it is left, and
// whether it is a dash or a gap.
type dashWalk struct {
	dashes []float64
	index  int
	left   float64
	on     bool
}

// newDashWalk starts at the offset into the dashes; a negative offset counts back from the end.
func newDashWalk(dashes []float64, offset float64) dashWalk {
	total := patternLength(dashes)
	offset = math.Mod(offset, total)
	if offset < 0 {
		offset += total
	}
	walk := dashWalk{dashes: dashes, left: dashes[0], on: true}
	for offset > 0 {
		step := min(offset, walk.left)
		walk.left, offset = walk.left-step, offset-step
		if walk.left == 0 {
			walk.next()
		}
	}
	return walk
}

// next moves on to the next dash or gap.
func (walk *dashWalk) next() {
	walk.index = (walk.index + 1) % len(walk.dashes)
	walk.left, walk.on = walk.dashes[walk.index], !walk.on
}

// dashesAlong walks a line of points, keeping the dashes as runs of their own.
func dashesAlong(points []point, walk dashWalk) []run {
	var pieces []run
	var current []point
	if walk.on {
		current = []point{points[0]}
	}
	for index := 1; index < len(points); index++ {
		from, to := points[index-1], points[index]
		length, done := distance(from, to), 0.0
		for length-done > walk.left && len(pieces) < maximumDashes {
			done += walk.left
			at := add(from, scaled(direction(from, to), done))
			if walk.on {
				pieces = append(pieces, run{points: append(current, at)})
				current = nil
			} else {
				current = []point{at}
			}
			walk.next()
		}
		walk.left -= length - done
		if walk.on {
			current = append(current, to)
		}
	}
	if walk.on && len(current) > 1 {
		pieces = append(pieces, run{points: current})
	}
	return pieces
}

// patternLength is how long the dashes and gaps are, one after the other.
func patternLength(dashes []float64) float64 {
	total := 0.0
	for _, each := range dashes {
		total += each
	}
	return total
}
