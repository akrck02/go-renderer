package camera

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
)

// isometricElevation is the angle above the ground of a true isometric view: the three axes look
// equally long (35.26°, the arctangent of 1/√2).
var isometricElevation = math.Atan(1 / math.Sqrt2)

// The map's defaults and speeds, as fractions of the scene's size or of the span on screen.
const (
	mapDistanceToScene    = 4      // the camera stands four scene sizes away, so all of it is in front
	mapClosestSpan        = 0.0005 // the narrowest span, as a fraction of the scene
	mapWidestSpan         = 3      // the widest span, in scene sizes
	mapZoomPerScrollStep  = 0.12   // the fraction of the span one scroll step zooms
	mapStickSpanPerSecond = 0.4    // the stick moves the map this many spans a second
	mapArrowSpanPerSecond = 0.6    // the arrows move it this many spans a second
	mapKeyZoomPerSecond   = 2      // + and - zoom by e to this power each second
	mapNearToDistance     = 0.01   // the near plane, as a fraction of the camera's distance
	mapFarToDistance      = 2      // the far plane, in camera distances
	quarterTurnsInATurn   = 4
)

// Map looks at the ground like a map, without perspective: from straight above, with north up,
// or isometric, from 35.26° up and at 45° to the axes, like the drawings of a town in an atlas.
// Any drag moves the map with the cursor, scroll and +/- zoom, arrows move it too, and Q and E
// turn it a quarter.
type Map struct {
	Target        graphics.Vector
	Span          float64 // world units from the bottom to the top of the screen
	Isometric     bool
	Turns         int     // quarter turns: 0 has north up in the top view
	Distance      float64 // how far the camera stands from the target, so the whole scene is in front of it
	MinimumSpan   float64
	MaximumSpan   float64
	ZoomSpeed     float64    // fraction of the span per scroll step
	TargetLimits  [4]float32 // optional: minimum x, maximum x, minimum z, maximum z (all zero = none)
	Ground        GroundFunc
	VerticalScale float32 // scale applied to ground heights (vertical exaggeration); 0 = 1
}

// NewMap returns a map of a scene of the given size (in world units), showing all of it.
func NewMap(target graphics.Vector, sceneSize float32) *Map {
	size := float64(sceneSize)
	return &Map{
		Target: target, Span: size, Distance: size * mapDistanceToScene,
		MinimumSpan: size * mapClosestSpan, MaximumSpan: size * mapWidestSpan, ZoomSpeed: mapZoomPerScrollStep,
	}
}

// Update applies this frame's input and places the camera.
func (view *Map) Update(state *input.State, seconds float64, camera *Camera) {
	if state != nil {
		view.applyMouse(state)
		view.applyKeyboard(state, seconds)
		stickX, stickY := state.Stick()
		view.move(stickX*view.Span*mapStickSpanPerSecond*seconds, stickY*view.Span*mapStickSpanPerSecond*seconds) // the stick moves the map as the arrows do
	}
	view.Span = clamp(view.Span, view.MinimumSpan, view.MaximumSpan)
	view.clampTarget()
	view.placeTargetOnGround()
	view.place(camera)
}

// heading is the direction the screen's top looks towards on the ground, in radians from north
// (−z), clockwise: north in the top view, north-east in the isometric one.
func (view *Map) heading() float64 {
	heading := float64(view.Turns) * math.Pi / 2
	if view.Isometric {
		heading += math.Pi / 4
	}
	return heading
}

func (view *Map) elevation() float64 {
	if view.Isometric {
		return isometricElevation
	}
	return math.Pi / 2
}

// groundAxes returns, on the ground, the directions of the screen's top and of its right.
func (view *Map) groundAxes() (upX, upZ, rightX, rightZ float64) {
	heading := view.heading()
	upX, upZ = math.Sin(heading), -math.Cos(heading)
	return upX, upZ, -upZ, upX
}

// place puts the camera Distance away from the target, looking at it from the plan's elevation.
func (view *Map) place(camera *Camera) {
	upX, upZ, _, _ := view.groundAxes()
	elevation := view.elevation()
	horizontal, vertical := math.Cos(elevation), math.Sin(elevation)
	camera.Position = graphics.Vector{
		view.Target[0] - float32(view.Distance*horizontal*upX),
		view.Target[1] + float32(view.Distance*vertical),
		view.Target[2] - float32(view.Distance*horizontal*upZ),
		1,
	}
	camera.Target = view.Target
	// the screen's up: towards the heading, tilted up as the view tilts (straight above, it is
	// the heading itself, which keeps it from being parallel to the line of sight)
	camera.Up = graphics.Vector{float32(upX * vertical), float32(horizontal), float32(upZ * vertical), 0}
	camera.OrthographicHeight = float32(view.Span)
	camera.Near = float32(view.Distance * mapNearToDistance)
	camera.Far = float32(view.Distance * mapFarToDistance)
}

func (view *Map) applyMouse(state *input.State) {
	if state.Button(input.MouseLeft) || state.Button(input.MouseRight) {
		view.drag(state.CursorDeltaX, state.CursorDeltaY, state.WindowHeight)
	}
	view.Span *= math.Exp(-state.ScrollDelta * view.ZoomSpeed)
}

// drag moves the map with the cursor: what is under it stays under it.
func (view *Map) drag(deltaX, deltaY float64, windowHeight int) {
	perCursorUnit := view.Span / math.Max(float64(windowHeight), 1)
	view.move(-deltaX*perCursorUnit, deltaY*perCursorUnit)
}

// move shifts the target by some world units towards the screen's right and its top. Seen at an
// angle, the ground is foreshortened up the screen, so a step up the screen covers more ground.
func (view *Map) move(right, up float64) {
	upX, upZ, rightX, rightZ := view.groundAxes()
	up /= math.Sin(view.elevation())
	view.Target[0] += float32(right*rightX + up*upX)
	view.Target[2] += float32(right*rightZ + up*upZ)
}

func (view *Map) applyKeyboard(state *input.State, seconds float64) {
	step := view.Span * mapArrowSpanPerSecond * seconds
	if state.Down(input.KeyLeft) {
		view.move(-step, 0)
	}
	if state.Down(input.KeyRight) {
		view.move(step, 0)
	}
	if state.Down(input.KeyUp) {
		view.move(0, step)
	}
	if state.Down(input.KeyDown) {
		view.move(0, -step)
	}
	if state.Down(input.KeyPlus) {
		view.Span *= math.Exp(-mapKeyZoomPerSecond * seconds)
	}
	if state.Down(input.KeyMinus) {
		view.Span *= math.Exp(mapKeyZoomPerSecond * seconds)
	}
	if state.Pressed(input.KeyQ) {
		view.Turns = (view.Turns + quarterTurnsInATurn - 1) % quarterTurnsInATurn
	}
	if state.Pressed(input.KeyE) {
		view.Turns = (view.Turns + 1) % quarterTurnsInATurn
	}
}

func (view *Map) clampTarget() {
	limits := view.TargetLimits
	if limits == [4]float32{} {
		return
	}
	view.Target[0] = float32(clamp(float64(view.Target[0]), float64(limits[0]), float64(limits[1])))
	view.Target[2] = float32(clamp(float64(view.Target[2]), float64(limits[2]), float64(limits[3])))
}

// placeTargetOnGround keeps the isometric view centred on the ground under the target.
func (view *Map) placeTargetOnGround() {
	if view.Ground == nil {
		return
	}
	if height, found := view.Ground(view.Target[0], view.Target[2]); found {
		view.Target[1] = float32(math.Max(0, float64(height*scaleOrOne(view.VerticalScale))))
	}
}
