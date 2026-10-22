package viewer

import (
	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/window"
)

// View is what an editing program knows about the view at a step: the camera that drew the frame
// the user sees, and the window's size. The pointer's position (input.State.CursorX and CursorY)
// is in the same window pixels.
type View struct {
	Camera        camera.Lens
	Width, Height int // window pixels
	world         *scene.Scene
	lowest        float32
	highest       float32
	framing       **[4]float32
	zooming       *float64
	span          float64 // the map's span when the step began
}

// view returns the view of this step.
func (session *viewer) view(app *window.Application) View {
	width, height := app.Input.WindowWidth, app.Input.WindowHeight
	if width == 0 || height == 0 {
		width, height = app.Width, app.Height
	}
	return View{Camera: app.Camera, Width: width, Height: height, world: session.world,
		lowest: session.lowest, highest: session.highest, framing: &session.framing, zooming: &session.zooming, span: session.mapView.Span}
}

// Ray returns the ray from the camera through a point of the window, in drawn heights.
func (view View) Ray(x, y float64) (camera.Ray, bool) {
	return camera.RayThrough(view.Camera, view.Width, view.Height, x, y)
}

// PointOnPlane returns the point of the window (x, y in window pixels) on the horizontal plane at a
// true height: a flat floor. It is false when that point is not in sight.
func (view View) PointOnPlane(x, y float64, height float32) (graphics.Vector, bool) {
	ray, ok := view.Ray(x, y)
	if !ok {
		return graphics.Vector{}, false
	}
	scale := view.world.Environment.EffectiveVerticalScale()
	point, hit := ray.PointAtHeight(height * scale)
	point[1] = height
	return point, hit
}

// PointOnGround returns the point of the window (x, y in window pixels) on the scene's walkable
// ground (meshes marked Ground), with its true height. It is false when the pointer is not on it.
func (view View) PointOnGround(x, y float64) (graphics.Vector, bool) {
	ray, ok := view.Ray(x, y)
	if !ok {
		return graphics.Vector{}, false
	}
	return ray.PointOnGround(view.world.GroundHeight, view.world.Environment.EffectiveVerticalScale(), view.lowest, view.highest)
}

// FrameGround asks the cameras to show a rectangle of ground whole (x and z from minimum to
// maximum): they aim at its middle and the map's span fits it in the window. It takes effect after
// this step, as when a program opens another part of what it edits.
func (view View) FrameGround(minimumX, minimumZ, maximumX, maximumZ float32) {
	if view.framing != nil {
		*view.framing = &[4]float32{minimumX, minimumZ, maximumX, maximumZ}
	}
}

// applyFraming moves the cameras to the rectangle the program asked to show, if it asked.
func (session *viewer) applyFraming(app *window.Application) {
	framing := session.framing
	if framing == nil {
		return
	}
	session.framing = nil
	width, depth := float64(framing[2]-framing[0]), float64(framing[3]-framing[1])
	centerX, centerZ := (framing[0]+framing[2])/2, (framing[1]+framing[3])/2
	session.orbit.Target[0], session.orbit.Target[2] = centerX, centerZ
	session.mapView.Target[0], session.mapView.Target[2] = centerX, centerZ
	pixelsWide, pixelsHigh := app.Input.WindowWidth, app.Input.WindowHeight
	if pixelsWide == 0 || pixelsHigh == 0 {
		pixelsWide, pixelsHigh = app.Width, app.Height
	}
	span := depth
	if pixelsWide > 0 && pixelsHigh > 0 {
		span = max(depth, width*float64(pixelsHigh)/float64(pixelsWide))
	}
	if span > 0 {
		session.mapView.Span = min(max(span, session.mapView.MinimumSpan), session.mapView.MaximumSpan)
	}
	app.Input.RequestFrame(0)
}

// ZoomBy asks the cameras to come nearer by a factor (2 shows half as much, 0.5 twice as much),
// keeping what they aim at, as a zoom button does. It takes effect after this step.
func (view View) ZoomBy(factor float64) {
	if view.zooming != nil && factor > 0 {
		if *view.zooming == 0 {
			*view.zooming = 1
		}
		*view.zooming *= factor
	}
}

// Span is how much ground the map shows from the bottom of the window to its top, in world units:
// a program shows its zoom from it.
func (view View) Span() float64 { return view.span }

// applyZoom moves the cameras by the zoom the program asked for, if it asked.
func (session *viewer) applyZoom(app *window.Application) {
	factor := session.zooming
	if factor == 0 {
		return
	}
	session.zooming = 0
	session.mapView.Span = min(max(session.mapView.Span/factor, session.mapView.MinimumSpan), session.mapView.MaximumSpan)
	session.orbit.Distance /= factor
	app.Input.RequestFrame(0)
}
