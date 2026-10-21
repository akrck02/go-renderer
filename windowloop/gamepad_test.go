package windowloop

import "testing"

func TestAGamePadsStickReadsUpAsPositiveAndRestsInItsDeadZone(t *testing.T) {
	if x, y := stickFromAxes(0.5, -1); x != 0.5 || y != 1 {
		t.Fatalf("a stick pushed up and right reads %v, %v", x, y)
	}
	if x, y := stickFromAxes(0.05, -0.08); x != 0 || y != 0 {
		t.Fatalf("a stick a little off centre reads %v, %v", x, y)
	}
}
