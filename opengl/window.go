package opengl

import (
	"fmt"

	"github.com/akrck02/go-renderer/models"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// initGlfw initializes glfw and returns a Window to use.
func InitWindow(app *models.Application) *glfw.Window {
	if err := glfw.Init(); err != nil {
		panic(err)
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4) // OR 2
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

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
	window.MakeContextCurrent()

	return window
}
