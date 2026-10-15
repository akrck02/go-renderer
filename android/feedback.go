//go:build android

package android

/*
#include "activity.h"
*/
import "C"

import "sync"

var (
	levelOnce sync.Once
	level     int
)

// Feel gives a buzz of a kind, if the person has touch feedback on. Safe from any goroutine.
func Feel(feeling Feeling) {
	levelOnce.Do(func() { level = int(C.android_level()) })
	C.feel(C.int(constantFor(feeling, level)))
}

// FeelLongPress gives the short buzz Android's own applications give when a touch becomes a long
// press, if the person has touch feedback on. Safe from any goroutine.
func FeelLongPress() { Feel(FeelingLongPress) }
