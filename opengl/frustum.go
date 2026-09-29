package opengl

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
)

// box is an axis-aligned box.
type box struct {
	minimum, maximum graphics.Vec4
}

func emptyBox() box {
	infinity := float32(math.Inf(1))
	return box{graphics.Vec4{infinity, infinity, infinity, 1}, graphics.Vec4{-infinity, -infinity, -infinity, 1}}
}

func (bounds *box) include(point graphics.Vec4) {
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
func (bounds box) corners() [8]graphics.Vec4 {
	var corners [8]graphics.Vec4
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
func (bounds box) transformed(matrix graphics.Mat4, verticalScale float32) box {
	result := emptyBox()
	for _, corner := range bounds.corners() {
		point := matrix.Apply(corner)
		point[1] *= verticalScale
		result.include(point)
	}
	return result
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
func (bounds box) distanceTo(point graphics.Vec4) float32 {
	var squared float64
	for axis := 0; axis < 3; axis++ {
		outside := math.Max(float64(bounds.minimum[axis]-point[axis]), math.Max(0, float64(point[axis]-bounds.maximum[axis])))
		squared += outside * outside
	}
	return float32(math.Sqrt(squared))
}

// frustum is the volume a camera (or the sun, for shadows) sees, as six planes facing inwards.
type frustum struct {
	planes [6]graphics.Vec4
}

// frustumFromMatrix extracts the planes of a view-projection matrix (column-major).
func frustumFromMatrix(viewProjection graphics.Mat4) frustum {
	row := func(index int) graphics.Vec4 {
		return graphics.Vec4{viewProjection[index], viewProjection[4+index], viewProjection[8+index], viewProjection[12+index]}
	}
	sum := func(first, second graphics.Vec4, sign float32) graphics.Vec4 {
		return graphics.Vec4{first[0] + sign*second[0], first[1] + sign*second[1], first[2] + sign*second[2], first[3] + sign*second[3]}
	}
	last := row(3)
	return frustum{planes: [6]graphics.Vec4{
		sum(last, row(0), 1), sum(last, row(0), -1),
		sum(last, row(1), 1), sum(last, row(1), -1),
		sum(last, row(2), 1), sum(last, row(2), -1),
	}}
}

// containsBox reports whether any part of the box may be inside the frustum.
func (volume frustum) containsBox(bounds box) bool {
	for _, plane := range volume.planes {
		farthest := graphics.Vec4{
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

// viewpoint is what decides which parts of the scene are drawn and how detailed they are.
type viewpoint struct {
	volume      frustum
	eye         graphics.Vec4
	screenScale float32 // apparent size = world size × screenScale / distance
	levelBias   int     // extra coarseness (the shadow pass uses simpler models)
}

// newViewpoint builds the viewpoint of a camera with this vertical field of view.
func newViewpoint(viewProjection graphics.Mat4, eye graphics.Vec4, fieldOfView float64) viewpoint {
	return viewpoint{
		volume:      frustumFromMatrix(viewProjection),
		eye:         eye,
		screenScale: float32(1 / (2 * math.Tan(fieldOfView/2))),
	}
}

// apparentSize returns the fraction of the screen height taken by something of this size seen
// from this distance.
func (view viewpoint) apparentSize(size, distanceToEye float32) float32 {
	return size * view.screenScale / float32(math.Max(float64(distanceToEye), 1e-6))
}
