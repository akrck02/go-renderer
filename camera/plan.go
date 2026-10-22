package camera

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
)

// isometricElevation is the angle above the ground of a true isometric view: the three axes look
// equally long (35.26°, the arctangent of 1/√2).
var isometricElevation = math.Atan(1 / math.Sqrt2)

// Plan looks at the ground like a map, without perspective: from straight above, with north up,
// or isometric, from 35.26° up and at 45° to the axes, like the drawings of a town in an atlas.
// Any drag moves the map with the cursor, scroll and +/- zoom, arrows move it too, and Q and E
// turn it a quarter.
type Plan struct {
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

// NewPlan returns a plan view of a scene of the given size, showing all of it.
func NewPlan(target graphics.Vector, sceneSize float64) *Plan {
	return &Plan{
		Target: target, Span: sceneSize, Distance: sceneSize * 4,
		MinimumSpan: sceneSize * 0.0005, MaximumSpan: sceneSize * 3, ZoomSpeed: 0.12,
	}
}

// Update applies this frame's input and places the camera.
func (plan *Plan) Update(state *input.State, seconds float64, camera *models.Camera) {
	if state != nil {
		plan.applyMouse(state)
		plan.applyKeyboard(state, seconds)
		plan.move(state.StickX*plan.Span*0.4*seconds, state.StickY*plan.Span*0.4*seconds) // the stick moves the map as the arrows do
	}
	plan.Span = clamp(plan.Span, plan.MinimumSpan, plan.MaximumSpan)
	plan.clampTarget()
	plan.placeTargetOnGround()
	plan.place(camera)
}

// heading is the direction the screen's top looks towards on the ground, in radians from north
// (−z), clockwise: north in the top view, north-east in the isometric one.
func (plan *Plan) heading() float64 {
	heading := float64(plan.Turns) * math.Pi / 2
	if plan.Isometric {
		heading += math.Pi / 4
	}
	return heading
}

func (plan *Plan) elevation() float64 {
	if plan.Isometric {
		return isometricElevation
	}
	return math.Pi / 2
}

// groundAxes returns, on the ground, the directions of the screen's top and of its right.
func (plan *Plan) groundAxes() (upX, upZ, rightX, rightZ float64) {
	heading := plan.heading()
	upX, upZ = math.Sin(heading), -math.Cos(heading)
	return upX, upZ, -upZ, upX
}

// place puts the camera Distance away from the target, looking at it from the plan's elevation.
func (plan *Plan) place(camera *models.Camera) {
	upX, upZ, _, _ := plan.groundAxes()
	elevation := plan.elevation()
	horizontal, vertical := math.Cos(elevation), math.Sin(elevation)
	camera.Position = graphics.Vector{
		plan.Target[0] - float32(plan.Distance*horizontal*upX),
		plan.Target[1] + float32(plan.Distance*vertical),
		plan.Target[2] - float32(plan.Distance*horizontal*upZ),
		1,
	}
	camera.Target = plan.Target
	// the screen's up: towards the heading, tilted up as the view tilts (straight above, it is
	// the heading itself, which keeps it from being parallel to the line of sight)
	camera.Up = graphics.Vector{float32(upX * vertical), float32(horizontal), float32(upZ * vertical), 0}
	camera.OrthographicHeight = float32(plan.Span)
	camera.Near = float32(plan.Distance * 0.01)
	camera.Far = float32(plan.Distance * 2)
}

func (plan *Plan) applyMouse(state *input.State) {
	if state.Button(input.MouseLeft) || state.Button(input.MouseRight) {
		plan.drag(state.CursorDeltaX, state.CursorDeltaY, state.WindowHeight)
	}
	plan.Span *= math.Exp(-state.ScrollDelta * plan.ZoomSpeed)
}

// drag moves the map with the cursor: what is under it stays under it.
func (plan *Plan) drag(deltaX, deltaY float64, windowHeight int) {
	perCursorUnit := plan.Span / math.Max(float64(windowHeight), 1)
	plan.move(-deltaX*perCursorUnit, deltaY*perCursorUnit)
}

// move shifts the target by some world units towards the screen's right and its top. Seen at an
// angle, the ground is foreshortened up the screen, so a step up the screen covers more ground.
func (plan *Plan) move(right, up float64) {
	upX, upZ, rightX, rightZ := plan.groundAxes()
	up /= math.Sin(plan.elevation())
	plan.Target[0] += float32(right*rightX + up*upX)
	plan.Target[2] += float32(right*rightZ + up*upZ)
}

func (plan *Plan) applyKeyboard(state *input.State, seconds float64) {
	step := plan.Span * 0.6 * seconds
	if state.Down(input.KeyLeft) {
		plan.move(-step, 0)
	}
	if state.Down(input.KeyRight) {
		plan.move(step, 0)
	}
	if state.Down(input.KeyUp) {
		plan.move(0, step)
	}
	if state.Down(input.KeyDown) {
		plan.move(0, -step)
	}
	if state.Down(input.KeyPlus) {
		plan.Span *= math.Exp(-2 * seconds)
	}
	if state.Down(input.KeyMinus) {
		plan.Span *= math.Exp(2 * seconds)
	}
	if state.Pressed(input.KeyQ) {
		plan.Turns = (plan.Turns + 3) % 4
	}
	if state.Pressed(input.KeyE) {
		plan.Turns = (plan.Turns + 1) % 4
	}
}

func (plan *Plan) clampTarget() {
	limits := plan.TargetLimits
	if limits == [4]float32{} {
		return
	}
	plan.Target[0] = float32(clamp(float64(plan.Target[0]), float64(limits[0]), float64(limits[1])))
	plan.Target[2] = float32(clamp(float64(plan.Target[2]), float64(limits[2]), float64(limits[3])))
}

// placeTargetOnGround keeps the isometric view centred on the ground under the target.
func (plan *Plan) placeTargetOnGround() {
	if plan.Ground == nil {
		return
	}
	if height, found := plan.Ground(plan.Target[0], plan.Target[2]); found {
		plan.Target[1] = float32(math.Max(0, float64(height*scaleOrOne(plan.VerticalScale))))
	}
}
