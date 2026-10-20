package sceneplan

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
)

func litTriangle() *scene.Mesh {
	return &scene.Mesh{
		Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0},
		Normals:   []float32{0, 0, 1, 0, 0, 1, 0, 0, 1},
		Material:  &scene.Material{Kind: scene.KindLit, BaseColor: graphics.Vec4{1, 1, 1, 1}},
	}
}

func TestASceneWithoutWaterHasNoSeabed(t *testing.T) {
	world := scene.New()
	mesh := litTriangle()
	world.Nodes = []*scene.Node{{Transform: graphics.Identity(), Mesh: mesh}}
	planner := NewPlanner(MergingStore{})
	camera := models.Camera{Position: graphics.Vec4{0, 0, 5, 1}, Up: graphics.Vec4{0, 1, 0, 0}}
	mesh.Dirty = true // a changed mesh made the seabed measure every caster again
	if plan := planner.Plan(world, camera, 400, 240, ShadowSettings{}).Seabed; plan.Available || plan.Redraw {
		t.Fatalf("got %+v, want no seabed without water", plan)
	}
}

func TestASceneWithWaterStillPlansItsSeabed(t *testing.T) {
	world := scene.New()
	world.Environment.Water = &scene.Water{Size: 100}
	world.Nodes = []*scene.Node{{Transform: graphics.Identity(), Mesh: litTriangle()}}
	planner := NewPlanner(MergingStore{})
	camera := models.Camera{Position: graphics.Vec4{0, 0, 5, 1}, Up: graphics.Vec4{0, 1, 0, 0}}
	if plan := planner.Plan(world, camera, 400, 240, ShadowSettings{}).Seabed; !plan.Available {
		t.Fatalf("got %+v, want a seabed", plan)
	}
}
