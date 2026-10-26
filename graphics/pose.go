package graphics

import "math"

// Quaternion is a rotation, written as OpenXR (and glTF) give it: x, y and z are the axis times
// the sine of half the angle, w the cosine of half the angle. It is expected to be of length 1.
type Quaternion struct{ X, Y, Z, W float32 }

// IdentityQuaternion is the rotation that turns nothing.
func IdentityQuaternion() Quaternion { return Quaternion{W: 1} }

// QuaternionAroundY returns the rotation around the y axis (up) by radians, counter-clockwise
// seen from above, as RotateY.
func QuaternionAroundY(radians float64) Quaternion {
	half := radians / 2
	return Quaternion{Y: float32(math.Sin(half)), W: float32(math.Cos(half))}
}

// Matrix returns the matrix that turns points as the quaternion does.
func (rotation Quaternion) Matrix() Matrix {
	x, y, z, w := rotation.X, rotation.Y, rotation.Z, rotation.W
	return Matrix{
		1 - 2*(y*y+z*z), 2 * (x*y + z*w), 2 * (x*z - y*w), 0,
		2 * (x*y - z*w), 1 - 2*(x*x+z*z), 2 * (y*z + x*w), 0,
		2 * (x*z + y*w), 2 * (y*z - x*w), 1 - 2*(x*x+y*y), 0,
		0, 0, 0, 1,
	}
}

// Rotate returns vector turned by the quaternion; its w is kept, so points stay points.
func (rotation Quaternion) Rotate(vector Vector) Vector {
	turned := rotation.Matrix().Apply(Vector{vector[0], vector[1], vector[2], 0})
	turned[3] = vector[3]
	return turned
}

// Multiply returns rotation × other: the rotation that turns by other first, then by rotation.
func (rotation Quaternion) Multiply(other Quaternion) Quaternion {
	return Quaternion{
		X: rotation.W*other.X + rotation.X*other.W + rotation.Y*other.Z - rotation.Z*other.Y,
		Y: rotation.W*other.Y - rotation.X*other.Z + rotation.Y*other.W + rotation.Z*other.X,
		Z: rotation.W*other.Z + rotation.X*other.Y - rotation.Y*other.X + rotation.Z*other.W,
		W: rotation.W*other.W - rotation.X*other.X - rotation.Y*other.Y - rotation.Z*other.Z,
	}
}

// Conjugate returns the rotation that undoes this one (for a quaternion of length 1).
func (rotation Quaternion) Conjugate() Quaternion {
	return Quaternion{X: -rotation.X, Y: -rotation.Y, Z: -rotation.Z, W: rotation.W}
}

// Pose is where something is and which way it is turned, as a headset tracks an eye, the head or
// a hand: Position in metres in the headset's space (its w is ignored), Orientation turning the
// thing's own axes (x right, y up, looking down -z) into that space.
type Pose struct {
	Orientation Quaternion
	Position    Vector
}

// Matrix returns the matrix that takes the thing's own space to the space it is posed in: rotate,
// then move.
func (pose Pose) Matrix() Matrix {
	matrix := pose.Orientation.Matrix()
	matrix[12] = pose.Position[0] // column 3, row 0
	matrix[13] = pose.Position[1] // column 3, row 1
	matrix[14] = pose.Position[2] // column 3, row 2
	return matrix
}

// ViewMatrix returns the inverse of Matrix: from the space the eye is posed in to the eye's own
// space, what a view matrix is. A rotation is undone by its transpose, so no general inverse is
// needed.
func (pose Pose) ViewMatrix() Matrix {
	undo := pose.Orientation.Conjugate()
	moved := undo.Rotate(Vector{-pose.Position[0], -pose.Position[1], -pose.Position[2], 0})
	matrix := undo.Matrix()
	matrix[12] = moved[0]
	matrix[13] = moved[1]
	matrix[14] = moved[2]
	return matrix
}

// FieldOfView is how far an eye sees to each side, as OpenXR gives it: angles from the eye's
// forward direction, left and down negative. A headset's eyes are not symmetric: each sees more
// to the outside than towards the nose.
type FieldOfView struct{ LeftRadians, RightRadians, UpRadians, DownRadians float32 }

// PerspectiveFromFieldOfView returns the OpenGL projection (y up, depth from -1 at near to 1 at
// far) of an eye that sees field: an off-centre frustum, the four angles each at an edge of the
// picture. With left = -right and down = -up it is PerspectiveOpenGL's.
func PerspectiveFromFieldOfView(field FieldOfView, near, far float32) Matrix {
	tangentLeft := float32(math.Tan(float64(field.LeftRadians)))
	tangentRight := float32(math.Tan(float64(field.RightRadians)))
	tangentUp := float32(math.Tan(float64(field.UpRadians)))
	tangentDown := float32(math.Tan(float64(field.DownRadians)))
	width := tangentRight - tangentLeft
	height := tangentUp - tangentDown

	matrix := Matrix{}
	matrix[0] = 2 / width
	matrix[5] = 2 / height
	matrix[8] = (tangentRight + tangentLeft) / width // column 2, row 0: shifts the centre sideways
	matrix[9] = (tangentUp + tangentDown) / height   // column 2, row 1: and up or down
	matrix[10] = (far + near) / (near - far)
	matrix[11] = -1
	matrix[14] = (2 * far * near) / (near - far)
	return matrix
}
