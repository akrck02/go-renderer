package graphics

import (
	"math"
	"testing"
)

func TestAQuaternionTurnsLikeItsMatrix(test *testing.T) {
	rotation := QuaternionAroundY(math.Pi / 2)
	forward := Vector{0, 0, -1, 0}
	turned := rotation.Rotate(forward)
	// a quarter turn counter-clockwise seen from above takes -z to -x
	if !vectorsApproximatelyEqual(turned, Vector{-1, 0, 0, 0}) {
		test.Fatalf("turned = %v, want -x", turned)
	}
	if fromMatrix := RotateY(math.Pi / 2).Apply(forward); !vectorsApproximatelyEqual(fromMatrix, turned) {
		test.Fatalf("RotateY gives %v, the quaternion %v", fromMatrix, turned)
	}
}

func TestMultiplyingQuaternionsAppliesTheRightOneFirst(test *testing.T) {
	first := Quaternion{X: float32(math.Sin(0.3)), W: float32(math.Cos(0.3))} // around x
	second := QuaternionAroundY(0.9)
	point := Vector{1, 2, 3, 1}
	together := second.Multiply(first).Rotate(point)
	oneThenOther := second.Rotate(first.Rotate(point))
	if !vectorsApproximatelyEqual(together, oneThenOther) {
		test.Fatalf("product turns to %v, one after the other to %v", together, oneThenOther)
	}
}

func TestAPoseViewMatrixUndoesItsMatrix(test *testing.T) {
	pose := Pose{
		Orientation: QuaternionAroundY(0.4).Multiply(Quaternion{X: float32(math.Sin(0.1)), W: float32(math.Cos(0.1))}),
		Position:    Vector{1, 1.5, -2, 1},
	}
	product := pose.ViewMatrix().Multiply(pose.Matrix())
	identity := Identity()
	for element := range product {
		if !approximatelyEqual(product[element], identity[element]) {
			test.Fatalf("view × pose != identity at %d: %v", element, product)
		}
	}
	if eye := pose.ViewMatrix().Apply(pose.Position); !vectorsApproximatelyEqual(eye, Vector{0, 0, 0, 1}) {
		test.Fatalf("the pose's own position seen from it = %v, want the origin", eye)
	}
}

func TestASymmetricFieldOfViewIsTheUsualPerspective(test *testing.T) {
	halfVertical := float32(0.5)
	aspect := float32(1.5)
	halfHorizontal := float32(math.Atan(math.Tan(float64(halfVertical)) * float64(aspect)))
	field := FieldOfView{LeftRadians: -halfHorizontal, RightRadians: halfHorizontal, UpRadians: halfVertical, DownRadians: -halfVertical}
	fromField := PerspectiveFromFieldOfView(field, 0.1, 100)
	usual := PerspectiveOpenGL(float64(2*halfVertical), float64(aspect), 0.1, 100)
	for element := range usual {
		if !approximatelyEqual(fromField[element], usual[element]) {
			test.Fatalf("element %d = %v, PerspectiveOpenGL has %v", element, fromField[element], usual[element])
		}
	}
}

func TestAnAsymmetricFieldOfViewPutsItsEdgesAtTheEdgesOfThePicture(test *testing.T) {
	field := FieldOfView{LeftRadians: -0.9, RightRadians: 0.6, UpRadians: 0.8, DownRadians: -0.7}
	projection := PerspectiveFromFieldOfView(field, 0.1, 100)
	distance := 5.0
	edges := []struct {
		name  string
		point Vector
		axis  int
		want  float32
	}{
		{"left", Vector{float32(math.Tan(-0.9) * distance), 0, float32(-distance), 1}, 0, -1},
		{"right", Vector{float32(math.Tan(0.6) * distance), 0, float32(-distance), 1}, 0, 1},
		{"top", Vector{0, float32(math.Tan(0.8) * distance), float32(-distance), 1}, 1, 1},
		{"bottom", Vector{0, float32(math.Tan(-0.7) * distance), float32(-distance), 1}, 1, -1},
	}
	for _, edge := range edges {
		clip := projection.Apply(edge.point)
		if got := clip[edge.axis] / clip[3]; !approximatelyEqual(got, edge.want) {
			test.Fatalf("%s edge lands at %v, want %v", edge.name, got, edge.want)
		}
	}
	near := projection.Apply(Vector{0, 0, -0.1, 1})
	far := projection.Apply(Vector{0, 0, -100, 1})
	if !approximatelyEqual(near[2]/near[3], -1) || math.Abs(float64(far[2]/far[3]-1)) > 1e-3 {
		test.Fatalf("depth runs from %v to %v, want -1 to 1", near[2]/near[3], far[2]/far[3])
	}
}
