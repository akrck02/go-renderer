package windowloop

import "testing"

func TestWakingBeforeAWindowOpensDoesNothing(test *testing.T) {
	Wake() // before StartGLFW, GLFW's own wake panics: a message may arrive before the window opens
}
