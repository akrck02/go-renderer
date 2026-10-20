package sceneplan

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
)

// unitCube is a mesh one unit across around its origin.
func unitCube() *scene.Mesh {
	return &scene.Mesh{Positions: []float32{-0.5, -0.5, -0.5, 0.5, 0.5, 0.5, 0.5, -0.5, 0.5},
		Normals: []float32{0, 1, 0, 0, 1, 0, 0, 1, 0}, Indices: []uint32{0, 1, 2}}
}

// lamp is a white light of intensity 1 at a point, reaching outer units.
func lamp(x, y, z, outer float32) scene.PointLight {
	return scene.PointLight{Position: graphics.Vec4{x, y, z, 1}, Color: graphics.Vec4{1, 1, 1, 1}, Outer: outer}
}

// planWithLights plans a frame of one cube at the origin under some lights.
func planWithLights(lights []scene.PointLight) Frame {
	world := scene.New()
	world.Nodes = []*scene.Node{{Name: "crate", Mesh: unitCube()}}
	world.Lights = lights
	camera := models.Camera{Position: graphics.Vec4{0, 2, 10, 1}, Up: graphics.Vec4{0, 1, 0, 0}, Near: 0.1, Far: 100}
	return NewPlanner(MergingStore{}).Plan(world, camera, 640, 480, ShadowSettings{})
}

func TestADrawTakesOnlyTheLightsThatReachIt(test *testing.T) {
	frame := planWithLights([]scene.PointLight{lamp(3, 0, 0, 2), lamp(1.5, 0, 0, 2), lamp(50, 0, 0, 100)})
	crate := frame.Commands[0].Lights
	if crate.Count != 2 || crate.Indices[0] != 1 || crate.Indices[1] != 2 {
		test.Fatalf("the crate takes %+v; wanted the near lamp, then the far one that reaches it", crate)
	}
}

func TestADrawTakesAtMostItsNearestLights(test *testing.T) {
	var lights []scene.PointLight
	for index := range 10 {
		lights = append(lights, lamp(float32(10-index), 0, 0, 20)) // the last is the nearest
	}
	crate := planWithLights(lights).Commands[0].Lights
	if crate.Count != MaxLightsPerDraw || crate.Indices[0] != 9 || crate.Indices[3] != 6 {
		test.Fatalf("the crate takes %+v; wanted the four nearest, nearest first", crate)
	}
}

func TestFrameLightsCarryTheirIntensityAndTheVerticalScale(test *testing.T) {
	world := scene.New()
	world.Environment.VerticalScale = 2
	world.Lights = []scene.PointLight{{Position: graphics.Vec4{0, 3, 0, 1}, Color: graphics.Vec4{1, 0.5, 0, 2}, Inner: 5, Outer: 4}}
	frame := NewPlanner(MergingStore{}).Plan(world, models.Camera{Up: graphics.Vec4{0, 1, 0, 0}, Near: 0.1, Far: 10}, 64, 64, ShadowSettings{})
	light := frame.Lights[0]
	if light.Position[1] != 6 || light.Color != (graphics.Vec4{2, 1, 0, 1}) || light.Inner != 4 {
		test.Fatalf("the frame's light is %+v; wanted its height scaled, its colour times 2, its inner radius within the outer", light)
	}
}

func TestChoosingLightsAllocatesNothing(test *testing.T) {
	lights := []FrameLight{{Position: graphics.Vec4{1, 0, 0, 1}, Outer: 3}, {Position: graphics.Vec4{2, 0, 0, 1}, Outer: 3}}
	bounds := box{minimum: graphics.Vec4{-1, -1, -1, 1}, maximum: graphics.Vec4{1, 1, 1, 1}}
	allocations := testing.AllocsPerRun(100, func() { nearestLights(lights, bounds, true) })
	if allocations != 0 {
		test.Fatalf("choosing lights allocates %v times", allocations)
	}
}
