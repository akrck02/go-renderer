package svg

import (
	"cmp"
	"math"
	"slices"
)

// subRows is how many rows of samples each row of pixels is cut into: where an edge crosses each
// one is found exactly, so 16 of them smooth a nearly flat edge as well as a steep one.
const subRows = 16

// curveFlatness is the length, in pixels, of the straight pieces a curve is cut into: short
// enough that the pieces cannot be told from the curve on a screen.
const curveFlatness = 1.0

// edge is a straight piece of an outline, in pixels.
type edge struct{ from, to point }

// top and bottom are the highest and the lowest of an edge's ends, y growing down.
func (piece edge) top() float64    { return min(piece.from.y, piece.to.y) }
func (piece edge) bottom() float64 { return max(piece.from.y, piece.to.y) }

// crossing is where an edge crosses a row of samples, and whether it goes down (+1) or up (-1).
type crossing struct {
	x         float64
	direction int
}

// coverage is how much of each pixel an outline covers, from 0 to 1, row after row: in each row
// of samples, the edges' crossings are sorted along the row, and the spans between them that are
// inside, as the fill rule says, are added, each pixel by the part of it they cover.
func coverage(steps []step, width, height int, evenOdd bool) []float32 {
	covered := make([]float32, width*height)
	edges := flatten(steps)
	// edges from the top down, so each row looks only at those it may cross: a stroked icon has
	// hundreds, and a row meets a few of them
	slices.SortFunc(edges, func(one, other edge) int { return cmp.Compare(one.top(), other.top()) })
	var crossings []crossing
	var across []edge
	next := 0
	for row := range height {
		line := covered[row*width : (row+1)*width]
		for next < len(edges) && edges[next].top() < float64(row+1) {
			across, next = append(across, edges[next]), next+1
		}
		across = slices.DeleteFunc(across, func(piece edge) bool { return piece.bottom() <= float64(row) })
		for sample := range subRows {
			crossings = crossingsAt(across, float64(row)+(float64(sample)+0.5)/subRows, crossings[:0])
			addInside(line, crossings, evenOdd)
		}
	}
	for index, amount := range covered {
		covered[index] = min(amount, 1)
	}
	return covered
}

// crossingsAt are where edges cross the line y, sorted along it. An edge holds its upper end and
// not its lower one, so two edges meeting at a corner cross once, not twice.
func crossingsAt(edges []edge, y float64, crossings []crossing) []crossing {
	for _, piece := range edges {
		if (piece.from.y <= y) == (piece.to.y <= y) {
			continue
		}
		along := (y - piece.from.y) / (piece.to.y - piece.from.y)
		direction := 1
		if piece.to.y < piece.from.y {
			direction = -1
		}
		crossings = append(crossings, crossing{piece.from.x + along*(piece.to.x-piece.from.x), direction})
	}
	slices.SortFunc(crossings, func(one, other crossing) int { return cmp.Compare(one.x, other.x) })
	return crossings
}

// addInside adds a row of samples' spans that are inside to a row of pixels. Non-zero is inside
// where the edges crossed so far do not cancel out; even-odd where an odd number were crossed.
func addInside(line []float32, crossings []crossing, evenOdd bool) {
	winding := 0
	for index, each := range crossings[:max(len(crossings)-1, 0)] {
		winding += each.direction
		inside := winding != 0
		if evenOdd {
			inside = winding%2 != 0
		}
		if inside {
			addSpan(line, each.x, crossings[index+1].x)
		}
	}
}

// addSpan adds a span of a row of samples to the pixels under it: whole pixels by a sample's
// share, the pixels at its ends by the part of them it covers.
func addSpan(line []float32, start, end float64) {
	const share = 1.0 / subRows
	start, end = max(start, 0), min(end, float64(len(line)))
	if !(start < end) { // written so, a span that is not a number is left out too
		return
	}
	first, last := int(start), int(end)
	if first == last {
		line[first] += float32((end - start) * share)
		return
	}
	line[first] += float32((float64(first+1) - start) * share)
	for pixel := first + 1; pixel < last; pixel++ {
		line[pixel] += share
	}
	if last < len(line) {
		line[last] += float32((end - float64(last)) * share)
	}
}

// flatten cuts an outline into straight edges, each run closed, as a fill closes it.
func flatten(steps []step) []edge {
	var edges []edge
	var reached, runStart point
	for _, piece := range steps {
		switch piece.kind {
		case moveStep:
			edges = closeRun(edges, reached, runStart)
			reached, runStart = piece.points[0], piece.points[0]
		case curveStep:
			edges = append(edges, curveEdges(reached, piece.points)...)
			reached = piece.points[2]
		default: // a line, or a close back to the run's start
			edges = append(edges, edge{reached, piece.end()})
			reached = piece.end()
		}
	}
	return closeRun(edges, reached, runStart)
}

// closeRun adds the edge back to a run's start, when the run did not end there.
func closeRun(edges []edge, reached, runStart point) []edge {
	if reached == runStart {
		return edges
	}
	return append(edges, edge{reached, runStart})
}

// curveEdges cuts a cubic curve from where the outline has reached into straight edges about curveFlatness long.
func curveEdges(from point, controls [3]point) []edge {
	reach := distance(from, controls[0]) + distance(controls[0], controls[1]) + distance(controls[1], controls[2])
	pieces := min(max(int(math.Ceil(reach/curveFlatness)), 1), 256)
	edges := make([]edge, 0, pieces)
	previous := from
	for piece := 1; piece <= pieces; piece++ {
		next := bezier(from, controls, float64(piece)/float64(pieces))
		edges = append(edges, edge{previous, next})
		previous = next
	}
	return edges
}

// bezier is the point of a cubic curve so far along it, from 0 at its start to 1 at its end.
func bezier(from point, controls [3]point, along float64) point {
	rest := 1 - along
	weights := [4]float64{rest * rest * rest, 3 * rest * rest * along, 3 * rest * along * along, along * along * along}
	return point{
		weights[0]*from.x + weights[1]*controls[0].x + weights[2]*controls[1].x + weights[3]*controls[2].x,
		weights[0]*from.y + weights[1]*controls[0].y + weights[2]*controls[1].y + weights[3]*controls[2].y,
	}
}

func distance(one, other point) float64 { return math.Hypot(other.x-one.x, other.y-one.y) }
