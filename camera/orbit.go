package camera

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
)

// Orbit rotates around a target: left drag rotates, right drag (or shift + left drag) pans,
// scroll zooms. Arrow keys rotate and +/- zoom from the keyboard.
type Orbit struct {
	Target          graphics.Vec4
	Distance        float64
	Yaw             float64 // radians around the vertical axis
	Pitch           float64 // radians; 0 looks from the horizon, pi/2 from straight above
	MinimumDistance float64
	MaximumDistance float64
	RotationSpeed   float64 // radians per cursor unit
	ZoomSpeed       float64 // fraction of the distance per scroll step
	Ground          GroundFunc
	GroundClearance float32    // minimum camera height above the ground
	TargetLimits    [4]float32 // optional: minimum x, maximum x, minimum z, maximum z (all zero = none)
	VerticalScale   float32    // scale applied to ground heights (vertical exaggeration); 0 = 1
}

// NewOrbit returns an orbit controller with sensible defaults for a scene of the given size.
func NewOrbit(target graphics.Vec4, sceneSize float64) *Orbit {
	return &Orbit{
		Target: target, Distance: sceneSize, Yaw: 0.4, Pitch: 0.95,
		MinimumDistance: sceneSize * 0.002, MaximumDistance: sceneSize * 3,
		RotationSpeed: 0.006, ZoomSpeed: 0.12, GroundClearance: float32(sceneSize * 0.0005),
	}
}

// Update applies this frame's input and places the camera.
func (orbit *Orbit) Update(state *input.State, seconds float64, camera *models.Camera) {
	if state != nil {
		orbit.applyMouse(state)
		orbit.applyKeyboard(state, seconds)
	}
	orbit.clampAngleAndDistance()
	orbit.clampTarget()
	orbit.placeTargetOnGround()
	camera.Position = orbit.cameraPositionAboveGround()
	camera.Target = orbit.Target
	camera.Up = graphics.Vec4{0, 1, 0, 0}
}

func isPanning(state *input.State) bool {
	return state.Button(input.MouseRight) || (state.Button(input.MouseLeft) && state.Down(input.KeyShift))
}

func (orbit *Orbit) applyMouse(state *input.State) {
	if isPanning(state) {
		orbit.pan(state.CursorDeltaX, state.CursorDeltaY, state.WindowHeight)
	} else if state.Button(input.MouseLeft) {
		orbit.Yaw -= state.CursorDeltaX * orbit.RotationSpeed
		orbit.Pitch += state.CursorDeltaY * orbit.RotationSpeed * 0.8
	}
	orbit.Distance *= math.Exp(-state.ScrollDelta * orbit.ZoomSpeed)
}

// pan moves the target on the ground plane so that the scene follows the cursor.
func (orbit *Orbit) pan(deltaX, deltaY float64, windowHeight int) {
	worldPerCursorUnit := orbit.Distance / math.Max(float64(windowHeight), 1) * 0.9
	rightX, rightZ := math.Cos(orbit.Yaw), -math.Sin(orbit.Yaw)
	forwardX, forwardZ := -math.Sin(orbit.Yaw), -math.Cos(orbit.Yaw)
	orbit.Target[0] += float32((-deltaX*rightX + deltaY*forwardX) * worldPerCursorUnit)
	orbit.Target[2] += float32((-deltaX*rightZ + deltaY*forwardZ) * worldPerCursorUnit)
}

func (orbit *Orbit) applyKeyboard(state *input.State, seconds float64) {
	angleStep := 1.6 * seconds
	if state.Down(input.KeyLeft) {
		orbit.Yaw += angleStep
	}
	if state.Down(input.KeyRight) {
		orbit.Yaw -= angleStep
	}
	if state.Down(input.KeyUp) {
		orbit.Pitch += angleStep
	}
	if state.Down(input.KeyDown) {
		orbit.Pitch -= angleStep
	}
	if state.Down(input.KeyPlus) {
		orbit.Distance *= math.Exp(-2 * seconds)
	}
	if state.Down(input.KeyMinus) {
		orbit.Distance *= math.Exp(2 * seconds)
	}
}

func (orbit *Orbit) clampAngleAndDistance() {
	orbit.Pitch = clamp(orbit.Pitch, 0.02, 1.55)
	orbit.Distance = clamp(orbit.Distance, orbit.MinimumDistance, orbit.MaximumDistance)
}

func (orbit *Orbit) clampTarget() {
	limits := orbit.TargetLimits
	if limits == [4]float32{} {
		return
	}
	orbit.Target[0] = float32(clamp(float64(orbit.Target[0]), float64(limits[0]), float64(limits[1])))
	orbit.Target[2] = float32(clamp(float64(orbit.Target[2]), float64(limits[2]), float64(limits[3])))
}

func (orbit *Orbit) placeTargetOnGround() {
	if orbit.Ground == nil {
		return
	}
	if height, found := orbit.Ground(orbit.Target[0], orbit.Target[2]); found {
		orbit.Target[1] = float32(math.Max(0, float64(height*scaleOrOne(orbit.VerticalScale))))
	}
}

func (orbit *Orbit) cameraPositionAboveGround() graphics.Vec4 {
	cosinePitch, sinePitch := math.Cos(orbit.Pitch), math.Sin(orbit.Pitch)
	position := graphics.Vec4{
		orbit.Target[0] + float32(orbit.Distance*cosinePitch*math.Sin(orbit.Yaw)),
		orbit.Target[1] + float32(orbit.Distance*sinePitch),
		orbit.Target[2] + float32(orbit.Distance*cosinePitch*math.Cos(orbit.Yaw)),
		1,
	}
	if orbit.Ground == nil {
		return position
	}
	if height, found := orbit.Ground(position[0], position[2]); found {
		lowest := height*scaleOrOne(orbit.VerticalScale) + orbit.GroundClearance
		position[1] = float32(math.Max(float64(position[1]), float64(lowest)))
	}
	return position
}
