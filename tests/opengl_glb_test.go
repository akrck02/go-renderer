package test

import (
	"math"
	"runtime"
	"testing"
	"time"

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
	}

	startTime := time.Now()

	app.Renderer = opengl.OpenGLRenderer{}
	app.Draw = func(app *models.Application) error {
		elapsed := time.Since(startTime).Seconds()

		// 1. Rotation for animation
		rotY := graphics.RotateY(elapsed)
		modelMatrix := rotY

		// 2. View (Camera) - move back to see the model
		// You might need to adjust this depending on the model's scale
		view := graphics.Translate(graphics.Vec4{0, 0, -5, 0})

		// 3. Projection
		aspect := float64(app.Width) / float64(app.Height)
		projection := graphics.PerspectiveOpenGL(math.Pi/4, aspect, 0.1, 1000.0)

		// Combined transformation matrix
		mvp := projection.Multiply(view).Multiply(modelMatrix)

		// Create a copy of the model with transformed vertices for our current renderer
		// Note: In a more optimized version, the MVP would be passed to the GPU as a uniform
		transformedModel := &models.Model{
			Meshes: make([]models.Mesh, len(model.Meshes)),
		}

		for i, mesh := range model.Meshes {
			transformedMesh := models.Mesh{
				Indices:  mesh.Indices,
				Vertices: make([]graphics.Vec4, len(mesh.Vertices)),
			}

			for j, v := range mesh.Vertices {
				// Apply MVP in Go
				clip := mvp.Apply(v)

				// Perspective divide
				if clip[3] != 0 {
					transformedMesh.Vertices[j] = graphics.Vec4{
						clip[0] / clip[3],
						clip[1] / clip[3],
						clip[2] / clip[3],
						1.0,
					}
				} else {
					transformedMesh.Vertices[j] = clip
				}
			}
			transformedModel.Meshes[i] = transformedMesh
		}

		return app.Renderer.Render3dModel(graphics.WorldSpace, transformedModel, nil)
	}

	g := &opengl.OpenGL{}
	err = g.StartLoop(app)
	if err != nil {
		t.Fatalf("StartLoop failed: %v", err)
	}
}
