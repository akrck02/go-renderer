//go:build android

package android

/*
#include "activity.h"
*/
import "C"

// FeelLongPress gives the short buzz Android's own applications give when a touch becomes a long
// press, if the person has touch feedback on. Safe from any goroutine.
func FeelLongPress() { C.feel_long_press() }
