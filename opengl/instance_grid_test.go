package opengl

import (
	"math/rand"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

var unitBox = box{graphics.Vec4{-0.5, 0, -0.5, 1}, graphics.Vec4{0.5, 1, 0.5, 1}}

func forest(count int) *scene.Instances {
	random := rand.New(rand.NewSource(1))
	instances := &scene.Instances{}
	for index := 0; index < count; index++ {
		position := graphics.Vec4{random.Float32() * 100, 0, random.Float32() * 100, 0}
		instances.Transforms = append(instances.Transforms, graphics.Translate(position))
	}
	return instances
}

func TestGridKeepsEveryInstanceOnce(t *testing.T) {
	instances := forest(5000)
	grid := buildInstanceGrid(instances, unitBox)
	total := 0
	for _, cell := range grid.cells {
		total += cell.count
	}
	if total != 5000 || len(grid.packed) != 5000*floatsPerInstance {
		t.Fatalf("grid holds %d instances", total)
	}
	for instance, transform := range instances.Transforms {
		if unpackTransform(grid.packed, grid.slots[instance]) != transform {
			t.Fatalf("instance %d is not in its slot", instance)
		}
	}
}

func TestUpdatedInstanceMovesAndWidensItsCell(t *testing.T) {
	instances := forest(5000)
	grid := buildInstanceGrid(instances, unitBox)
	instances.Transforms[42] = graphics.Translate(graphics.Vec4{500, 0, 500, 0})
	cell := grid.updateInstance(instances, 42, unitBox)
	if unpackTransform(grid.packed, grid.slots[42]) != instances.Transforms[42] {
		t.Fatalf("the packed instance was not updated")
	}
	if grid.cells[cell].bounds.maximum[0] < 500 {
		t.Fatalf("the cell box should grow to contain the moved instance")
	}
}

// Compares preparing 87,000 instances again with updating 1,000 of them.
func BenchmarkRebuildGrid(b *testing.B) {
	instances := forest(87000)
	for iteration := 0; iteration < b.N; iteration++ {
		buildInstanceGrid(instances, unitBox)
	}
}

func BenchmarkUpdateThousandInstances(b *testing.B) {
	instances := forest(87000)
	grid := buildInstanceGrid(instances, unitBox)
	for iteration := 0; iteration < b.N; iteration++ {
		for instance := 0; instance < 1000; instance++ {
			grid.updateInstance(instances, instance*87, unitBox)
		}
	}
}
