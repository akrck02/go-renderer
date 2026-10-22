package android

import "testing"

// Each feeling has Android's constant for it, and the older phones the nearest they have.
func TestFeelingsHaveTheirConstants(test *testing.T) {
	cases := []struct {
		feeling Feeling
		level   int
		want    int
	}{
		{FeelingLongPress, 34, longPressConstant},
		{FeelingTick, 21, clockTickConstant},
		{FeelingConfirm, 30, confirmConstant},
		{FeelingConfirm, 26, contextClickConstant},
		{FeelingConfirm, 21, clockTickConstant},
		{FeelingReject, 30, rejectConstant},
		{FeelingReject, 29, longPressConstant},
	}
	for _, test := range cases {
		if got := constantFor(test.feeling, test.level); got != test.want {
			test.Errorf("feeling %d on API %d: %d, want %d", test.feeling, test.level, got, test.want)
		}
	}
}
