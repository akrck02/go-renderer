package opengl

import (
	"fmt"

	"github.com/akrck02/go-renderer/models"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// initGlfw initializes glfw and returns a Window to use.
func InitWindow(app *models.Application) *glfw.Window {
	if err := glfw.Init(); err != nil {
		panic(err)
	}

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4) // OR 2
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.CocoaRetinaFramebuffer, glfw.True)
	if app.Type == models.HeadlessApplication {
		glfw.WindowHint(glfw.Visible, glfw.False)
	}
	if app.FlatOnly {
		glfw.WindowHint(glfw.DepthBits, 0)
		glfw.WindowHint(glfw.StencilBits, 0)
		// a Mac with two GPUs would otherwise keep the discrete one running while the window is
		// open; flat drawing is light work for the integrated one
		glfw.WindowHint(glfw.CocoaGraphicsSwitching, glfw.True)
	}

	window, err := glfw.CreateWindow(
		app.Width,
		app.Height,
		fmt.Sprintf("%s %s", app.Title, app.Version),
		nil,
		nil,
	)

	if err != nil {
		panic(err)
	}

	if app.Input == nil {
		window.SetAspectRatio(app.Width, app.Height)
	}

	if app.MinWidth > 0 || app.MinHeight > 0 {
		window.SetSizeLimits(max(app.MinWidth, glfw.DontCare), max(app.MinHeight, glfw.DontCare), glfw.DontCare, glfw.DontCare)
	}

	window.MakeContextCurrent()
	// Ask for vsync either way: GLFW sets no swap interval when it makes the context, and the
	// driver's default is not reliable (on macOS it was seen presenting at twice the display rate).
	if app.DisableVsync {
		glfw.SwapInterval(0)
	} else {
		glfw.SwapInterval(1)
	}
	window.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})

	return window
}
