package android

import (
	"math"

	space "github.com/akrck02/go-renderer/graphics"
)

// screenTurned turns a reading of Android's rotation vector into the screen's orientation in the
// scene. The sensor turns the phone's own axes (x right and y up when held upright in its natural
// way, z out of the screen) into a world with z up; the screen's axes are the phone's turned by the
// display's quarter turns, and the scene has y up and looks along -z.
func screenTurned(device space.Quaternion, quarterTurns int) space.Quaternion {
	screen := aroundAxis(0, 0, 1, -float64(quarterTurns)*math.Pi/2) // the screen turns against the phone
	upright := aroundAxis(1, 0, 0, -math.Pi/2)                      // the world's z up becomes the scene's y
	return upright.Multiply(device.Multiply(screen))
}

// aroundAxis is the rotation by radians around an axis of length 1, counter-clockwise looking
// down the axis towards the origin.
func aroundAxis(x, y, z, radians float64) space.Quaternion {
	sine, cosine := math.Sin(radians/2), math.Cos(radians/2)
	return space.Quaternion{X: float32(x * sine), Y: float32(y * sine), Z: float32(z * sine), W: float32(cosine)}
}
