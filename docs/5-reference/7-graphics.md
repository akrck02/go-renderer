# graphics

```go
import "github.com/akrck02/go-renderer/graphics"
```

### Functions

| Function | |
|---|---|
| `func Move(vertices []Vector, offset Vector)` | Move adds offset to every vertex, in place: it allocates nothing. |

### CoordinateSpace

CoordinateSpace says which space a shape's coordinates are in.

```go
type CoordinateSpace int
```

```go
const (
	// LocalSpace is the shape's own space, before the model matrix places it.
	LocalSpace CoordinateSpace = iota
	// WorldSpace is the scene's space, shared by every shape.
	WorldSpace
	// CameraSpace is the space seen from the camera, after the view matrix.
	CameraSpace
)
```

The spaces a shape can be given in.

### FieldOfView

FieldOfView is how far an eye sees to each side, as OpenXR gives it: angles from the eye's forward direction, left and down negative. A headset's eyes are not symmetric: each sees more to the outside than towards the nose.

```go
type FieldOfView struct{ LeftRadians, RightRadians, UpRadians, DownRadians float32 }
```

### Matrix

Matrix is a 4 × 4 matrix flattened into 16 values, column after column (column-major), the order OpenGL, Vulkan and Metal read. Element (row, column) is at index column*4 + row.

```go
type Matrix [16]float32
```

| Function or method | |
|---|---|
| `func FromTranslationRotationScale(translation Vector, rotation Vector, scale Vector) Matrix` | FromTranslationRotationScale returns the matrix that scales, then rotates by the quaternion rotation (x, y, z, w), then moves by translation: the order glTF nodes use. |
| `func Identity() Matrix` | Identity returns the matrix that changes nothing: ones on the diagonal, zeros elsewhere. |
| `func LookAt(eye, target, up Vector) Matrix` | LookAt returns the view matrix of an eye looking at target, with up saying which way is up (right-handed). |
| `func OrthographicOpenGL(left, right, bottom, top, near, far float32) Matrix` | OrthographicOpenGL returns a projection without perspective for OpenGL: the box from (left, bottom, -near) to (right, top, -far) fills the screen, depth mapped to [-1, 1]. |
| `func PerspectiveFromFieldOfView(field FieldOfView, near, far float32) Matrix` | PerspectiveFromFieldOfView returns the OpenGL projection (y up, depth from -1 at near to 1 at far) of an eye that sees field: an off-centre frustum, the four angles each at an edge of the picture. |
| `func PerspectiveMetal(fieldOfViewRadians, aspect, near, far float64) Matrix` | PerspectiveMetal returns a perspective projection for Metal: y points up, as in OpenGL, and depth runs from 0 (near) to 1 (far), as in Vulkan. |
| `func PerspectiveOpenGL(fieldOfViewRadians, aspect, near, far float64) Matrix` | PerspectiveOpenGL returns a perspective projection for OpenGL: y points up and depth runs from -1 (near) to 1 (far). |
| `func PerspectiveVulkan(fieldOfViewRadians, aspect, near, far float64) Matrix` | PerspectiveVulkan returns a perspective projection for Vulkan: y points down and depth runs from 0 (near) to 1 (far). |
| `func Rotate(axis Vector, radians float64) Matrix` | Rotate returns the matrix that turns points around axis (any length but zero) by radians, counter-clockwise when the axis points at the viewer. |
| `func RotateX(radians float64) Matrix` | RotateX returns the matrix that turns points around the x axis by radians. |
| `func RotateY(radians float64) Matrix` | RotateY returns the matrix that turns points around the y axis by radians. |
| `func Scale(factors Vector) Matrix` | Scale returns the matrix that stretches x, y and z by the factors of factors (its w is ignored). |
| `func Translate(offset Vector) Matrix` | Translate returns the matrix that moves points by offset (its w is ignored). |
| `func (matrix Matrix) Apply(vector Vector) Vector` | Apply returns vector transformed by the matrix (matrix × vector). |
| `func (matrix Matrix) Inverse() (Matrix, bool)` | Inverse returns the matrix that undoes this one, and false for a matrix that cannot be undone (its determinant is zero, as when a scale squashes an axis to nothing). |
| `func (matrix Matrix) Multiply(other Matrix) Matrix` | Multiply returns matrix × other: the transform that applies other first, then matrix. |

### Pose

Pose is where something is and which way it is turned, as a headset tracks an eye, the head or a hand: Position in metres in the headset's space (its w is ignored), Orientation turning the thing's own axes (x right, y up, looking down -z) into that space.

```go
type Pose struct {
	Orientation Quaternion
	Position    Vector
}
```

| Function or method | |
|---|---|
| `func (pose Pose) Matrix() Matrix` | Matrix returns the matrix that takes the thing's own space to the space it is posed in: rotate, then move. |
| `func (pose Pose) ViewMatrix() Matrix` | ViewMatrix returns the inverse of Matrix: from the space the eye is posed in to the eye's own space, what a view matrix is. |

### Quaternion

Quaternion is a rotation, written as OpenXR (and glTF) give it: x, y and z are the axis times the sine of half the angle, w the cosine of half the angle. It is expected to be of length 1.

```go
type Quaternion struct{ X, Y, Z, W float32 }
```

| Function or method | |
|---|---|
| `func IdentityQuaternion() Quaternion` | IdentityQuaternion is the rotation that turns nothing. |
| `func QuaternionAroundY(radians float64) Quaternion` | QuaternionAroundY returns the rotation around the y axis (up) by radians, counter-clockwise seen from above, as RotateY. |
| `func (rotation Quaternion) Conjugate() Quaternion` | Conjugate returns the rotation that undoes this one (for a quaternion of length 1). |
| `func (rotation Quaternion) Matrix() Matrix` | Matrix returns the matrix that turns points as the quaternion does. |
| `func (rotation Quaternion) Multiply(other Quaternion) Quaternion` | Multiply returns rotation × other: the rotation that turns by other first, then by rotation. |
| `func (rotation Quaternion) Rotate(vector Vector) Vector` | Rotate returns vector turned by the quaternion; its w is kept, so points stay points. |

### Vector

Vector is a point or a direction in 3D with a fourth value, w: [x, y, z, w]. Points carry w = 1 and directions w = 0, so one matrix can both move points and turn directions. Sixteen bytes, the size GPUs expect, so it copies into OpenGL, Vulkan or Metal buffers as it is.

```go
type Vector [4]float32
```

| Function or method | |
|---|---|
| `func (vector Vector) Add(other Vector) Vector` | Add returns the sum of the two vectors, w included. |
| `func (vector Vector) Cross(other Vector) Vector` | Cross returns the cross product of the two vectors' x, y and z: a direction square to both (w is ignored and the result's is 0). |
| `func (vector Vector) Dot(other Vector) float32` | Dot returns the dot product of the two vectors' x, y and z (w is ignored). |
| `func (vector Vector) Normalize() Vector` | Normalize returns the direction of the vector with a length of 1 (w becomes 0); the zero vector stays zero. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
