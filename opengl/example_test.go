package opengl_test

import (
	"log"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
)

// These examples open a window, so go test only compiles them. Each is the body of a program's
// main; package main locks the main thread first (func init() { runtime.LockOSThread() }).

// ExampleFlatRenderer opens a window that only draws in 2D: a card, and a lighthouse in lines.
func ExampleFlatRenderer() {
	var renderer *opengl.FlatRenderer
	var drawing draw2d.List
	var tower draw2d.Path
	tower.MoveTo(0, 10)
	tower.LineTo(1, 0)
	tower.LineTo(3, 0)
	tower.LineTo(4, 10)
	tower.Close()

	app := &models.Application{
		Type: models.WindowApplication, Title: "Drawing in 2D",
		Width: 480, Height: 240, FlatOnly: true, Input: input.NewState(),
	}
	app.Draw = func(app *models.Application) error {
		if renderer == nil { // the OpenGL context exists from the first frame
			var err error
			if renderer, err = opengl.NewFlatRenderer(); err != nil {
				return err
			}
		}
		drawing.Reset()
		card := draw2d.Rect{X: 24, Y: 24, Width: 260, Height: 64}
		drawing.Shadow(card, 12, 12, 4, draw2d.Hex("#14202f33"))
		drawing.Rectangle(card, 12, draw2d.Hex("#ffffff"), draw2d.Hex("#dde3ea"), 1)
		drawing.Icon("lightbulb", 44, 48, 16, draw2d.Hex("#e0a020"))
		drawing.Text("The lighthouse is lit", 70, 61, 14, draw2d.Medium, draw2d.Hex("#14202f"))
		drawing.Stroke(&tower, 360, 40, 14, 3, draw2d.Hex("#14202f"))

		state := app.Input
		scale := float32(state.FramebufferWidth) / float32(state.WindowWidth) // 2 on a high-density screen
		renderer.Draw(&drawing, draw2d.Hex("#eef2f6"), state.FramebufferWidth, state.FramebufferHeight, scale)
		return nil
	}
	if err := (&opengl.OpenGL{}).StartLoop(app); err != nil {
		log.Fatal(err)
	}
}

// ExampleSceneRenderer_DrawIn draws the island twice in one frame, side by side: from above on
// the left, from the shore on the right, with one overlay over both.
func ExampleSceneRenderer_DrawIn() {
	world := sample.Island()
	above, shore := mapCamera(world), shoreCamera(world)
	var renderer *opengl.SceneRenderer

	app := &models.Application{
		Type: models.WindowApplication, Title: "Two views",
		Width: 960, Height: 400, Input: input.NewState(),
		Renderer: &opengl.OpenGLRenderer{}, // the window's own drawing, unused here
	}
	app.Draw = func(app *models.Application) error {
		if renderer == nil {
			var err error
			if renderer, err = opengl.NewSceneRenderer(); err != nil {
				return err
			}
		}
		width, height := app.Input.FramebufferWidth, app.Input.FramebufferHeight
		half := width / 2
		renderer.DrawIn(world, above, opengl.Area{Width: half, Height: height}, 0)
		renderer.DrawIn(world, shore, opengl.Area{X: half, Width: width - half, Height: height}, 0)
		renderer.DrawOverlay(world, width, height) // once, after every area
		return nil
	}
	if err := (&opengl.OpenGL{}).StartLoop(app); err != nil {
		log.Fatal(err)
	}
}

// ExampleSceneRenderer_DrawOver draws the island from the shore, and the same island as a radar
// floating in the top-right corner.
func ExampleSceneRenderer_DrawOver() {
	world := sample.Island()
	view, radar := shoreCamera(world), mapCamera(world)
	var renderer *opengl.SceneRenderer

	app := &models.Application{
		Type: models.WindowApplication, Title: "A radar",
		Width: 800, Height: 450, Input: input.NewState(),
		Renderer: &opengl.OpenGLRenderer{},
	}
	app.Draw = func(app *models.Application) error {
		if renderer == nil {
			var err error
			if renderer, err = opengl.NewSceneRenderer(); err != nil {
				return err
			}
		}
		width, height := app.Input.FramebufferWidth, app.Input.FramebufferHeight
		side, margin := height/3, height/30
		corner := opengl.Area{X: width - side - margin, Y: height - side - margin, Width: side, Height: side}
		renderer.DrawIn(world, view, opengl.Area{Width: width, Height: height}, 0)
		renderer.DrawOver(world, radar, corner, 0) // no sky: the view stays round the island
		renderer.DrawOverlay(world, width, height)
		return nil
	}
	if err := (&opengl.OpenGL{}).StartLoop(app); err != nil {
		log.Fatal(err)
	}
}

// mapCamera looks straight down on the whole scene, without perspective.
func mapCamera(world *scene.Scene) models.Camera {
	centre, size := middleAndSize(world)
	return models.Camera{
		Position:           graphics.Vec4{centre[0], size, centre[2], 1},
		Target:             centre,
		Up:                 graphics.Vec4{0, 0, -1, 0}, // north at the top
		OrthographicHeight: size,
		Far:                size * 3,
	}
}

// shoreCamera looks at the middle of the scene from above its southern shore.
func shoreCamera(world *scene.Scene) models.Camera {
	centre, size := middleAndSize(world)
	return models.Camera{
		Position: graphics.Vec4{centre[0], size / 4, centre[2] + size*0.7, 1},
		Target:   centre,
		Up:       graphics.Vec4{0, 1, 0, 0},
		Far:      size * 4,
	}
}

// middleAndSize is the middle of a scene's ground and the larger of its two sides.
func middleAndSize(world *scene.Scene) (graphics.Vec4, float32) {
	minimum, maximum := world.Bounds()
	centre := graphics.Vec4{(minimum[0] + maximum[0]) / 2, 0, (minimum[2] + maximum[2]) / 2, 1}
	return centre, max(maximum[0]-minimum[0], maximum[2]-minimum[2])
}
