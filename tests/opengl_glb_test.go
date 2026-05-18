package test

import (
	"math"
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
		Rotation: graphics.Vec4{1, 1, 1, 1},
	}

	//startTime := time.Now()

	app.Renderer = opengl.OpenGLRenderer{}
	app.Draw = func(app *models.Application) error {
		// 1. Rotation for animation + User input
		rotX := graphics.RotateX(float64(app.Rotation[0]))
		rotY := graphics.RotateY(float64(app.Rotation[1]))
		modelMatrix := rotX.Multiply(rotY)

		// 2. View (Camera) - Now uses the interactive application camera
		view := graphics.LookAt(app.Camera.Position, app.Camera.Target, app.Camera.Up)

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
