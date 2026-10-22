package camera

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
)

// Ray is a half-line in the scene as it is drawn: from Origin along Direction (of length 1).
// Heights are drawn heights, already multiplied by the scene's vertical exaggeration.
type Ray struct {
	Origin, Direction graphics.Vector
}

// RayThrough returns the ray from the camera through a point of the window: x and y in window
// pixels from the top-left corner of a window width × height pixels (where the pointer is). It
// is false when the camera's matrices cannot be inverted (a window of no size).
func RayThrough(view Camera, width, height int, x, y float64) (Ray, bool) {
	if width <= 0 || height <= 0 {
		return Ray{}, false
	}
	up := view.Up
	if up == (graphics.Vector{}) {
		up = graphics.Vector{0, 1, 0, 0}
	}
	projection := view.ProjectionMatrix(float64(width) / float64(height))
	viewProjection := projection.Multiply(graphics.LookAt(view.Position, view.Target, up))
	inverse, invertible := viewProjection.Inverse()
	if !invertible {
		return Ray{}, false
	}
	screenX := float32(2*x/float64(width) - 1)
	screenY := float32(1 - 2*y/float64(height))
	near := unproject(inverse, screenX, screenY, -1)
	far := unproject(inverse, screenX, screenY, 1)
	direction := graphics.Vector{far[0] - near[0], far[1] - near[1], far[2] - near[2], 0}
	length := float32(math.Sqrt(float64(direction.Dot(direction))))
	if length == 0 {
		return Ray{}, false
	}
	direction = graphics.Vector{direction[0] / length, direction[1] / length, direction[2] / length, 0}
	return Ray{Origin: near, Direction: direction}, true
}

// unproject turns a point of the screen (from -1 to 1 on each axis, depth too) into the scene.
func unproject(inverse graphics.Matrix, x, y, depth float32) graphics.Vector {
	point := inverse.Apply(graphics.Vector{x, y, depth, 1})
	return graphics.Vector{point[0] / point[3], point[1] / point[3], point[2] / point[3], 1}
}

// At returns the point at a distance along the ray.
func (ray Ray) At(distance float32) graphics.Vector {
	return graphics.Vector{
		ray.Origin[0] + ray.Direction[0]*distance,
		ray.Origin[1] + ray.Direction[1]*distance,
		ray.Origin[2] + ray.Direction[2]*distance,
		1,
	}
}

// PointAtHeight returns where the ray meets the horizontal plane at a drawn height: a flat floor,
// a table top. It is false when the ray runs parallel to the plane or away from it.
func (ray Ray) PointAtHeight(height float32) (graphics.Vector, bool) {
	if math.Abs(float64(ray.Direction[1])) < 1e-9 {
		return graphics.Vector{}, false
	}
	distance := (height - ray.Origin[1]) / ray.Direction[1]
	if distance < 0 {
		return graphics.Vector{}, false
	}
	return ray.At(distance), true
}

// groundSamples is how many points the search for the ground looks at along the ray before it
// narrows down a crossing: enough not to step over a hill in a scene's relief.
const groundSamples = 512

// PointOnGround returns the first point where the ray meets a relief: ground gives the height of
// the ground at a point (in true heights, as scene.Scene.GroundHeight does, false where there is
// none), verticalScale is the exaggeration it is drawn with, and lowest and highest bound the
// ground's true heights (the scene's bounds), so the search only covers where it can be. The
// point returned has the true height. It is false when the ray misses the ground.
func (ray Ray) PointOnGround(ground func(x, z float32) (float32, bool), verticalScale, lowest, highest float32) (graphics.Vector, bool) {
	if verticalScale <= 0 {
		verticalScale = 1
	}
	start, end, crosses := ray.slab(lowest*verticalScale, highest*verticalScale)
	if !crosses {
		return graphics.Vector{}, false
	}
	above := func(distance float32) (bool, bool) {
		point := ray.At(distance)
		height, found := ground(point[0], point[2])
		return point[1] > height*verticalScale, found
	}
	previous := start
	wasAbove, _ := above(start)
	for sample := 1; sample <= groundSamples; sample++ {
		distance := start + (end-start)*float32(sample)/groundSamples
		isAbove, found := above(distance)
		if found && wasAbove && !isAbove {
			return ray.refine(above, previous, distance, verticalScale), true
		}
		previous, wasAbove = distance, isAbove || !found
	}
	return graphics.Vector{}, false
}

// slab returns the part of the ray between two drawn heights.
func (ray Ray) slab(lowest, highest float32) (start, end float32, crosses bool) {
	if math.Abs(float64(ray.Direction[1])) < 1e-9 {
		inside := ray.Origin[1] >= lowest && ray.Origin[1] <= highest
		return 0, 0, inside
	}
	first := (highest - ray.Origin[1]) / ray.Direction[1]
	second := (lowest - ray.Origin[1]) / ray.Direction[1]
	start, end = float32(math.Min(float64(first), float64(second))), float32(math.Max(float64(first), float64(second)))
	start = float32(math.Max(float64(start), 0))
	return start, end, end > start
}

// refine narrows the crossing between a distance above the ground and one below it.
func (ray Ray) refine(above func(float32) (bool, bool), over, under, verticalScale float32) graphics.Vector {
	for step := 0; step < 40; step++ {
		middle := (over + under) / 2
		if isAbove, _ := above(middle); isAbove {
			over = middle
		} else {
			under = middle
		}
	}
	point := ray.At(under)
	point[1] /= verticalScale
	return point
}
