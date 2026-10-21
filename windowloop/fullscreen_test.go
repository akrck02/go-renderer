package windowloop

import "testing"

func TestTheScreenThatHoldsMostOfAWindowIsTheOneItFills(test *testing.T) {
	// a window from 1800 to 2400 across: 120 on a screen 0 to 1920 wide, 480 on the next
	if first, second := overlap(1800, 2400, 0, 1920), overlap(1800, 2400, 1920, 3840); first != 120 || second != 480 {
		test.Fatalf("the window shares %d with the first screen and %d with the second", first, second)
	}
	if apart := overlap(0, 100, 200, 300); apart != 0 {
		test.Fatalf("spans apart share %d", apart)
	}
}
