package graphics

import "testing"

func TestBackdropWeightsAddUpToOne(test *testing.T) {
	for _, deviation := range []float32{0.5, 1, 2, 3} {
		weights := BackdropWeights(deviation)
		total := weights[0]
		for _, weight := range weights[1:] {
			total += 2 * weight
		}
		if total < 0.999 || total > 1.001 {
			test.Fatalf("deviation %v: the weights add up to %v", deviation, total)
		}
		for tap := 1; tap < len(weights); tap++ {
			if weights[tap] > weights[tap-1] {
				test.Fatalf("deviation %v: a farther tap weighs more than a nearer one", deviation)
			}
		}
	}
}

func TestWithoutDeviationTheBlurKeepsTheTexel(test *testing.T) {
	if weights := BackdropWeights(0); weights[0] != 1 || weights[1] != 0 {
		test.Fatalf("got %v", weights)
	}
}

func TestASnapshotIsHalvedUntilTheTapsCoverTheBlur(test *testing.T) {
	for _, example := range []struct {
		deviation float32
		halvings  int
	}{{1, 0}, {2, 0}, {4, 1}, {24, 4}, {48, 5}, {1000, 5}} {
		if got := BackdropHalvings(example.deviation); got != example.halvings {
			test.Errorf("a deviation of %v pixels: %d halvings, wanted %d", example.deviation, got, example.halvings)
		}
	}
}
