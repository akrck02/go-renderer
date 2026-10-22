package sceneplan

import (
	"math"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
)

// box is an axis-aligned box.
type box struct {
	minimum, maximum graphics.Vector
}

func emptyBox() box {
	infinity := float32(math.Inf(1))
	return box{graphics.Vector{infinity, infinity, infinity, 1}, graphics.Vector{-infinity, -infinity, -infinity, 1}}
}

func (bounds *box) include(point graphics.Vector) {
	for axis := 0; axis < 3; axis++ {
		bounds.minimum[axis] = float32(math.Min(float64(bounds.minimum[axis]), float64(point[axis])))
		bounds.maximum[axis] = float32(math.Max(float64(bounds.maximum[axis]), float64(point[axis])))
	}
}

func (bounds *box) merge(other box) {
	bounds.include(other.minimum)
	bounds.include(other.maximum)
}

func (bounds box) isEmpty() bool { return bounds.minimum[0] > bounds.maximum[0] }

// corners returns the eight corners of the box.
func (bounds box) corners() [8]graphics.Vector {
	var corners [8]graphics.Vector
	for corner := range corners {
		for axis := 0; axis < 3; axis++ {
			if corner&(1<<axis) == 0 {
				corners[corner][axis] = bounds.minimum[axis]
			} else {
				corners[corner][axis] = bounds.maximum[axis]
			}
		}
		corners[corner][3] = 1
	}
	return corners
}

// transformed returns the box around this box moved by a matrix, with heights multiplied by the
// vertical scale as the vertex shader does.
func (bounds box) transformed(matrix graphics.Matrix, verticalScale float32) box {
	result := emptyBox()
	for _, corner := range bounds.corners() {
		point := movedPoint(matrix, corner)
		point[1] *= verticalScale
		result.include(point)
	}
	return result
}

// movedPoint is matrix.Apply for a point (w = 1) whose w is not needed afterwards: the fourth
// row is left out, a quarter of the work, for the eight corners of every box of every frame.
func movedPoint(matrix graphics.Matrix, point graphics.Vector) graphics.Vector {
	return graphics.Vector{
		matrix[0]*point[0] + matrix[4]*point[1] + matrix[8]*point[2] + matrix[12],
		matrix[1]*point[0] + matrix[5]*point[1] + matrix[9]*point[2] + matrix[13],
		matrix[2]*point[0] + matrix[6]*point[1] + matrix[10]*point[2] + matrix[14],
		1,
	}
}

// grown returns the box enlarged by a margin on every side.
func (bounds box) grown(margin float32) box {
	for axis := 0; axis < 3; axis++ {
		bounds.minimum[axis] -= margin
		bounds.maximum[axis] += margin
	}
	return bounds
}

// diagonal returns the length of the box diagonal.
func (bounds box) diagonal() float32 {
	return distance(bounds.minimum, bounds.maximum)
}

// distanceTo returns the distance from a point to the box (0 inside it).
func (bounds box) distanceTo(point graphics.Vector) float32 {
	var squared float64
	for axis := 0; axis < 3; axis++ {
		outside := math.Max(float64(bounds.minimum[axis]-point[axis]), math.Max(0, float64(point[axis]-bounds.maximum[axis])))
		squared += outside * outside
	}
	return float32(math.Sqrt(squared))
}

// frustum is the volume a camera (or the sun, for shadows) sees, as six planes facing inwards.
type frustum struct {
	planes [6]graphics.Vector
}

// frustumFromMatrix extracts the planes of a view-projection matrix (column-major).
func frustumFromMatrix(viewProjection graphics.Matrix) frustum {
	row := func(index int) graphics.Vector {
		return graphics.Vector{viewProjection[index], viewProjection[4+index], viewProjection[8+index], viewProjection[12+index]}
	}
	sum := func(first, second graphics.Vector, sign float32) graphics.Vector {
		return graphics.Vector{first[0] + sign*second[0], first[1] + sign*second[1], first[2] + sign*second[2], first[3] + sign*second[3]}
	}
	last := row(3)
	return frustum{planes: [6]graphics.Vector{
		sum(last, row(0), 1), sum(last, row(0), -1),
		sum(last, row(1), 1), sum(last, row(1), -1),
		sum(last, row(2), 1), sum(last, row(2), -1),
	}}
}

// containsBox reports whether any part of the box may be inside the frustum.
func (volume frustum) containsBox(bounds box) bool {
	for _, plane := range volume.planes {
		farthest := graphics.Vector{
			pick(plane[0] >= 0, bounds.maximum[0], bounds.minimum[0]),
			pick(plane[1] >= 0, bounds.maximum[1], bounds.minimum[1]),
			pick(plane[2] >= 0, bounds.maximum[2], bounds.minimum[2]),
		}
		if plane[0]*farthest[0]+plane[1]*farthest[1]+plane[2]*farthest[2]+plane[3] < 0 {
			return false
		}
	}
	return true
}

func pick(condition bool, whenTrue, whenFalse float32) float32 {
	if condition {
		return whenTrue
	}
	return whenFalse
}

// Viewpoint is what decides which parts of the scene are drawn and how detailed they are.
type Viewpoint struct {
	volume      frustum
	eye         graphics.Vector
	screenScale float32 // apparent size = world size × screenScale / distance (÷ 1 without perspective)
	levelBias   int     // extra coarseness (the shadow pass uses simpler models)
	// orthographic views have no perspective: things look the same size however far they are
	orthographic bool
}

// newViewpoint builds the viewpoint of a camera.
func newViewpoint(viewProjection graphics.Matrix, camera camera.Lens) Viewpoint {
	view := Viewpoint{volume: frustumFromMatrix(viewProjection), eye: camera.Position}
	if camera.IsOrthographic() {
		view.orthographic, view.screenScale = true, 1/camera.OrthographicHeight
		return view
	}
	fieldOfView, _, _ := camera.Projection()
	view.screenScale = float32(1 / (2 * math.Tan(float64(fieldOfView)/2)))
	return view
}

// apparentSize returns the fraction of the screen height taken by something of this size seen
// from this distance.
func (view Viewpoint) apparentSize(size, distanceToEye float32) float32 {
	if view.orthographic {
		return size * view.screenScale
	}
	return size * view.screenScale / float32(math.Max(float64(distanceToEye), 1e-6))
}
