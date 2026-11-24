package svg

import "testing"

// stroked is a document of 10 units a side with one stroked path, drawn at a pixel a unit.
func stroked(data, attributes string) string {
	return `<svg viewBox="0 0 10 10"><path fill="none" stroke="#000" stroke-width="2" ` + attributes + ` d="` + data + `"/></svg>`
}

func TestALinesCapsReachAsFarAsTheirKindSays(test *testing.T) {
	line := "M2 5 H8" // two units wide: rows 4 and 5, from x 2 to 8
	cases := []struct {
		cap                  string
		beforeEnd, roundPart float64 // pixel (1,5), past the start; pixel (1,4), the cap's corner
	}{
		{"butt", 0, 0},
		{"square", 1, 1},
		{"round", 0.785, 0.785}, // a quarter of a circle of radius 1: π/4 of the pixel
	}
	for _, each := range cases {
		source := stroked(line, `stroke-linecap="`+each.cap+`"`)
		if got := opacityAt(test, source, 10, 10, 1, 5); got < each.beforeEnd-0.05 || got > each.beforeEnd+0.05 {
			test.Errorf("a %s cap covers %.2f of the pixel past the line's end, want %.2f", each.cap, got, each.beforeEnd)
		}
		if got := opacityAt(test, source, 10, 10, 1, 4); got < each.roundPart-0.05 || got > each.roundPart+0.05 {
			test.Errorf("a %s cap covers %.2f of the pixel at its corner, want %.2f", each.cap, got, each.roundPart)
		}
		if got := opacityAt(test, source, 10, 10, 4, 5); got < 0.99 {
			test.Errorf("with a %s cap the line itself covers only %.2f", each.cap, got)
		}
	}
}

func TestACornerIsJoinedAsItsKindSays(test *testing.T) {
	corner := "M2 2 H8 V8" // its outer corner is the pixel from (8,0) to (9,1)
	for join, want := range map[string]float64{"miter": 1, "bevel": 0.5, "round": 0.785} {
		source := stroked(corner, `stroke-linejoin="`+join+`"`)
		if got := opacityAt(test, source, 10, 10, 8, 1); got < want-0.05 || got > want+0.05 {
			test.Errorf("a %s join covers %.2f of the outer corner's pixel, want %.2f", join, got, want)
		}
	}
	// past the miter limit a miter is cut to a bevel: a sharp turn, limit 1
	sharp := stroked("M1 1 L9 5 L1 9", `stroke-miterlimit="1"`)
	if got := opacityAt(test, sharp, 10, 10, 9, 5); got > 0.5 {
		test.Errorf("a miter past its limit still reaches its point: %.2f", got)
	}
}

// A closed outline's stroke has no ends: its last corner is joined like the others.
func TestAClosedOutlinesStrokeHasNoEnds(test *testing.T) {
	square := stroked("M2 2 H8 V8 H2 Z", ``)
	if got := opacityAt(test, square, 10, 10, 1, 1); got < 0.99 {
		test.Errorf("the closing corner covers %.2f, not joined as the others", got)
	}
	if got := opacityAt(test, square, 10, 10, 5, 5); got > 0.01 {
		test.Errorf("the inside of a stroked square is painted: %.2f", got)
	}
}

// A dashed stroke paints its dashes and leaves its gaps, from where its offset says.
func TestADashedStrokePaintsItsDashesAndLeavesItsGaps(test *testing.T) {
	dashed := stroked("M0 5 H10", `stroke-dasharray="2 2"`) // dashes from 0 to 2, 4 to 6, 8 to 10
	for x, want := range map[int]float64{1: 1, 2: 0, 3: 0, 5: 1, 9: 1} {
		if got := opacityAt(test, dashed, 10, 10, x, 5); got < want-0.01 || got > want+0.01 {
			test.Errorf("pixel %d of the dashed line covers %.2f, want %.0f", x, got, want)
		}
	}
	shifted := stroked("M0 5 H10", `stroke-dasharray="2 2" stroke-dashoffset="2"`) // a gap first
	if got := opacityAt(test, shifted, 10, 10, 1, 5); got > 0.01 {
		test.Errorf("with an offset of a dash, the line starts painted: %.2f", got)
	}
}

// Only what a path draws is stroked: a move alone paints nothing, even with caps that would give
// it a dot, and a closed run's start gets no dot of its own.
func TestAMoveAloneIsNotStroked(test *testing.T) {
	lone := stroked("M1 1 M5 5 H8", `stroke-linecap="round"`)
	if got := opacityAt(test, lone, 10, 10, 1, 1); got > 0.01 {
		test.Errorf("a move with nothing drawn from it paints a dot: %.2f", got)
	}
	triangle := `<svg viewBox="0 0 40 40"><path fill="none" stroke="#000" stroke-width="8" stroke-linecap="square" ` +
		`stroke-linejoin="bevel" d="M10 10 L30 10 L20 30 Z"/></svg>`
	if got := opacityAt(test, triangle, 40, 40, 7, 7); got > 0.01 {
		test.Errorf("a closed triangle's start gets a square cap: %.2f", got)
	}
}

// A closed run of two points goes there and back: joined at both ends, never capped.
func TestAClosedRunOfTwoPointsIsJoinedNotCapped(test *testing.T) {
	square := stroked("M2 5 H8 Z", `stroke-linecap="square"`)
	if got := opacityAt(test, square, 10, 10, 1, 5); got > 0.01 {
		test.Errorf("a closed line gets square caps: %.2f past its end", got)
	}
	round := stroked("M2 5 H8 Z", `stroke-linejoin="round"`)
	if got := opacityAt(test, round, 10, 10, 1, 5); got < 0.2 {
		test.Errorf("a closed line's round joins are missing: %.2f past its end", got)
	}
}
