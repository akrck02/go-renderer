package opengl

import (
	"testing"

	"github.com/go-gl/gl/v4.1-core/gl"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
)

// BenchmarkDrawingTheSampleIsland measures a frame of the sample island, its textures included, the
// GPU's work waited for.
func BenchmarkDrawingTheSampleIsland(benchmark *testing.B) {
	measureSceneFrame(benchmark, sample.Island())
}

// BenchmarkDrawingTheSampleIslandWithoutTextures measures the same frame with every surface in its
// plain colour: what a scene without textures costs.
func BenchmarkDrawingTheSampleIslandWithoutTextures(benchmark *testing.B) {
	world := sample.Island()
	world.Walk(func(node *scene.Node, _ graphics.Matrix) {
		if node.Mesh != nil && node.Mesh.Material != nil {
			node.Mesh.Material.Texture = nil
		}
	})
	measureSceneFrame(benchmark, world)
}

func measureSceneFrame(benchmark *testing.B, world *scene.Scene) {
	openTestContext(benchmark, measuredWidth, measuredHeight)
	renderer, err := NewSceneRenderer(overlay.Fonts{})
	if err != nil {
		benchmark.Fatal(err)
	}
	lens := islandCamera()
	renderer.Draw(world, lens, measuredWidth, measuredHeight, 1)
	benchmark.ReportAllocs()
	benchmark.ResetTimer()
	for range benchmark.N {
		renderer.Draw(world, lens, measuredWidth, measuredHeight, 1)
		gl.Finish()
	}
}

// islandCamera looks at the sample island's village from the shore.
func islandCamera() camera.Lens {
	return camera.Lens{Position: graphics.Vector{60, 30, 80, 1}, Target: graphics.Vector{20, 8, 15, 1},
		Up: graphics.Vector{0, 1, 0, 0}, FieldOfViewRadians: 1, Near: 0.5, Far: 2000}
}
