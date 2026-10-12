package viewer

import (
	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
)

// View is what an editing program knows about the view at a step: the camera that drew the frame
// the user sees, and the window's size. The pointer's position (input.State.CursorX and CursorY)
// is in the same window pixels.
type View struct {
	Camera        models.Camera
	Width, Height int // window pixels
	world         *scene.Scene
	lowest        float32
	highest       float32
}

// view returns the view of this step.
func (session *viewer) view(app *models.Application) View {
	width, height := app.Input.WindowWidth, app.Input.WindowHeight
	if width == 0 || height == 0 {
		width, height = app.Width, app.Height
	}
	return View{Camera: app.Camera, Width: width, Height: height, world: session.world,
		lowest: session.lowest, highest: session.highest}
}

// Ray returns the ray from the camera through a point of the window, in drawn heights.
func (view View) Ray(x, y float64) (camera.Ray, bool) {
	return camera.RayThrough(view.Camera, view.Width, view.Height, x, y)
}

// PointOnPlane returns the point of the window (x, y in window pixels) on the horizontal plane at a
// true height: a flat floor. It is false when that point is not in sight.
func (view View) PointOnPlane(x, y float64, height float32) (graphics.Vec4, bool) {
	ray, ok := view.Ray(x, y)
	if !ok {
		return graphics.Vec4{}, false
	}
	scale := view.world.Environment.EffectiveVerticalScale()
	point, hit := ray.PointAtHeight(height * scale)
	point[1] = height
	return point, hit
}

// PointOnGround returns the point of the window (x, y in window pixels) on the scene's walkable
// ground (meshes marked Ground), with its true height. It is false when the pointer is not on it.
func (view View) PointOnGround(x, y float64) (graphics.Vec4, bool) {
	ray, ok := view.Ray(x, y)
	if !ok {
		return graphics.Vec4{}, false
	}
	return ray.PointOnGround(view.world.GroundHeight, view.world.Environment.EffectiveVerticalScale(), view.lowest, view.highest)
}
