package main

import (
	"runtime"

	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl"
)

const (
	graphicsApi = "OpenGL"
)

var (
	triangle = []float32{
		0, 0.5, 0, // top
		-0.5, -0.5, 0, // left
		0.5, -0.5, 0, // right
	}
)

func main() {

	// This is needed to arrange that main() runs on main thread.
	// See documentation for functions that are only allowed to be called from the main thread.
	runtime.LockOSThread()

	app := &models.Application{
		Width:   700,
		Height:  500,
		Title:   "Renderer",
		Version: "v0.0.1",
	}

	// Try to render using propper graphics API
	var graphics models.GraphicsApi
	switch graphicsApi {
	case "OpenGL":
		graphics = &opengl.OpenGL{}
		app.Renderer = opengl.OpenGLRenderer{}
	default:
		panic("Unknown graphics API")
	}

	app.Draw = Draw
	graphics.Start(app)

}

func Draw(app *models.Application) {
	app.Renderer.FillTriangle(triangle, 0)
}
