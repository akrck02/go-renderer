package main

import (
	"runtime"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/metal"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl"
	"github.com/akrck02/go-renderer/software"
	"github.com/akrck02/go-renderer/vulkan"
	"github.com/akrck02/go-renderer/webgpu"
)

const (
	graphicsApi = models.OpenGL
)

func main() {

	// This is needed to arrange that main() runs on main thread.
	// See documentation for functions that are only allowed to be called from the main thread.
	runtime.LockOSThread()

	app := &models.Application{
		Type:       models.WindowApplication,
		Title:      "GO Renderer",
		Version:    "v0.0.1",
		Width:      1000,
		Height:     800,
		Background: graphics.Vec4{0.1, 0.1, 0.1, 1.0},
	}

	// Try to render using propper graphics API
	graphics := getCurrentGraphicsApi(app)
	app.Draw = Draw

	// start loop
	err := graphics.StartLoop(app)
	if nil != err {
		panic(err)
	}

}

func getCurrentGraphicsApi(app *models.Application) models.GraphicsApi {
	var graphics models.GraphicsApi
	switch graphicsApi {
	case models.Software:
		graphics = &software.Software{}
		app.Renderer = software.SoftwareRenderer{}
	case models.OpenGL:
		graphics = &opengl.OpenGL{}
		app.Renderer = opengl.OpenGLRenderer{}
	case models.Vulkan:
		graphics = &vulkan.Vulkan{}
		app.Renderer = vulkan.VulkanRenderer{}
	case models.Metal:
		graphics = &metal.Metal{}
		app.Renderer = metal.MetalRenderer{}
	case models.WebGPU:
		graphics = &webgpu.WebGPU{}
		app.Renderer = webgpu.WebGPURenderer{}
	default:
		panic("Unknown graphics API")
	}

	return graphics
}

var first = true

func Draw(app *models.Application) error {

	triangle := []graphics.Vec4{
		{-0.5, 0.8, 0, 0},  // top
		{-0.65, 0.5, 0, 0}, // left
		{-0.35, 0.5, 0, 0}, // right
	}

	app.Renderer.RenderTriangle(graphics.WorldSpace, triangle, nil, true)

	rectangle := graphics.Vec4{0, 0, 0}
	app.Renderer.RenderRectangle(graphics.WorldSpace, rectangle, 0.4, 0.3, nil, true)

	return nil
}
