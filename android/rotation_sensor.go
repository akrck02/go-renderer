//go:build android

package android

/*
#include "activity.h"
*/
import "C"

import (
	"sync/atomic"

	space "github.com/akrck02/go-renderer/graphics"
)

// displayQuarterTurns is how far the screen is turned from the phone's natural way up, as the
// window last read it: Rotation needs it to know which way the screen's own axes point.
var displayQuarterTurns atomic.Int32

// StartRotation starts reading the phone's rotation sensor, for Rotation: it says false when the
// phone has none. The sensor draws power, so a program starts it while it needs it and stops it
// with StopRotation. Starting it again while it runs does nothing. Safe from any goroutine.
func StartRotation() bool { return C.rotation_start() != 0 }

// StopRotation stops the rotation sensor. Safe from any goroutine.
func StopRotation() { C.rotation_stop() }

// Rotation is how the screen is turned in the room, as a camera's orientation: a camera turned
// by it looks out of the back of the phone, with the screen's top as its up and the room's up as
// the scene's y. Holding the phone upright, the screen turned to the person, looks ahead; the
// heading it starts with is arbitrary. It follows the screen as the phone turns it, upright or on
// its side. It says false before StartRotation and until the sensor's first reading.
func Rotation() (space.Quaternion, bool) {
	var values [4]C.float
	if C.rotation_read(&values[0]) == 0 {
		return space.IdentityQuaternion(), false
	}
	device := space.Quaternion{X: float32(values[0]), Y: float32(values[1]), Z: float32(values[2]), W: float32(values[3])}
	return screenTurned(device, int(displayQuarterTurns.Load())), true
}
