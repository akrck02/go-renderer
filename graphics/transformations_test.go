package graphics

import (
	"math"
	"testing"
)

func approximatelyEqual(first, second float32) bool { return math.Abs(float64(first-second)) < 1e-4 }

func TestInverseRoundTrip(t *testing.T) {
	matrix := Translate(Vec4{3, -2, 5, 0}).Multiply(RotateY(0.7)).Multiply(Scale(Vec4{2, 3, 0.5, 1}))
	inverse, invertible := matrix.Inverse()
	if !invertible {
		t.Fatal("matrix reported singular")
	}
	product := matrix.Multiply(inverse)
	identity := Identity()
	for element := range product {
		if !approximatelyEqual(product[element], identity[element]) {
			t.Fatalf("matrix * inverse != identity at %d: %v", element, product)
		}
	}
	if _, invertible := (Mat4{}).Inverse(); invertible {
		t.Fatal("zero matrix should be singular")
	}
}

func TestFromTranslationRotationScale(t *testing.T) {
	// 90 degrees around Y: +X goes to -Z
	halfSquareRoot := float32(math.Sqrt(0.5))
	matrix := FromTranslationRotationScale(Vec4{1, 2, 3, 0}, Vec4{0, halfSquareRoot, 0, halfSquareRoot}, Vec4{2, 2, 2, 0})
	point := matrix.Apply(Vec4{1, 0, 0, 1})
	if !approximatelyEqual(point[0], 1) || !approximatelyEqual(point[1], 2) || !approximatelyEqual(point[2], 1) {
		t.Fatalf("unexpected transform: %v", point)
	}
}

func TestOrthographicMapsBoxToClipCube(t *testing.T) {
	projection := OrthographicOpenGL(-2, 6, -1, 3, 1, 11)
	nearCorner := projection.Apply(Vec4{-2, -1, -1, 1})
	farCorner := projection.Apply(Vec4{6, 3, -11, 1})
	for axis, want := range []float32{-1, -1, -1} {
		if !approximatelyEqual(nearCorner[axis], want) {
			t.Fatalf("near corner maps to %v", nearCorner)
		}
	}
	for axis, want := range []float32{1, 1, 1} {
		if !approximatelyEqual(farCorner[axis], want) {
			t.Fatalf("far corner maps to %v", farCorner)
		}
	}
}
