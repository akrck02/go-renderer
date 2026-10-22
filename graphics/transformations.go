package graphics

import (
	"math"
)

// Vector is a point or a direction in 3D with a fourth value, w: [x, y, z, w]. Points carry w = 1
// and directions w = 0, so one matrix can both move points and turn directions. Sixteen bytes,
// the size GPUs expect, so it copies into OpenGL, Vulkan or Metal buffers as it is.
type Vector [4]float32

// Matrix is a 4 × 4 matrix flattened into 16 values, column after column (column-major), the
// order OpenGL, Vulkan and Metal read. Element (row, column) is at index column*4 + row.
type Matrix [16]float32

// matrixSize is the number of rows (and columns) of a Matrix.
const matrixSize = 4

// Identity returns the matrix that changes nothing: ones on the diagonal, zeros elsewhere.
func Identity() Matrix {
	return Matrix{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Multiply returns matrix × other: the transform that applies other first, then matrix.
func (matrix Matrix) Multiply(other Matrix) Matrix {
	var result Matrix
	for column := range matrixSize {
		columnStart := column * matrixSize
		for row := range matrixSize {
			result[columnStart+row] =
				matrix[0+row]*other[columnStart+0] +
					matrix[4+row]*other[columnStart+1] +
					matrix[8+row]*other[columnStart+2] +
					matrix[12+row]*other[columnStart+3]
		}
	}
	return result
}

// Translate returns the matrix that moves points by offset (its w is ignored).
func Translate(offset Vector) Matrix {
	matrix := Identity()
	matrix[12] = offset[0] // column 3, row 0
	matrix[13] = offset[1] // column 3, row 1
	matrix[14] = offset[2] // column 3, row 2
	return matrix
}

// Scale returns the matrix that stretches x, y and z by the factors of factors (its w is ignored).
func Scale(factors Vector) Matrix {
	matrix := Identity()
	matrix[0] = factors[0]  // column 0, row 0
	matrix[5] = factors[1]  // column 1, row 1
	matrix[10] = factors[2] // column 2, row 2
	return matrix
}

// RotateY returns the matrix that turns points around the y axis by radians.
func RotateY(radians float64) Matrix {
	matrix := Identity()
	cosine := float32(math.Cos(radians))
	sine := float32(math.Sin(radians))

	matrix[0] = cosine  // column 0, row 0
	matrix[2] = -sine   // column 0, row 2
	matrix[8] = sine    // column 2, row 0
	matrix[10] = cosine // column 2, row 2
	return matrix
}

// RotateX returns the matrix that turns points around the x axis by radians.
func RotateX(radians float64) Matrix {
	matrix := Identity()
	cosine := float32(math.Cos(radians))
	sine := float32(math.Sin(radians))

	matrix[5] = cosine  // column 1, row 1
	matrix[6] = sine    // column 1, row 2
	matrix[9] = -sine   // column 2, row 1
	matrix[10] = cosine // column 2, row 2
	return matrix
}

// Rotate returns the matrix that turns points around axis (any length but zero) by radians,
// counter-clockwise when the axis points at the viewer.
func Rotate(axis Vector, radians float64) Matrix {
	matrix := Identity()
	axis = axis.Normalize()
	x, y, z := axis[0], axis[1], axis[2]
	sine := float32(math.Sin(radians))
	cosine := float32(math.Cos(radians))
	oneMinusCosine := 1.0 - cosine

	matrix[0] = x*x*oneMinusCosine + cosine
	matrix[1] = y*x*oneMinusCosine + z*sine
	matrix[2] = z*x*oneMinusCosine - y*sine

	matrix[4] = x*y*oneMinusCosine - z*sine
	matrix[5] = y*y*oneMinusCosine + cosine
	matrix[6] = z*y*oneMinusCosine + x*sine

	matrix[8] = x*z*oneMinusCosine + y*sine
	matrix[9] = y*z*oneMinusCosine - x*sine
	matrix[10] = z*z*oneMinusCosine + cosine

	return matrix
}

// Apply returns vector transformed by the matrix (matrix × vector).
func (matrix Matrix) Apply(vector Vector) Vector {
	return Vector{
		matrix[0]*vector[0] + matrix[4]*vector[1] + matrix[8]*vector[2] + matrix[12]*vector[3],
		matrix[1]*vector[0] + matrix[5]*vector[1] + matrix[9]*vector[2] + matrix[13]*vector[3],
		matrix[2]*vector[0] + matrix[6]*vector[1] + matrix[10]*vector[2] + matrix[14]*vector[3],
		matrix[3]*vector[0] + matrix[7]*vector[1] + matrix[11]*vector[2] + matrix[15]*vector[3],
	}
}

// Cross returns the cross product of the two vectors' x, y and z: a direction square to both
// (w is ignored and the result's is 0).
func (vector Vector) Cross(other Vector) Vector {
	return Vector{
		vector[1]*other[2] - vector[2]*other[1],
		vector[2]*other[0] - vector[0]*other[2],
		vector[0]*other[1] - vector[1]*other[0],
		0.0,
	}
}

// Normalize returns the direction of the vector with a length of 1 (w becomes 0); the zero
// vector stays zero.
func (vector Vector) Normalize() Vector {
	length := float32(math.Sqrt(float64(vector[0]*vector[0] + vector[1]*vector[1] + vector[2]*vector[2])))
	if length == 0 {
		return Vector{}
	}
	return Vector{vector[0] / length, vector[1] / length, vector[2] / length, 0.0}
}

// Dot returns the dot product of the two vectors' x, y and z (w is ignored).
func (vector Vector) Dot(other Vector) float32 {
	return vector[0]*other[0] + vector[1]*other[1] + vector[2]*other[2]
}

// Add returns the sum of the two vectors, w included.
func (vector Vector) Add(other Vector) Vector {
	return Vector{vector[0] + other[0], vector[1] + other[1], vector[2] + other[2], vector[3] + other[3]}
}

// Move adds offset to every vertex, in place: it allocates nothing.
func Move(vertices []Vector, offset Vector) {
	for index := range vertices {
		vertices[index] = vertices[index].Add(offset)
	}
}

// LookAt returns the view matrix of an eye looking at target, with up saying which way is up
// (right-handed). It suits OpenGL, Vulkan and Metal alike: view space does not depend on the
// backend, only clip space does (see the Perspective functions).
func LookAt(eye, target, up Vector) Matrix {
	forward := Vector{target[0] - eye[0], target[1] - eye[1], target[2] - eye[2], 0.0}.Normalize()
	side := forward.Cross(up).Normalize()
	trueUp := side.Cross(forward)

	matrix := Identity()
	matrix[0] = side[0]
	matrix[4] = side[1]
	matrix[8] = side[2]

	matrix[1] = trueUp[0]
	matrix[5] = trueUp[1]
	matrix[9] = trueUp[2]

	matrix[2] = -forward[0]
	matrix[6] = -forward[1]
	matrix[10] = -forward[2]

	matrix[12] = -side.Dot(eye)
	matrix[13] = -trueUp.Dot(eye)
	matrix[14] = forward.Dot(eye)
	return matrix
}

// The projections below differ by backend: each GPU interface has its own clip space (which way
// y points, and whether depth runs from -1 or from 0).

// perspectiveTerms returns what every perspective shares: the focal length (one over the tangent
// of half the vertical field of view) and the arguments as float32.
func perspectiveTerms(fieldOfViewRadians, aspect, near, far float64) (focalLength, aspect32, near32, far32 float32) {
	return float32(1.0 / math.Tan(fieldOfViewRadians/2.0)), float32(aspect), float32(near), float32(far)
}

// PerspectiveOpenGL returns a perspective projection for OpenGL: y points up and depth runs from
// -1 (near) to 1 (far). fieldOfViewRadians is vertical; aspect is width over height.
func PerspectiveOpenGL(fieldOfViewRadians, aspect, near, far float64) Matrix {
	focalLength, aspect32, near32, far32 := perspectiveTerms(fieldOfViewRadians, aspect, near, far)
	matrix := Matrix{}
	matrix[0] = focalLength / aspect32
	matrix[5] = focalLength
	matrix[10] = (far32 + near32) / (near32 - far32)
	matrix[11] = -1.0
	matrix[14] = (2.0 * far32 * near32) / (near32 - far32)
	return matrix
}

// PerspectiveVulkan returns a perspective projection for Vulkan: y points down and depth runs
// from 0 (near) to 1 (far).
func PerspectiveVulkan(fieldOfViewRadians, aspect, near, far float64) Matrix {
	focalLength, aspect32, near32, far32 := perspectiveTerms(fieldOfViewRadians, aspect, near, far)
	matrix := Matrix{}
	matrix[0] = focalLength / aspect32
	matrix[5] = -focalLength // Vulkan's y points down
	matrix[10] = far32 / (near32 - far32)
	matrix[11] = -1.0
	matrix[14] = (near32 * far32) / (near32 - far32)
	return matrix
}

// PerspectiveMetal returns a perspective projection for Metal: y points up, as in OpenGL, and
// depth runs from 0 (near) to 1 (far), as in Vulkan.
func PerspectiveMetal(fieldOfViewRadians, aspect, near, far float64) Matrix {
	focalLength, aspect32, near32, far32 := perspectiveTerms(fieldOfViewRadians, aspect, near, far)
	matrix := Matrix{}
	matrix[0] = focalLength / aspect32
	matrix[5] = focalLength
	matrix[10] = far32 / (near32 - far32)
	matrix[11] = -1.0
	matrix[14] = (near32 * far32) / (near32 - far32)
	return matrix
}

// Inverse returns the matrix that undoes this one, and false for a matrix that cannot be undone
// (its determinant is zero, as when a scale squashes an axis to nothing).
func (matrix Matrix) Inverse() (Matrix, bool) {
	inverse := cofactors(matrix)
	determinant := matrix[0]*inverse[0] + matrix[1]*inverse[4] + matrix[2]*inverse[8] + matrix[3]*inverse[12]
	if determinant == 0 {
		return Matrix{}, false
	}
	for element := range inverse {
		inverse[element] /= determinant
	}
	return inverse, true
}

// cofactors returns the adjugate of the matrix (the transposed matrix of its cofactors): the
// inverse times the determinant, written out in full.
func cofactors(matrix Matrix) Matrix {
	var result Matrix
	result[0] = matrix[5]*matrix[10]*matrix[15] - matrix[5]*matrix[11]*matrix[14] - matrix[9]*matrix[6]*matrix[15] + matrix[9]*matrix[7]*matrix[14] + matrix[13]*matrix[6]*matrix[11] - matrix[13]*matrix[7]*matrix[10]
	result[4] = -matrix[4]*matrix[10]*matrix[15] + matrix[4]*matrix[11]*matrix[14] + matrix[8]*matrix[6]*matrix[15] - matrix[8]*matrix[7]*matrix[14] - matrix[12]*matrix[6]*matrix[11] + matrix[12]*matrix[7]*matrix[10]
	result[8] = matrix[4]*matrix[9]*matrix[15] - matrix[4]*matrix[11]*matrix[13] - matrix[8]*matrix[5]*matrix[15] + matrix[8]*matrix[7]*matrix[13] + matrix[12]*matrix[5]*matrix[11] - matrix[12]*matrix[7]*matrix[9]
	result[12] = -matrix[4]*matrix[9]*matrix[14] + matrix[4]*matrix[10]*matrix[13] + matrix[8]*matrix[5]*matrix[14] - matrix[8]*matrix[6]*matrix[13] - matrix[12]*matrix[5]*matrix[10] + matrix[12]*matrix[6]*matrix[9]
	result[1] = -matrix[1]*matrix[10]*matrix[15] + matrix[1]*matrix[11]*matrix[14] + matrix[9]*matrix[2]*matrix[15] - matrix[9]*matrix[3]*matrix[14] - matrix[13]*matrix[2]*matrix[11] + matrix[13]*matrix[3]*matrix[10]
	result[5] = matrix[0]*matrix[10]*matrix[15] - matrix[0]*matrix[11]*matrix[14] - matrix[8]*matrix[2]*matrix[15] + matrix[8]*matrix[3]*matrix[14] + matrix[12]*matrix[2]*matrix[11] - matrix[12]*matrix[3]*matrix[10]
	result[9] = -matrix[0]*matrix[9]*matrix[15] + matrix[0]*matrix[11]*matrix[13] + matrix[8]*matrix[1]*matrix[15] - matrix[8]*matrix[3]*matrix[13] - matrix[12]*matrix[1]*matrix[11] + matrix[12]*matrix[3]*matrix[9]
	result[13] = matrix[0]*matrix[9]*matrix[14] - matrix[0]*matrix[10]*matrix[13] - matrix[8]*matrix[1]*matrix[14] + matrix[8]*matrix[2]*matrix[13] + matrix[12]*matrix[1]*matrix[10] - matrix[12]*matrix[2]*matrix[9]
	result[2] = matrix[1]*matrix[6]*matrix[15] - matrix[1]*matrix[7]*matrix[14] - matrix[5]*matrix[2]*matrix[15] + matrix[5]*matrix[3]*matrix[14] + matrix[13]*matrix[2]*matrix[7] - matrix[13]*matrix[3]*matrix[6]
	result[6] = -matrix[0]*matrix[6]*matrix[15] + matrix[0]*matrix[7]*matrix[14] + matrix[4]*matrix[2]*matrix[15] - matrix[4]*matrix[3]*matrix[14] - matrix[12]*matrix[2]*matrix[7] + matrix[12]*matrix[3]*matrix[6]
	result[10] = matrix[0]*matrix[5]*matrix[15] - matrix[0]*matrix[7]*matrix[13] - matrix[4]*matrix[1]*matrix[15] + matrix[4]*matrix[3]*matrix[13] + matrix[12]*matrix[1]*matrix[7] - matrix[12]*matrix[3]*matrix[5]
	result[14] = -matrix[0]*matrix[5]*matrix[14] + matrix[0]*matrix[6]*matrix[13] + matrix[4]*matrix[1]*matrix[14] - matrix[4]*matrix[2]*matrix[13] - matrix[12]*matrix[1]*matrix[6] + matrix[12]*matrix[2]*matrix[5]
	result[3] = -matrix[1]*matrix[6]*matrix[11] + matrix[1]*matrix[7]*matrix[10] + matrix[5]*matrix[2]*matrix[11] - matrix[5]*matrix[3]*matrix[10] - matrix[9]*matrix[2]*matrix[7] + matrix[9]*matrix[3]*matrix[6]
	result[7] = matrix[0]*matrix[6]*matrix[11] - matrix[0]*matrix[7]*matrix[10] - matrix[4]*matrix[2]*matrix[11] + matrix[4]*matrix[3]*matrix[10] + matrix[8]*matrix[2]*matrix[7] - matrix[8]*matrix[3]*matrix[6]
	result[11] = -matrix[0]*matrix[5]*matrix[11] + matrix[0]*matrix[7]*matrix[9] + matrix[4]*matrix[1]*matrix[11] - matrix[4]*matrix[3]*matrix[9] - matrix[8]*matrix[1]*matrix[7] + matrix[8]*matrix[3]*matrix[5]
	result[15] = matrix[0]*matrix[5]*matrix[10] - matrix[0]*matrix[6]*matrix[9] - matrix[4]*matrix[1]*matrix[10] + matrix[4]*matrix[2]*matrix[9] + matrix[8]*matrix[1]*matrix[6] - matrix[8]*matrix[2]*matrix[5]
	return result
}

// FromTranslationRotationScale returns the matrix that scales, then rotates by the quaternion
// rotation (x, y, z, w), then moves by translation: the order glTF nodes use.
func FromTranslationRotationScale(translation Vector, rotation Vector, scale Vector) Matrix {
	x, y, z, w := rotation[0], rotation[1], rotation[2], rotation[3]
	return Matrix{
		(1 - 2*(y*y+z*z)) * scale[0], 2 * (x*y + z*w) * scale[0], 2 * (x*z - y*w) * scale[0], 0,
		2 * (x*y - z*w) * scale[1], (1 - 2*(x*x+z*z)) * scale[1], 2 * (y*z + x*w) * scale[1], 0,
		2 * (x*z + y*w) * scale[2], 2 * (y*z - x*w) * scale[2], (1 - 2*(x*x+y*y)) * scale[2], 0,
		translation[0], translation[1], translation[2], 1,
	}
}

// OrthographicOpenGL returns a projection without perspective for OpenGL: the box from (left,
// bottom, -near) to (right, top, -far) fills the screen, depth mapped to [-1, 1].
func OrthographicOpenGL(left, right, bottom, top, near, far float32) Matrix {
	return Matrix{
		2 / (right - left), 0, 0, 0,
		0, 2 / (top - bottom), 0, 0,
		0, 0, -2 / (far - near), 0,
		-(right + left) / (right - left), -(top + bottom) / (top - bottom), -(far + near) / (far - near), 1,
	}
}
