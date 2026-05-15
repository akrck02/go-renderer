package main

import (
	"runtime"

	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl"
)

const (
	graphicsApi = "OpenGL"
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
	default:
		panic("Unknown graphics API")
	}

	graphics.Start(app)

}
