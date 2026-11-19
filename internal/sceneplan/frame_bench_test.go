package sceneplan

import (
	"testing"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
)

// The framebuffer the benchmark plans for, in pixels.
const benchmarkWidth, benchmarkHeight = 1280, 720

// BenchmarkPlanningAFrameOfTheSampleIsland measures what a frame of the sample island costs the
// planner, with shadows, once its first frame has measured the scene and filled the batches.
func BenchmarkPlanningAFrameOfTheSampleIsland(benchmark *testing.B) {
	measurePlanning(benchmark, sample.Island())
}

// BenchmarkPlanningAFrameOfTheSampleIslandWithoutTextures measures the same frame with every
// surface in its plain colour: what planning a scene without textures costs.
func BenchmarkPlanningAFrameOfTheSampleIslandWithoutTextures(benchmark *testing.B) {
	world := sample.Island()
	world.Walk(func(node *scene.Node, _ graphics.Matrix) {
		if node.Mesh != nil && node.Mesh.Material != nil {
			node.Mesh.Material.Texture = nil
		}
	})
	measurePlanning(benchmark, world)
}

func measurePlanning(benchmark *testing.B, world *scene.Scene) {
	camera := camera.Lens{Position: graphics.Vector{0, 60, 220, 1}, Target: graphics.Vector{0, 0, 0, 1},
		Up: graphics.Vector{0, 1, 0, 0}, Near: 0.5, Far: 2000}
	planner := NewPlanner(MergingStore{})
	planner.Plan(world, camera, benchmarkWidth, benchmarkHeight, DefaultShadowSettings())
	benchmark.ReportAllocs()
	benchmark.ResetTimer()
	for range benchmark.N {
		planner.Plan(world, camera, benchmarkWidth, benchmarkHeight, DefaultShadowSettings())
	}
}

func TestPlanningAFrameAfterTheFirstAllocatesNothing(test *testing.T) {
	world := sample.Island()
	camera := camera.Lens{Position: graphics.Vector{0, 60, 220, 1}, Target: graphics.Vector{0, 0, 0, 1},
		Up: graphics.Vector{0, 1, 0, 0}, Near: 0.5, Far: 2000}
	planner := NewPlanner(MergingStore{})
	planner.Plan(world, camera, benchmarkWidth, benchmarkHeight, DefaultShadowSettings())
	allocations := testing.AllocsPerRun(100, func() {
		planner.Plan(world, camera, benchmarkWidth, benchmarkHeight, DefaultShadowSettings())
	})
	if allocations != 0 {
		test.Fatalf("a frame allocates %v times", allocations)
	}
}
