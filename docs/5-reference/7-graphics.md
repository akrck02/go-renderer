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

### Mat4

Mat4 represents a 4x4 matrix flattened into a column-major 16-element array. Allows direct, zero-allocation memory copying into Vulkan, OpenGL, or Metal buffers.

```go
type Mat4 [16]float32
```

| Function or method | |
|---|---|
| `func FromTranslationRotationScale(translation Vec4, rotation Vec4, scale Vec4) Mat4` | FromTranslationRotationScale builds a matrix from a translation, a rotation quaternion (x, y, z, w) and a scale. |
| `func Identity() Mat4` | Identity returns a 4x4 identity matrix. |
| `func LookAt(eye, center, up Vec4) Mat4` | LookAt creates a Right-Handed View Matrix. |
| `func OrthographicOpenGL(left, right, bottom, top, near, far float32) Mat4` | OrthographicOpenGL creates an orthographic projection for OpenGL (depth mapped to [-1, 1]). |
| `func PerspectiveMetal(fovyRad, aspect, near, far float64) Mat4` | PerspectiveMetal creates a Projection Matrix tailored for Apple's Metal API. |
| `func PerspectiveOpenGL(fovyRad, aspect, near, far float64) Mat4` | PerspectiveOpenGL creates a Projection Matrix tailored for OpenGL. |
| `func PerspectiveVulkan(fovyRad, aspect, near, far float64) Mat4` | PerspectiveVulkan creates a Projection Matrix tailored for Vulkan. |
| `func Rotate(axis Vec4, rad float64) Mat4` | Rotate returns a rotation matrix around an arbitrary axis. |
| `func RotateX(rad float64) Mat4` | RotateX returns a rotation matrix around the X axis (in radians). |
| `func RotateY(rad float64) Mat4` | RotateY returns a rotation matrix around the Y axis (in radians). |
| `func Scale(v Vec4) Mat4` | Scale returns a scaling matrix using a Vec4. |
| `func Translate(v Vec4) Mat4` | Translate returns a translation matrix using a Vec4 (ignoring W component). |
| `func (m Mat4) Apply(v Vec4) Vec4` | Apply transforms a Vec4 using the matrix (m * v). |
| `func (m Mat4) Inverse() (Mat4, bool)` | Inverse returns the inverse of a 4x4 matrix and false when it is singular. |
| `func (a Mat4) Multiply(b Mat4) Mat4` | Multiply multiplies two 4x4 matrices (A * B). |

### Vec4

Vec4 represents a 4D vector or 3D coordinate with padding [X, Y, Z, W]. Aligns perfectly to 16 bytes, matching GPU hardware expectations across all APIs.

```go
type Vec4 [4]float32
```

| Function or method | |
|---|---|
| `func Move(vertices []Vec4, delta Vec4) []Vec4` | Move applies a translation to a slice of Vec4 vertices. |
| `func (v Vec4) Add(other Vec4) Vec4` | Add returns a new Vec4 that is the sum of v and other. |
| `func (v Vec4) Cross(other Vec4) Vec4` | Cross computes the 3D cross product of two Vec4 vectors (W component is ignored). |
| `func (v Vec4) Dot(other Vec4) float32` | Dot computes the 3D dot product of two vectors (ignoring W). |
| `func (v Vec4) Normalize() Vec4` | Normalize returns the unit vector of a Vec4. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
