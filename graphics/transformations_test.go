package graphics

import (
	"math"
	"testing"
)

// tolerance is how far two float32 results may drift apart and still count as equal.
const tolerance = 1e-4

func approximatelyEqual(first, second float32) bool {
	return math.Abs(float64(first-second)) < tolerance
}

func vectorsApproximatelyEqual(first, second Vector) bool {
	for axis := range first {
		if !approximatelyEqual(first[axis], second[axis]) {
			return false
		}
	}
	return true
}

func TestAMatrixTimesItsInverseIsTheIdentity(test *testing.T) {
	matrix := Translate(Vector{3, -2, 5, 0}).Multiply(RotateY(0.7)).Multiply(Scale(Vector{2, 3, 0.5, 1}))
	inverse, invertible := matrix.Inverse()
	if !invertible {
		test.Fatal("matrix reported singular")
	}
	product := matrix.Multiply(inverse)
	identity := Identity()
	for element := range product {
		if !approximatelyEqual(product[element], identity[element]) {
			test.Fatalf("matrix * inverse != identity at %d: %v", element, product)
		}
	}
}

func TestTheZeroMatrixHasNoInverse(test *testing.T) {
	if _, invertible := (Matrix{}).Inverse(); invertible {
		test.Fatal("zero matrix should be singular")
	}
}

func TestTranslationRotationAndScaleApplyInGLTFOrder(test *testing.T) {
	// 90 degrees around y: +x goes to -z
	halfSquareRoot := float32(math.Sqrt(0.5))
	matrix := FromTranslationRotationScale(Vector{1, 2, 3, 0}, Vector{0, halfSquareRoot, 0, halfSquareRoot}, Vector{2, 2, 2, 0})
	point := matrix.Apply(Vector{1, 0, 0, 1})
	if !vectorsApproximatelyEqual(point, Vector{1, 2, 1, 1}) {
		test.Fatalf("unexpected transform: %v", point)
	}
}

func TestTheOrthographicProjectionMapsItsBoxToTheClipCube(test *testing.T) {
	projection := OrthographicOpenGL(-2, 6, -1, 3, 1, 11)
	nearCorner := projection.Apply(Vector{-2, -1, -1, 1})
	farCorner := projection.Apply(Vector{6, 3, -11, 1})
	if !vectorsApproximatelyEqual(nearCorner, Vector{-1, -1, -1, 1}) {
		test.Fatalf("near corner maps to %v", nearCorner)
	}
	if !vectorsApproximatelyEqual(farCorner, Vector{1, 1, 1, 1}) {
		test.Fatalf("far corner maps to %v", farCorner)
	}
}

func TestTranslateMovesPointsButNotDirections(test *testing.T) {
	matrix := Translate(Vector{1, 2, 3, 0})
	if point := matrix.Apply(Vector{1, 1, 1, 1}); !vectorsApproximatelyEqual(point, Vector{2, 3, 4, 1}) {
		test.Fatalf("point moved to %v", point)
	}
	if direction := matrix.Apply(Vector{1, 1, 1, 0}); !vectorsApproximatelyEqual(direction, Vector{1, 1, 1, 0}) {
		test.Fatalf("direction moved to %v", direction)
	}
}

func TestScaleStretchesEachAxisByItsFactor(test *testing.T) {
	point := Scale(Vector{2, 3, 4, 0}).Apply(Vector{1, 1, 1, 1})
	if !vectorsApproximatelyEqual(point, Vector{2, 3, 4, 1}) {
		test.Fatalf("scaled to %v", point)
	}
}

func TestRotationsTurnAQuarterAroundTheirAxis(test *testing.T) {
	quarter := math.Pi / 2
	cases := []struct {
		name   string
		matrix Matrix
		from   Vector
		want   Vector
	}{
		{"around y, x goes to -z", RotateY(quarter), Vector{1, 0, 0, 1}, Vector{0, 0, -1, 1}},
		{"around x, y goes to z", RotateX(quarter), Vector{0, 1, 0, 1}, Vector{0, 0, 1, 1}},
		{"around z, x goes to y", Rotate(Vector{0, 0, 5, 0}, quarter), Vector{1, 0, 0, 1}, Vector{0, 1, 0, 1}},
	}
	for _, testCase := range cases {
		if got := testCase.matrix.Apply(testCase.from); !vectorsApproximatelyEqual(got, testCase.want) {
			test.Errorf("%s: got %v, want %v", testCase.name, got, testCase.want)
		}
	}
}

func TestMultiplyAppliesTheRightMatrixFirst(test *testing.T) {
	scaleThenMove := Translate(Vector{10, 0, 0, 0}).Multiply(Scale(Vector{2, 2, 2, 0}))
	if point := scaleThenMove.Apply(Vector{1, 0, 0, 1}); !vectorsApproximatelyEqual(point, Vector{12, 0, 0, 1}) {
		test.Fatalf("got %v", point)
	}
}

func TestCrossOfXAndYIsZ(test *testing.T) {
	if got := (Vector{1, 0, 0, 0}).Cross(Vector{0, 1, 0, 0}); got != (Vector{0, 0, 1, 0}) {
		test.Fatalf("got %v", got)
	}
}

func TestNormalizeGivesALengthOfOneAndLeavesZeroAlone(test *testing.T) {
	if got := (Vector{3, 0, 4, 1}).Normalize(); !vectorsApproximatelyEqual(got, Vector{0.6, 0, 0.8, 0}) {
		test.Fatalf("got %v", got)
	}
	if got := (Vector{}).Normalize(); got != (Vector{}) {
		test.Fatalf("zero vector became %v", got)
	}
}

func TestMoveShiftsEveryVertexInPlaceWithoutAllocating(test *testing.T) {
	vertices := []Vector{{0, 0, 0, 1}, {1, 1, 1, 1}}
	allocations := testing.AllocsPerRun(10, func() { Move(vertices, Vector{1, 0, 0, 0}) })
	if allocations != 0 {
		test.Fatalf("Move allocated %v times", allocations)
	}
	if vertices[0][0] != 11 || vertices[1][0] != 12 {
		test.Fatalf("got %v", vertices)
	}
}

func TestLookAtPutsTheTargetStraightAhead(test *testing.T) {
	view := LookAt(Vector{0, 0, 5, 1}, Vector{0, 0, 0, 1}, Vector{0, 1, 0, 0})
	if got := view.Apply(Vector{0, 0, 0, 1}); !vectorsApproximatelyEqual(got, Vector{0, 0, -5, 1}) {
		test.Fatalf("target seen at %v", got)
	}
}

func TestPerspectivesMapTheNearPlaneToEachBackendsDepth(test *testing.T) {
	cases := []struct {
		name       string
		projection Matrix
		nearDepth  float32
	}{
		{"OpenGL", PerspectiveOpenGL(math.Pi/2, 1, 1, 10), -1},
		{"Vulkan", PerspectiveVulkan(math.Pi/2, 1, 1, 10), 0},
		{"Metal", PerspectiveMetal(math.Pi/2, 1, 1, 10), 0},
	}
	for _, testCase := range cases {
		clip := testCase.projection.Apply(Vector{0, 0, -1, 1})
		if depth := clip[2] / clip[3]; !approximatelyEqual(depth, testCase.nearDepth) {
			test.Errorf("%s: near plane at depth %v, want %v", testCase.name, depth, testCase.nearDepth)
		}
	}
}
