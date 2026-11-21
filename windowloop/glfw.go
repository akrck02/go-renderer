package windowloop

import (
	"sync"

	"github.com/go-gl/glfw/v3.4/glfw"
)

// Wake may be called from any goroutine at any time, but GLFW can only be woken while it runs:
// waking it before it starts or after it stops is an error, which the Go binding turns into a
// panic. StartGLFW and StopGLFW mark when it runs, under a lock that Wake takes too, so a wake
// never meets GLFW half stopped.

// glfwState is whether GLFW runs, and the lock that guards it.
var glfwState struct {
	sync.RWMutex
	running bool
}

// StartGLFW starts GLFW, which Wake can wake from then on. Call it on the main thread, before
// opening a window, and StopGLFW once the window has closed.
func StartGLFW() error {
	if err := glfw.Init(); err != nil {
		return err
	}
	glfwState.Lock()
	defer glfwState.Unlock()
	glfwState.running = true
	return nil
}

// StopGLFW stops GLFW, closing what windows are left; Wake does nothing from then on.
func StopGLFW() {
	glfwState.Lock()
	defer glfwState.Unlock()
	glfwState.running = false
	glfw.Terminate()
}

// Wake makes a window that draws on demand draw a frame now: for a program whose data changes
// on another goroutine (a download finishing, a message arriving). Safe from any goroutine, and
// at any time: before the window opens or after it closes, it does nothing.
func Wake() {
	glfwState.RLock()
	defer glfwState.RUnlock()
	if glfwState.running {
		glfw.PostEmptyEvent()
	}
}
