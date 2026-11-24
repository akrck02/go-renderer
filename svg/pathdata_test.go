package svg

import (
	"errors"
	"math"
	"testing"
)

// near tells whether two points are the same, give or take a rounding.
func near(one, other point) bool {
	return math.Abs(one.x-other.x) < 1e-6 && math.Abs(one.y-other.y) < 1e-6
}

// ends are where each step of an outline ends.
func ends(steps []step) []point {
	var points []point
	for _, step := range steps {
		points = append(points, step.end())
	}
	return points
}

func TestEveryPathCommandReadsIntoMovesLinesAndCurves(test *testing.T) {
	cases := []struct {
		data  string
		kinds []stepKind
		ends  []point
	}{
		{"M1 2 L3 4", []stepKind{moveStep, lineStep}, []point{{1, 2}, {3, 4}}},
		{"m1 2 l3 4", []stepKind{moveStep, lineStep}, []point{{1, 2}, {4, 6}}},
		{"M0 0 3 4 5 6", []stepKind{moveStep, lineStep, lineStep}, []point{{0, 0}, {3, 4}, {5, 6}}},
		{"M1 1 H5 V7 h-1 v-2", []stepKind{moveStep, lineStep, lineStep, lineStep, lineStep}, []point{{1, 1}, {5, 1}, {5, 7}, {4, 7}, {4, 5}}},
		{"M0 0 C1 1 2 1 3 0 S5 -1 6 0", []stepKind{moveStep, curveStep, curveStep}, []point{{0, 0}, {3, 0}, {6, 0}}},
		{"M0 0 Q1 1 2 0 T4 0", []stepKind{moveStep, curveStep, curveStep}, []point{{0, 0}, {2, 0}, {4, 0}}},
		{"M0 0 L1 0 L1 1 Z M5 5 l1 1 z", []stepKind{moveStep, lineStep, lineStep, closeStep, moveStep, lineStep, closeStep}, []point{{0, 0}, {1, 0}, {1, 1}, {0, 0}, {5, 5}, {6, 6}, {5, 5}}},
		{"M.5-.5l1e1,2.5.5-1", []stepKind{moveStep, lineStep, lineStep}, []point{{0.5, -0.5}, {10.5, 2}, {11, 1}}}, // numbers run together, as minifiers write them
	}
	for _, each := range cases {
		steps, err := readPathData(each.data)
		if err != nil {
			test.Errorf("%q: %v", each.data, err)
			continue
		}
		if len(steps) != len(each.kinds) {
			test.Errorf("%q reads into %d steps, want %d", each.data, len(steps), len(each.kinds))
			continue
		}
		for index, step := range steps {
			if step.kind != each.kinds[index] || !near(step.end(), each.ends[index]) {
				test.Errorf("%q step %d is %v to %v, want %v to %v", each.data, index, step.kind, step.end(), each.kinds[index], each.ends[index])
			}
		}
	}
}

// A smooth curve mirrors the control point before it: S and T continue a curve without a corner.
func TestASmoothCurveMirrorsTheControlBeforeIt(test *testing.T) {
	steps, _ := readPathData("M0 0 C0 1 2 1 2 0 S4 -1 4 0")
	if want := (point{2, -1}); !near(steps[2].points[0], want) {
		test.Errorf("the smooth curve starts towards %v, want the mirror %v", steps[2].points[0], want)
	}
}

// A quadratic curve becomes the cubic one that draws the same line.
func TestAQuadraticCurveBecomesTheSameCubicOne(test *testing.T) {
	steps, _ := readPathData("M0 0 Q3 3 6 0")
	curve := steps[1]
	if !near(curve.points[0], point{2, 2}) || !near(curve.points[1], point{4, 2}) {
		test.Errorf("the cubic controls are %v and %v, want (2,2) and (4,2)", curve.points[0], curve.points[1])
	}
}

func TestAnArcBecomesCurvesThroughItsEnds(test *testing.T) {
	// half a circle of radius 5 around (5,0), from (0,0) to (10,0), flags written together
	steps, err := readPathData("M0 0 A5 5 0 0110 0")
	if err != nil {
		test.Fatal(err)
	}
	if len(steps) < 3 || !near(steps[len(steps)-1].end(), point{10, 0}) {
		test.Fatalf("the arc reads into %v, want curves ending at (10,0)", ends(steps))
	}
	for _, step := range steps[1:] {
		end := step.end()
		if step.kind != curveStep || math.Abs(math.Hypot(end.x-5, end.y)-5) > 1e-6 {
			test.Errorf("a piece of the arc ends at %v, off its circle", end)
		}
	}
	// sweep 1 turns the positive way, clockwise as the screen shows it, since y grows down: from
	// the left end it goes up, over the top
	if middle := steps[1].end(); middle.y > 0 {
		test.Errorf("the arc goes through %v, on the side its sweep does not say", middle)
	}
}

// An arc too small to reach its end is scaled up, as SVG says, and one with no radius is a line.
func TestAnArcOutOfReachIsScaledAndOneWithoutRadiusIsALine(test *testing.T) {
	steps, err := readPathData("M0 0 A1 1 0 0 1 10 0")
	if err != nil || !near(steps[len(steps)-1].end(), point{10, 0}) {
		test.Errorf("the small arc reads into %v, %v", ends(steps), err)
	}
	steps, _ = readPathData("M0 0 A0 5 0 0 1 10 0")
	if len(steps) != 2 || steps[1].kind != lineStep {
		test.Errorf("the arc without radius reads into %v, want a line", steps)
	}
}

func TestAPathThatCannotBeReadHasItsCode(test *testing.T) {
	for _, data := range []string{"L1 1", "M1", "M1 1 X2 2", "M0 0 A1 1 0 2 1 3 3"} {
		if _, err := readPathData(data); !errors.Is(err, BadPath) {
			test.Errorf("%q: %v, not %s", data, err, BadPath.Name())
		}
	}
}

// An empty path draws nothing, and is no error: icon sets have them.
func TestAnEmptyPathIsNothing(test *testing.T) {
	if steps, err := readPathData("  "); err != nil || len(steps) != 0 {
		test.Errorf("an empty path reads into %v, %v", steps, err)
	}
}
