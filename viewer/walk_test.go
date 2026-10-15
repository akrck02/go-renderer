package viewer

import "testing"

// A small scene in large units (a 0.08 km yard) must still walk at a person's pace: the range follows the
// starting speed, not only the scene's size.
func TestWalkingSpeedLimitsFollowTheStartingSpeed(t *testing.T) {
	limits := walkingSpeedLimits(0.08, 0.0014)
	if limits[1] < 0.0014*20*0.999 {
		t.Errorf("fastest %g, want at least 20 times the starting speed", limits[1])
	}
	if limits[0] > 0.0014/20*1.001 {
		t.Errorf("slowest %g, want at most a twentieth of the starting speed", limits[0])
	}
}

// Without a chosen speed the range is the scene's, as before.
func TestWalkingSpeedLimitsKeepTheSceneRange(t *testing.T) {
	size := float32(100)
	limits := walkingSpeedLimits(size, size*0.00075)
	if limits[0] != size*0.00002 || limits[1] != size*0.02 {
		t.Errorf("got %v, want the scene's range [%g %g]", limits, size*0.00002, size*0.02)
	}
}
