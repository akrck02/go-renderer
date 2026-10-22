package camera

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
)

func closeTo(first, second graphics.Vec4, tolerance float32) bool {
	for axis := 0; axis < 3; axis++ {
		if math.Abs(float64(first[axis]-second[axis])) > float64(tolerance) {
			return false
		}
	}
	return true
}

// A camera straight above, without perspective, like the map: the middle of the window is the
// target, and a corner lies half the shown height (times the aspect) away.
func TestTheMapsRayFallsStraightDown(test *testing.T) {
	view := models.Camera{Position: graphics.Vec4{10, 100, 20, 1}, Target: graphics.Vec4{10, 0, 20, 1},
		Up: graphics.Vec4{0, 0, -1, 0}, OrthographicHeight: 40, Near: 1, Far: 500}
	ray, ok := RayThrough(view, 800, 400, 400, 200)
	if !ok {
		test.Fatal("no ray")
	}
	point, hit := ray.PointAtHeight(0)
	if !hit || !closeTo(point, graphics.Vec4{10, 0, 20, 1}, 1e-3) {
		test.Fatalf("the middle of the window is %v, want the target", point)
	}
	corner, _ := RayThrough(view, 800, 400, 0, 0)
	point, _ = corner.PointAtHeight(0)
	if !closeTo(point, graphics.Vec4{10 - 40, 0, 20 - 20, 1}, 1e-3) {
		test.Errorf("the top-left corner is %v, want 40 units west and 20 north of the target", point)
	}
}

func TestAPerspectiveRayFindsWhereTheCameraLooks(test *testing.T) {
	view := models.Camera{Position: graphics.Vec4{0, 30, 40, 1}, Target: graphics.Vec4{5, 0, -3, 1}}
	ray, ok := RayThrough(view, 1280, 800, 640, 400)
	if !ok {
		test.Fatal("no ray")
	}
	point, hit := ray.PointAtHeight(0)
	if !hit || !closeTo(point, view.Target, 1e-2) {
		test.Errorf("the middle of the window is %v, want the target %v", point, view.Target)
	}
	if _, hit := ray.PointAtHeight(100); hit {
		test.Error("a plane above the camera is behind the ray")
	}
}

// A hill in front of a flat plain: the ray hits the hill's slope before the plain behind it, at
// the hill's true height even when the relief is drawn exaggerated.
func TestTheRayStopsAtTheFirstHill(test *testing.T) {
	hill := func(x, z float32) (float32, bool) {
		return float32(math.Max(0, 15-math.Abs(float64(z)))), true
	}
	for _, scale := range []float32{1, 3} {
		view := models.Camera{Position: graphics.Vec4{0, 20 * scale, 60, 1}, Target: graphics.Vec4{0, 0, -60, 1}}
		ray, _ := RayThrough(view, 1000, 1000, 500, 500)
		point, hit := ray.PointOnGround(hill, scale, 0, 15)
		if !hit {
			test.Fatalf("scale %v: the ray missed the hill", scale)
		}
		height, _ := hill(point[0], point[2])
		if math.Abs(float64(point[1]-height)) > 1e-3 || point[2] < 0 {
			test.Errorf("scale %v: hit %v, want a point on the near slope at its true height %v", scale, point, height)
		}
	}
}

func TestARayAboveTheGroundMissesIt(test *testing.T) {
	flat := func(x, z float32) (float32, bool) { return 0, true }
	view := models.Camera{Position: graphics.Vec4{0, 5, 0, 1}, Target: graphics.Vec4{0, 50, -10, 1}}
	ray, _ := RayThrough(view, 100, 100, 50, 50)
	if _, hit := ray.PointOnGround(flat, 1, 0, 0); hit {
		test.Error("a ray looking up should miss the ground")
	}
}
