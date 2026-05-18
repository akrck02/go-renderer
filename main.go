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
		Type:    models.WindowApplication,
		Width:   1000,
		Height:  800,
		Title:   "Renderer",
		Version: "v0.0.1",
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

func Draw(app *models.Application) error {
	//app.Renderer.FillTriangle(triangle, 0)
	app.Renderer.RenderRectangle(graphics.WorldSpace, graphics.Vec4{1, 1, 1}, 1, 0.5, nil, true)
	return nil
}
