package camera

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
)

func standingHead() graphics.Pose {
	return graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0.5, 1.6, 0.2, 1}}
}

func stateWithSticks(leftX, leftY, rightX float64) *input.State {
	state := input.NewState()
	state.Feed().SetStick(leftX, leftY)
	state.Feed().SetRightStick(rightX, 0)
	return state
}

func TestRigWalkWalksTheWayTheHeadLooks(test *testing.T) {
	var rig Rig
	var walk RigWalk
	walk.Update(stateWithSticks(0, 1, 0), 1, standingHead(), &rig)
	if !near(rig.Position[2], -DefaultRigWalkSpeedMetres) || !near(rig.Position[0], 0) {
		test.Fatalf("a full stick ahead for a second should walk 1.5 m towards -z, the rig is at %v", rig.Position)
	}
}

func TestRigWalkWalksFurtherAsAGiant(test *testing.T) {
	rig := Rig{Scale: 10}
	var walk RigWalk
	walk.Update(stateWithSticks(1, 0, 0), 1, standingHead(), &rig)
	if !near(rig.Position[0], 10*DefaultRigWalkSpeedMetres) {
		test.Fatalf("a giant ten times a person should walk ten times as far to the right, the rig is at %v", rig.Position)
	}
}

func TestRigWalkTurnsOncePerPushAroundTheHead(test *testing.T) {
	var rig Rig
	var walk RigWalk
	head := standingHead()
	before := rig.EyePosition(head)
	walk.Update(stateWithSticks(0, 0, 0), 0.1, head, &rig)
	walk.Update(stateWithSticks(0, 0, 1), 0.1, head, &rig)
	walk.Update(stateWithSticks(0, 0, 1), 0.1, head, &rig)
	if !near(rig.YawRadians, -DefaultRigTurnRadians) {
		test.Fatalf("a stick held right should turn one step clockwise, the yaw is %v", rig.YawRadians)
	}
	after := rig.EyePosition(head)
	if !near(before[0], after[0]) || !near(before[2], after[2]) {
		test.Fatalf("the head should stay where it stands while turning: %v then %v", before, after)
	}
	walk.Update(stateWithSticks(0, 0, 0), 0.1, head, &rig)
	walk.Update(stateWithSticks(0, 0, -1), 0.1, head, &rig)
	if !near(rig.YawRadians, 0) {
		test.Fatalf("back to the middle and left should turn a step back, the yaw is %v", rig.YawRadians)
	}
}

func TestRigWalkTurnsWithTheRoom(test *testing.T) {
	rig := Rig{YawRadians: math.Pi / 2} // the room turned left: its ahead is the world's -x
	var walk RigWalk
	walk.Update(stateWithSticks(0, 1, 0), 1, standingHead(), &rig)
	if !near(rig.Position[0], -DefaultRigWalkSpeedMetres) || !near(rig.Position[2], 0) {
		test.Fatalf("walking ahead in a room turned a quarter left should go towards -x, the rig is at %v", rig.Position)
	}
}

func TestRigWalkLooksDownAndStillWalks(test *testing.T) {
	head := standingHead()
	head.Orientation = graphics.Quaternion{X: float32(-math.Sqrt2 / 2), W: float32(math.Sqrt2 / 2)} // straight down
	var rig Rig
	var walk RigWalk
	walk.Update(stateWithSticks(0, 1, 0), 1, head, &rig)
	if !near(rig.Position[2], -DefaultRigWalkSpeedMetres) {
		test.Fatalf("looking straight down should walk the room's way ahead, the rig is at %v", rig.Position)
	}
}

func TestRigWalkFollowsTheGround(test *testing.T) {
	var rig Rig
	walk := RigWalk{Ground: func(x, z float32) (float32, bool) { return 3, true }}
	walk.Update(nil, 0.1, standingHead(), &rig)
	if !near(rig.Position[1], 3) {
		test.Fatalf("the floor should stand on the ground under the head, the rig is at %v", rig.Position)
	}
	walk.Ground = func(x, z float32) (float32, bool) { return 0, false }
	walk.Update(nil, 0.1, standingHead(), &rig)
	if !near(rig.Position[1], 3) {
		test.Fatalf("with no ground under the head the floor should stay where it was, the rig is at %v", rig.Position)
	}
}
