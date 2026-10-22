package viewer

import (
	"strings"
	"testing"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/scene"
)

// A small scene in large units (a 0.08 km yard) must still walk at a person's pace: the range follows the
// starting speed, not only the scene's size.
func TestWalkingSpeedLimitsFollowTheStartingSpeed(test *testing.T) {
	limits := walkingSpeedLimits(0.08, 0.0014)
	if limits[1] < 0.0014*20*0.999 {
		test.Errorf("fastest %g, want at least 20 times the starting speed", limits[1])
	}
	if limits[0] > 0.0014/20*1.001 {
		test.Errorf("slowest %g, want at most a twentieth of the starting speed", limits[0])
	}
}

// Without a chosen speed the range is the scene's, as before.
func TestWalkingSpeedLimitsKeepTheSceneRange(test *testing.T) {
	size := float32(100)
	limits := walkingSpeedLimits(size, size*0.00075)
	if limits[0] != size*0.00002 || limits[1] != size*0.02 {
		test.Errorf("got %v, want the scene's range [%g %g]", limits, size*0.00002, size*0.02)
	}
}

// A new walking speed is told on screen, where the person walking reads it, not printed.
func TestANewWalkingSpeedIsShownAsANotice(test *testing.T) {
	session := &viewer{world: &scene.Scene{}, walk: &camera.Walk{Speed: 1, RunFactor: 4}, sceneSize: 100}
	state := input.NewState()
	state.Feed().SetKey(input.KeyRightBracket, true)
	session.adjustWalkingSpeed(state)
	notices := session.world.Overlay.Notices
	if len(notices) != 1 || !strings.Contains(notices[0].Text, "1.5") {
		test.Fatalf("notices %+v, want one with the new speed", notices)
	}
}
