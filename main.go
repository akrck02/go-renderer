// Command go-renderer draws a triangle and a rectangle with the immediate-mode Renderer: the
// smallest program of the module. Choose the backend with -backend (only opengl draws today; the
// others say they are not implemented yet).
//
//	go run . -backend opengl
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/immediate"
	"github.com/akrck02/go-renderer/metal"
	"github.com/akrck02/go-renderer/opengl"
	"github.com/akrck02/go-renderer/software"
	"github.com/akrck02/go-renderer/vulkan"
	"github.com/akrck02/go-renderer/webgpu"
	"github.com/akrck02/go-renderer/window"
)

// The window opens on the main thread, which macOS needs for its events.
func init() { runtime.LockOSThread() }

func main() {
	backendName := flag.String("backend", "opengl", "drawing backend: opengl, software, vulkan, metal or webgpu")
	flag.Parse()
	backend, renderer, found := backendNamed(*backendName)
	if !found {
		fmt.Fprintf(os.Stderr, "no backend named %q\n", *backendName)
		flag.Usage()
		os.Exit(2)
	}
	app := &window.Application{
		Options: window.Options{
			Title: "Go Renderer", Width: 1000, Height: 800,
			Background: graphics.Vector{0.2, 0.3, 0.3, 1.0},
		},
		Callbacks: window.Callbacks{Draw: draw},
		Renderer:  renderer,
		Camera: camera.Lens{
			Position: graphics.Vector{0, 0, 2, 1}, // far enough back to see the triangle at the top
			Up:       graphics.Vector{0, 1, 0, 0},
		},
	}
	if err := backend.Run(app); err != nil {
		log.Fatal(err)
	}
}

// backendNamed returns the window and the renderer of the backend called name.
func backendNamed(name string) (window.Backend, immediate.Renderer, bool) {
	switch name {
	case "opengl":
		return &opengl.Window{}, &opengl.Renderer{}, true
	case "software":
		return &software.Window{}, software.Renderer{}, true
	case "vulkan":
		return &vulkan.Window{}, vulkan.Renderer{}, true
	case "metal":
		return &metal.Window{}, metal.Renderer{}, true
	case "webgpu":
		return &webgpu.Window{}, webgpu.Renderer{}, true
	}
	return nil, nil, false
}

// draw draws the frame: a triangle at the top left and a rectangle in the middle.
func draw(app *window.Application) error {
	triangle := []graphics.Vector{
		{-0.5, 0.8, 0, 1},  // top
		{-0.65, 0.5, 0, 1}, // left
		{-0.35, 0.5, 0, 1}, // right
	}
	if err := app.Renderer.RenderTriangle(graphics.WorldSpace, triangle, nil, true); err != nil {
		return err
	}
	centre := graphics.Vector{0, 0, 0, 1}
	return app.Renderer.RenderRectangle(graphics.WorldSpace, centre, 0.4, 0.3, nil, true)
}
