package test

import (
	"runtime"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/loaders"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl"
)

func TestRenderGLB(t *testing.T) {
	runtime.LockOSThread()

	// Load the model
	model, err := loaders.LoadGLB("../3dmodels/bogdan.glb")
	if err != nil {
		t.Fatalf("Failed to load GLB model: %v", err)
	}

	app := &models.Application{
		Type:       models.WindowApplication,
		Width:      1024,
		Height:     768,
		Title:      "GLB Model Test - bogdan.glb",
		Version:    "v0.0.1",
		Background: graphics.Vec4{0.1, 0.1, 0.1, 1.0},
		Camera: models.Camera{
			Position: graphics.Vec4{0, 0, 5, 1},
			Target:   graphics.Vec4{0, 0, 0, 1},
			Up:       graphics.Vec4{0, 1, 0, 0},
		},
		ModelMatrix: graphics.Identity(),
	}

	app.Renderer = &opengl.OpenGLRenderer{}
	app.Draw = func(app *models.Application) error {
		// The renderer now handles all matrices automatically
		return app.Renderer.Render3dModel(graphics.WorldSpace, model, nil)
	}

	g := &opengl.OpenGL{}
	err = g.StartLoop(app)
	if err != nil {
		t.Fatalf("StartLoop failed: %v", err)
	}
}
