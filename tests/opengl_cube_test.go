package test

import (
	"math"
	"runtime"
	"testing"
	"time"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl"
)

func TestRenderCube(t *testing.T) {
	runtime.LockOSThread()

	app := &models.Application{
		Type:       models.WindowApplication,
		Width:      800,
		Height:     600,
		Title:      "3D Cube Test",
		Version:    "v0.0.1",
		Background: graphics.Vec4{0.1, 0.1, 0.1, 1.0},
	}

	startTime := time.Now()

	app.Renderer = opengl.OpenGLRenderer{}
	app.Draw = func(app *models.Application) error {
		elapsed := time.Since(startTime).Seconds()

		// Matrices for 3D transformation
		// 1. Rotation
		rotX := graphics.RotateX(elapsed)
		rotY := graphics.RotateY(elapsed * 0.7)
		model := rotX.Multiply(rotY)

		// 2. View (Camera)
		// Move back so we can see the cube
		view := graphics.Translate(graphics.Vec4{0, 0, -2, 0})

		// 3. Projection
		aspect := float64(app.Width) / float64(app.Height)
		projection := graphics.PerspectiveOpenGL(math.Pi/4, aspect, 0.1, 100.0)

		// Combined transformation matrix
		mvp := projection.Multiply(view).Multiply(model)

		// Define cube faces with different "depth" to show 3D
		// We'll render each face separately to potentially give them different colors if we had that option,
		// but for now we'll just transform them all.

		cubeFaces := [][]graphics.Vec4{
			// Front face
			{{-0.3, -0.3, 0.3, 1}, {0.3, -0.3, 0.3, 1}, {0.3, 0.3, 0.3, 1}, {0.3, 0.3, 0.3, 1}, {-0.3, 0.3, 0.3, 1}, {-0.3, -0.3, 0.3, 1}},
			// Back face
			{{-0.3, -0.3, -0.3, 1}, {-0.3, 0.3, -0.3, 1}, {0.3, 0.3, -0.3, 1}, {0.3, 0.3, -0.3, 1}, {0.3, -0.3, -0.3, 1}, {-0.3, -0.3, -0.3, 1}},
			// Top face
			{{-0.3, 0.3, -0.3, 1}, {-0.3, 0.3, 0.3, 1}, {0.3, 0.3, 0.3, 1}, {0.3, 0.3, 0.3, 1}, {0.3, 0.3, -0.3, 1}, {-0.3, 0.3, -0.3, 1}},
			// Bottom face
			{{-0.3, -0.3, -0.3, 1}, {0.3, -0.3, -0.3, 1}, {0.3, -0.3, 0.3, 1}, {0.3, -0.3, 0.3, 1}, {-0.3, -0.3, 0.3, 1}, {-0.3, -0.3, -0.3, 1}},
			// Left face
			{{-0.3, -0.3, -0.3, 1}, {-0.3, -0.3, 0.3, 1}, {-0.3, 0.3, 0.3, 1}, {-0.3, 0.3, 0.3, 1}, {-0.3, 0.3, -0.3, 1}, {-0.3, -0.3, -0.3, 1}},
			// Right face
			{{0.3, -0.3, -0.3, 1}, {0.3, 0.3, -0.3, 1}, {0.3, 0.3, 0.3, 1}, {0.3, 0.3, 0.3, 1}, {0.3, -0.3, 0.3, 1}, {0.3, -0.3, -0.3, 1}},
		}

		for _, face := range cubeFaces {
			transformed := make([]graphics.Vec4, len(face))
			for i, v := range face {
				// Apply MVP
				clip := mvp.Apply(v)
				// Manual perspective divide since our shader is simple for now
				if clip[3] != 0 {
					transformed[i] = graphics.Vec4{clip[0] / clip[3], clip[1] / clip[3], clip[2] / clip[3], 1.0}
				} else {
					transformed[i] = clip
				}
			}
			app.Renderer.Render3dObject(graphics.WorldSpace, transformed, nil)
		}

		return nil
	}

	g := &opengl.OpenGL{}
	err := g.StartLoop(app)
	if err != nil {
		t.Fatalf("StartLoop failed: %v", err)
	}
}
