//go:build js && wasm

// Command web-island shows the made-up sample island in a web browser, drawn through WebGL2: drag
// to turn around it, right-drag to move, the wheel to come closer. Build and serve it with:
//
//	GOOS=js GOARCH=wasm go build -o cmd/web-island/island.wasm ./cmd/web-island
//	cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" cmd/web-island/
//	python3 -m http.server -d cmd/web-island 8080
package main

import (
	"log"
	"math"

	"github.com/akrck02/go-renderer/browser"
	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
	"github.com/akrck02/go-renderer/window"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	canvas, err := browser.OpenCanvas("view")
	if err != nil {
		return err
	}
	functions, err := canvas.WebGL(true)
	if err != nil {
		return err
	}
	renderer, err := gles.NewSceneRenderer(functions, overlay.Fonts{})
	if err != nil {
		return err
	}
	world := sample.Island()
	orbit, size := orbitAround(world)
	app := &window.Application{Input: input.NewState(), Options: window.Options{DrawOnDemand: true},
		Camera: camera.Camera{Near: size * 0.001, Far: size * 10}}
	var shown camera.Camera
	app.Update = func(app *window.Application, step float64) error {
		orbit.Update(app.Input, step, &app.Camera)
		if app.Camera != shown {
			// the camera is still moving (it eases to a stop): draw the next frame too
			app.Input.RequestFrame(0)
			shown = app.Camera
		}
		return nil
	}
	draw := func() error {
		state := app.Input
		renderer.OverlayScale = float32(state.FramebufferWidth) / float32(max(state.WindowWidth, 1))
		renderer.Draw(world, app.Camera, state.FramebufferWidth, state.FramebufferHeight)
		return nil
	}
	return browser.Run(canvas, app, draw)
}

// orbitAround places an orbit camera over the middle of the scene, as the desktop viewer does.
func orbitAround(world *scene.Scene) (*camera.Orbit, float32) {
	minimum, maximum := world.Bounds()
	size := float32(math.Max(float64(maximum[0]-minimum[0]), float64(maximum[2]-minimum[2])))
	if size <= 0 || math.IsInf(float64(size), 0) {
		size = 10
	}
	center := graphics.Vector{(minimum[0] + maximum[0]) / 2, 0, (minimum[2] + maximum[2]) / 2, 1}
	orbit := camera.NewOrbit(center, size*1.1)
	orbit.MaximumDistance = float64(size) * 3
	orbit.Ground = world.GroundHeight
	return orbit, size
}
