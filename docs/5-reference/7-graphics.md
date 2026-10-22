# graphics

```go
import "github.com/akrck02/go-renderer/graphics"
```

### Constants and variables

```go
const (
	LocalSpace  = 0
	WorldSpace  = 1
	CameraSpace = 2
)
```

```go
const FragmentShaderSource = …
```

```go
const VertexShaderSource = …
```

### CoordinateSpace

```go
type CoordinateSpace int
```

### Matrix

Matrix represents a 4x4 matrix flattened into a column-major 16-element array. Allows direct, zero-allocation memory copying into Vulkan, OpenGL, or Metal buffers.

```go
type Matrix [16]float32
```

| Function or method | |
|---|---|
| `func FromTranslationRotationScale(translation Vector, rotation Vector, scale Vector) Matrix` | FromTranslationRotationScale builds a matrix from a translation, a rotation quaternion (x, y, z, w) and a scale. |
| `func Identity() Matrix` | Identity returns a 4x4 identity matrix. |
| `func LookAt(eye, center, up Vector) Matrix` | LookAt creates a Right-Handed View Matrix. |
| `func OrthographicOpenGL(left, right, bottom, top, near, far float32) Matrix` | OrthographicOpenGL creates an orthographic projection for OpenGL (depth mapped to [-1, 1]). |
| `func PerspectiveMetal(fovyRad, aspect, near, far float64) Matrix` | PerspectiveMetal creates a Projection Matrix tailored for Apple's Metal API. |
| `func PerspectiveOpenGL(fovyRad, aspect, near, far float64) Matrix` | PerspectiveOpenGL creates a Projection Matrix tailored for OpenGL. |
| `func PerspectiveVulkan(fovyRad, aspect, near, far float64) Matrix` | PerspectiveVulkan creates a Projection Matrix tailored for Vulkan. |
| `func Rotate(axis Vector, rad float64) Matrix` | Rotate returns a rotation matrix around an arbitrary axis. |
| `func RotateX(rad float64) Matrix` | RotateX returns a rotation matrix around the X axis (in radians). |
| `func RotateY(rad float64) Matrix` | RotateY returns a rotation matrix around the Y axis (in radians). |
| `func Scale(v Vector) Matrix` | Scale returns a scaling matrix using a Vector. |
| `func Translate(v Vector) Matrix` | Translate returns a translation matrix using a Vector (ignoring W component). |
| `func (m Matrix) Apply(v Vector) Vector` | Apply transforms a Vector using the matrix (m * v). |
| `func (m Matrix) Inverse() (Matrix, bool)` | Inverse returns the inverse of a 4x4 matrix and false when it is singular. |
| `func (a Matrix) Multiply(b Matrix) Matrix` | Multiply multiplies two 4x4 matrices (A * B). |

### Vector

Vector represents a 4D vector or 3D coordinate with padding [X, Y, Z, W]. Aligns perfectly to 16 bytes, matching GPU hardware expectations across all APIs.

```go
type Vector [4]float32
```

| Function or method | |
|---|---|
| `func Move(vertices []Vector, delta Vector) []Vector` | Move applies a translation to a slice of Vector vertices. |
| `func (v Vector) Add(other Vector) Vector` | Add returns a new Vector that is the sum of v and other. |
| `func (v Vector) Cross(other Vector) Vector` | Cross computes the 3D cross product of two Vector vectors (W component is ignored). |
| `func (v Vector) Dot(other Vector) float32` | Dot computes the 3D dot product of two vectors (ignoring W). |
| `func (v Vector) Normalize() Vector` | Normalize returns the unit vector of a Vector. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
