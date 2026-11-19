package frameloop

import (
	"testing"
	"time"
)

func TestACaptureWaitsWhileTheProgramMovesButNotForALaterFrame(test *testing.T) {
	cases := []struct {
		name     string
		requests []time.Duration // the delays asked for, one per frame; then none
		drawn    int
	}{
		{"nothing moves", nil, 0},
		{"a motion of three frames", []time.Duration{16 * time.Millisecond, 16 * time.Millisecond, 16 * time.Millisecond}, 3},
		{"a caret's blink is not motion", []time.Duration{500 * time.Millisecond}, 0},
	}
	for _, each := range cases {
		requests, drawn := each.requests, 0
		next := func() (time.Duration, bool) {
			if len(requests) == 0 {
				return 0, false
			}
			delay := requests[0]
			requests = requests[1:]
			return delay, true
		}
		err := Settle(next, func() error { drawn++; return nil }, func(time.Duration) {})
		if err != nil || drawn != each.drawn {
			test.Errorf("%s: drew %d frames (err %v), not %d", each.name, drawn, err, each.drawn)
		}
	}
}

func TestACaptureStopsWaitingForAMotionThatNeverEnds(test *testing.T) {
	drawn := 0
	forever := func() (time.Duration, bool) { return 16 * time.Millisecond, true }
	if err := Settle(forever, func() error { drawn++; return nil }, func(time.Duration) {}); err != nil {
		test.Fatal(err)
	}
	if limit := int(settleLimit / (16 * time.Millisecond)); drawn != limit {
		test.Errorf("drew %d frames, not the %d that fit in the limit", drawn, limit)
	}
}
