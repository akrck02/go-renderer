package camera

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
)

// Defaults of a RigWalk, used where its fields are zero.
const (
	DefaultRigWalkSpeedMetres = 1.5         // a calm walk
	DefaultRigTurnRadians     = math.Pi / 6 // twelve steps make a whole turn
)

// The right stick turns once each time it is pushed past turnPush, and again only after it came
// back inside turnRelease: a stick held to one side does not keep spinning the player.
const (
	turnPush    = 0.7
	turnRelease = 0.3
)

// RigWalk moves a headset's rig with a pad's sticks: the left stick walks the way the head looks
// (flat on the ground), the right stick turns in steps around the head. The head itself is never
// moved by the program: a view that turns smoothly without the head turning makes many people
// sick, a step does not.
type RigWalk struct {
	// SpeedMetres is how far a full stick walks in a second, in the room's metres (0 =
	// DefaultRigWalkSpeedMetres); a giant (Rig.Scale above 1) covers as many of its own metres.
	SpeedMetres float32
	// TurnRadians is one step of a turn (0 = DefaultRigTurnRadians).
	TurnRadians float32
	// Ground, when set, keeps the room's floor on the ground under the head.
	Ground GroundFunc

	turnReady bool // the right stick came back to the middle since the last step
}

// Update walks and turns the rig from the input's sticks, seconds after the last update. head is
// the head's pose in the headset's space, as the headset reports it.
func (walk *RigWalk) Update(state *input.State, seconds float64, head graphics.Pose, rig *Rig) {
	if state != nil {
		stickX, stickY := state.Stick()
		walk.move(float32(stickX), float32(stickY), seconds, head, rig)
		turnX, _ := state.RightStick()
		walk.turn(float32(turnX), head, rig)
	}
	walk.followGround(head, rig)
}

// move walks along the head's forward and right, flattened onto the ground.
func (walk *RigWalk) move(sideways, forward float32, seconds float64, head graphics.Pose, rig *Rig) {
	if sideways == 0 && forward == 0 {
		return
	}
	ahead := flatForward(head, rig)
	right := ahead.Cross(graphics.Vector{0, 1, 0, 0}).Normalize()
	distance := walk.speed() * scaleOrOne(rig.Scale) * float32(seconds)
	for axis := 0; axis < 3; axis += 2 {
		rig.Position[axis] += (ahead[axis]*forward + right[axis]*sideways) * distance
	}
}

// turn takes a step of a turn when the stick is pushed far enough, around the head: the room
// turns under the player, who stays where they stand.
func (walk *RigWalk) turn(sideways float32, head graphics.Pose, rig *Rig) {
	if math.Abs(float64(sideways)) < turnRelease {
		walk.turnReady = true
		return
	}
	if !walk.turnReady || math.Abs(float64(sideways)) < turnPush {
		return
	}
	walk.turnReady = false
	before := rig.EyePosition(head)
	step := walk.turnStep()
	if sideways > 0 {
		step = -step // counter-clockwise is positive: pushing right turns clockwise, seen from above
	}
	rig.YawRadians += step
	after := rig.EyePosition(head)
	rig.Position[0] += before[0] - after[0]
	rig.Position[2] += before[2] - after[2]
}

// followGround puts the room's floor on the ground under the head, where there is ground.
func (walk *RigWalk) followGround(head graphics.Pose, rig *Rig) {
	if walk.Ground == nil {
		return
	}
	position := rig.EyePosition(head)
	if height, found := walk.Ground(position[0], position[2]); found {
		rig.Position[1] = height
	}
}

func (walk *RigWalk) speed() float32 {
	if walk.SpeedMetres == 0 {
		return DefaultRigWalkSpeedMetres
	}
	return walk.SpeedMetres
}

func (walk *RigWalk) turnStep() float32 {
	if walk.TurnRadians == 0 {
		return DefaultRigTurnRadians
	}
	return walk.TurnRadians
}

// flatForward is where the head looks in the world, flat on the ground and of unit length. Looking
// straight up or down leaves no direction on the ground; the room's own forward stands in for it.
func flatForward(head graphics.Pose, rig *Rig) graphics.Vector {
	turn := graphics.QuaternionAroundY(float64(rig.YawRadians))
	looking := turn.Multiply(head.Orientation).Rotate(graphics.Vector{0, 0, -1, 0})
	looking[1] = 0
	if math.Hypot(float64(looking[0]), float64(looking[2])) < 1e-3 {
		return turn.Rotate(graphics.Vector{0, 0, -1, 0})
	}
	return looking.Normalize()
}
