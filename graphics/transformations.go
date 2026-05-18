package graphics

import (
	"math"
)

// Vec4 represents a 4D vector or 3D coordinate with padding [X, Y, Z, W].
// Aligns perfectly to 16 bytes, matching GPU hardware expectations across all APIs.
type Vec4 [4]float32

// Mat4 represents a 4x4 matrix flattened into a column-major 16-element array.
// Allows direct, zero-allocation memory copying into Vulkan, OpenGL, or Metal buffers.
type Mat4 [16]float32

// Identity returns a 4x4 identity matrix.
func Identity() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Multiply multiplies two 4x4 matrices (A * B).
func (a Mat4) Multiply(b Mat4) Mat4 {
	var res Mat4
	for col := range 4 {
		bColIdx := col * 4
		for row := range 4 {
			res[bColIdx+row] =
				a[0+row]*b[bColIdx+0] +
					a[4+row]*b[bColIdx+1] +
					a[8+row]*b[bColIdx+2] +
					a[12+row]*b[bColIdx+3]
		}
	}
	return res
}

// Translate returns a translation matrix using a Vec4 (ignoring W component).
func Translate(v Vec4) Mat4 {
	m := Identity()
	m[12] = v[0] // Column 3, Row 0
	m[13] = v[1] // Column 3, Row 1
	m[14] = v[2] // Column 3, Row 2
	return m
}

// Scale returns a scaling matrix using a Vec4.
func Scale(v Vec4) Mat4 {
	m := Identity()
	m[0] = v[0]  // Column 0, Row 0
	m[5] = v[1]  // Column 1, Row 1
	m[10] = v[2] // Column 2, Row 2
	return m
}

// RotateY returns a rotation matrix around the Y axis (in radians).
func RotateY(rad float64) Mat4 {
	m := Identity()
	cos := float32(math.Cos(rad))
	sin := float32(math.Sin(rad))

	m[0] = cos  // Col 0, Row 0
	m[2] = -sin // Col 0, Row 2
	m[8] = sin  // Col 2, Row 0
	m[10] = cos // Col 2, Row 2
	return m
}

// RotateX returns a rotation matrix around the X axis (in radians).
func RotateX(rad float64) Mat4 {
	m := Identity()
	cos := float32(math.Cos(rad))
	sin := float32(math.Sin(rad))

	m[5] = cos   // Col 1, Row 1
	m[6] = sin   // Col 1, Row 2
	m[9] = -sin  // Col 2, Row 1
	m[10] = cos  // Col 2, Row 2
	return m
}

// Apply transforms a Vec4 using the matrix (m * v).
func (m Mat4) Apply(v Vec4) Vec4 {
	return Vec4{
		m[0]*v[0] + m[4]*v[1] + m[8]*v[2] + m[12]*v[3],
		m[1]*v[0] + m[5]*v[1] + m[9]*v[2] + m[13]*v[3],
		m[2]*v[0] + m[6]*v[1] + m[10]*v[2] + m[14]*v[3],
		m[3]*v[0] + m[7]*v[1] + m[11]*v[2] + m[15]*v[3],
	}
}

// Cross computes the 3D cross product of two Vec4 vectors (W component is ignored).
func (v Vec4) Cross(other Vec4) Vec4 {
	return Vec4{
		v[1]*other[2] - v[2]*other[1],
		v[2]*other[0] - v[0]*other[2],
		v[0]*other[1] - v[1]*other[0],
		0.0,
	}
}

// Normalize returns the unit vector of a Vec4.
func (v Vec4) Normalize() Vec4 {
	length := float32(math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])))
	if length == 0 {
		return Vec4{}
	}
	return Vec4{v[0] / length, v[1] / length, v[2] / length, 0.0}
}

// Dot computes the 3D dot product of two vectors (ignoring W).
func (v Vec4) Dot(other Vec4) float32 {
	return v[0]*other[0] + v[1]*other[1] + v[2]*other[2]
}

// Add returns a new Vec4 that is the sum of v and other.
func (v Vec4) Add(other Vec4) Vec4 {
	return Vec4{v[0] + other[0], v[1] + other[1], v[2] + other[2], v[3] + other[3]}
}

// Move applies a translation to a slice of Vec4 vertices.
func Move(vertices []Vec4, delta Vec4) []Vec4 {
	res := make([]Vec4, len(vertices))
	for i, v := range vertices {
		res[i] = v.Add(delta)
	}
	return res
}

// LookAt creates a Right-Handed View Matrix.
// Works universally for OpenGL, Vulkan, and Metal since view space is independent of clip space.
func LookAt(eye, center, up Vec4) Mat4 {
	f := Vec4{center[0] - eye[0], center[1] - eye[1], center[2] - eye[2], 0.0}.Normalize()
	s := f.Cross(up).Normalize()
	u := s.Cross(f)

	m := Identity()
	m[0] = s[0]
	m[4] = s[1]
	m[8] = s[2]

	m[1] = u[0]
	m[5] = u[1]
	m[9] = u[2]

	m[2] = -f[0]
	m[6] = -f[1]
	m[10] = -f[2]

	m[12] = -s.Dot(eye)
	m[13] = -u.Dot(eye)
	m[14] = f.Dot(eye)
	return m
}

// --- API-SPECIFIC PROJECTION MATRICES ---

// PerspectiveOpenGL creates a Projection Matrix tailored for OpenGL.
// - Y-axis points UP (+1 is top)
// - Depth (Z) maps to [-1, 1]
func PerspectiveOpenGL(fovyRad, aspect, near, far float64) Mat4 {
	g := float32(1.0 / math.Tan(fovyRad/2.0))
	n := float32(near)
	f := float32(far)
	asp := float32(aspect)

	m := Mat4{}
	m[0] = g / asp
	m[5] = g
	m[10] = (f + n) / (n - f)
	m[11] = -1.0
	m[14] = (2.0 * f * n) / (n - f)
	return m
}

// PerspectiveVulkan creates a Projection Matrix tailored for Vulkan.
// - Y-axis points DOWN (+1 is bottom)
// - Depth (Z) maps to [0, 1]
func PerspectiveVulkan(fovyRad, aspect, near, far float64) Mat4 {
	g := float32(1.0 / math.Tan(fovyRad/2.0))
	n := float32(near)
	f := float32(far)
	asp := float32(aspect)

	m := Mat4{}
	m[0] = g / asp
	m[5] = -g // Negative flattens Y axis natively
	m[10] = f / (n - f)
	m[11] = -1.0
	m[14] = (n * f) / (n - f)
	return m
}

// PerspectiveMetal creates a Projection Matrix tailored for Apple's Metal API.
// - Y-axis points UP (+1 is top, same as OpenGL)
// - Depth (Z) maps to [0, 1] (same as Vulkan)
func PerspectiveMetal(fovyRad, aspect, near, far float64) Mat4 {
	g := float32(1.0 / math.Tan(fovyRad/2.0))
	n := float32(near)
	f := float32(far)
	asp := float32(aspect)

	m := Mat4{}
	m[0] = g / asp
	m[5] = g
	m[10] = f / (n - f)
	m[11] = -1.0
	m[14] = (n * f) / (n - f)
	return m
}
