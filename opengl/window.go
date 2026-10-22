package opengl

import (
	"github.com/akrck02/go-renderer/failure"
	"github.com/akrck02/go-renderer/window"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// The OpenGL version the window asks for: 4.1 core, the last macOS has.
const (
	openGLMajorVersion = 4
	openGLMinorVersion = 1
)

// openWindow starts GLFW and opens the application's window with an OpenGL 4.1 context, current
// on this thread.
func openWindow(app *window.Application) (*glfw.Window, error) {
	if err := glfw.Init(); err != nil {
		return nil, failure.Wrap(WindowNotOpened, "starting GLFW", err)
	}
	setWindowHints(app)
	glfwWindow, err := glfw.CreateWindow(app.Width, app.Height, windowTitle(app), nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil, failure.Wrap(WindowNotOpened, "creating the window", err)
	}
	if app.Input == nil {
		glfwWindow.SetAspectRatio(app.Width, app.Height)
	}
	if app.MinWidth > 0 || app.MinHeight > 0 {
		glfwWindow.SetSizeLimits(max(app.MinWidth, glfw.DontCare), max(app.MinHeight, glfw.DontCare), glfw.DontCare, glfw.DontCare)
	}
	glfwWindow.MakeContextCurrent()
	// Ask for vsync either way: GLFW sets no swap interval when it makes the context, and the
	// driver's default is not reliable (on macOS it was seen presenting at twice the display rate).
	if app.DisableVsync {
		glfw.SwapInterval(0)
	} else {
		glfw.SwapInterval(1)
	}
	glfwWindow.SetFramebufferSizeCallback(func(_ *glfw.Window, width, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})
	return glfwWindow, nil
}

// setWindowHints tells GLFW what the next window should be.
func setWindowHints(app *window.Application) {
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, openGLMajorVersion)
	glfw.WindowHint(glfw.ContextVersionMinor, openGLMinorVersion)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.CocoaRetinaFramebuffer, glfw.True)
	if app.Kind == window.Headless {
		glfw.WindowHint(glfw.Visible, glfw.False)
	}
	if app.FlatOnly {
		glfw.WindowHint(glfw.DepthBits, 0)
		glfw.WindowHint(glfw.StencilBits, 0)
		// a Mac with two GPUs would otherwise keep the discrete one running while the window is
		// open; flat drawing is light work for the integrated one
		glfw.WindowHint(glfw.CocoaGraphicsSwitching, glfw.True)
	}
}

// windowTitle is the application's title and version, as the title bar shows them.
func windowTitle(app *window.Application) string {
	if app.Version == "" {
		return app.Title
	}
	return app.Title + " " + app.Version
}
