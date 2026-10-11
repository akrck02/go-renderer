package camera

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
)

// Walk is a first-person controller that stays on the ground: WASD or arrows move, the mouse
// looks around (while a button is held, or always when the cursor is locked), shift runs.
type Walk struct {
	X, Z          float32
	Yaw           float64 // radians; 0 looks towards -Z
	Pitch         float64 // radians; positive looks up
	EyeHeight     float32
	Speed         float32 // world units per second
	RunFactor     float32
	LookSpeed     float64 // radians per cursor unit
	Ground        GroundFunc
	Blocked       func(x, z float32) bool // optional: true where the walker cannot go (water, walls)
	VerticalScale float32                 // 0 = 1
}

// NewWalk returns a walker at (x, z) with defaults for the given eye height and walking speed.
func NewWalk(x, z, eyeHeight, speed float32) *Walk {
	return &Walk{X: x, Z: z, EyeHeight: eyeHeight, Speed: speed, RunFactor: 4, LookSpeed: 0.004}
}

// Update applies this frame's input and places the camera at eye height.
func (walk *Walk) Update(state *input.State, seconds float64, camera *models.Camera) {
	if state != nil {
		walk.look(state)
		walk.turnWithArrows(state, seconds)
		walk.move(state, seconds)
	}
	eye := graphics.Vec4{walk.X, walk.groundUnderFeet() + walk.EyeHeight, walk.Z, 1}
	camera.Position = eye
	camera.Target = eye.Add(walk.viewDirection())
	camera.Up = graphics.Vec4{0, 1, 0, 0}
}

func isLooking(state *input.State) bool {
	return state.CursorLocked() || state.Button(input.MouseLeft) || state.Button(input.MouseRight)
}

func (walk *Walk) look(state *input.State) {
	if !isLooking(state) {
		return
	}
	walk.Yaw += state.CursorDeltaX * walk.LookSpeed
	walk.Pitch = clamp(walk.Pitch-state.CursorDeltaY*walk.LookSpeed, -1.3, 1.3)
}

func (walk *Walk) turnWithArrows(state *input.State, seconds float64) {
	turnStep := 1.6 * seconds
	if state.Down(input.KeyLeft) {
		walk.Yaw -= turnStep
	}
	if state.Down(input.KeyRight) {
		walk.Yaw += turnStep
	}
}

// movementIntent returns -1, 0 or 1 for forward and sideways movement from the held keys.
func movementIntent(state *input.State) (forward, sideways float32) {
	if state.Down(input.KeyW) || state.Down(input.KeyUp) {
		forward++
	}
	if state.Down(input.KeyS) || state.Down(input.KeyDown) {
		forward--
	}
	if state.Down(input.KeyD) {
		sideways++
	}
	if state.Down(input.KeyA) {
		sideways--
	}
	return forward, sideways
}

func (walk *Walk) move(state *input.State, seconds float64) {
	forward, sideways := movementIntent(state)
	if forward == 0 && sideways == 0 {
		return
	}
	distance := walk.Speed * float32(seconds)
	if state.Down(input.KeyShift) {
		distance *= walk.RunFactor
	}
	sineYaw, cosineYaw := float32(math.Sin(walk.Yaw)), float32(math.Cos(walk.Yaw))
	nextX := walk.X + (sineYaw*forward+cosineYaw*sideways)*distance
	nextZ := walk.Z + (-cosineYaw*forward+sineYaw*sideways)*distance
	if walk.Blocked == nil || !walk.Blocked(nextX, nextZ) {
		walk.X, walk.Z = nextX, nextZ
	}
}

// groundUnderFeet samples a small footprint so the eye never dips below a slope between samples.
func (walk *Walk) groundUnderFeet() float32 {
	if walk.Ground == nil {
		return 0
	}
	scale := scaleOrOne(walk.VerticalScale)
	radius := walk.EyeHeight * 0.8
	highest := float32(0)
	for _, offset := range [][2]float32{{0, 0}, {radius, 0}, {-radius, 0}, {0, radius}, {0, -radius}} {
		if height, found := walk.Ground(walk.X+offset[0], walk.Z+offset[1]); found && height*scale > highest {
			highest = height * scale
		}
	}
	return highest
}

func (walk *Walk) viewDirection() graphics.Vec4 {
	cosinePitch := math.Cos(walk.Pitch)
	return graphics.Vec4{
		float32(math.Sin(walk.Yaw) * cosinePitch),
		float32(math.Sin(walk.Pitch)),
		float32(-math.Cos(walk.Yaw) * cosinePitch),
		0,
	}
}
