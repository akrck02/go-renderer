package sceneplan

import (
	"testing"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// unitCube is a mesh one unit across around its origin.
func unitCube() *scene.Mesh {
	return &scene.Mesh{Positions: []float32{-0.5, -0.5, -0.5, 0.5, 0.5, 0.5, 0.5, -0.5, 0.5},
		Normals: []float32{0, 1, 0, 0, 1, 0, 0, 1, 0}, Indices: []uint32{0, 1, 2}}
}

// lamp is a white light of intensity 1 at a point, reaching outer units.
func lamp(x, y, z, outer float32) scene.PointLight {
	return scene.PointLight{Position: graphics.Vector{x, y, z, 1}, Color: graphics.Vector{1, 1, 1, 1}, Outer: outer}
}

// planWithLights plans a frame of one cube at the origin under some lights.
func planWithLights(lights []scene.PointLight) Frame {
	world := scene.New()
	world.Nodes = []*scene.Node{{Name: "crate", Mesh: unitCube()}}
	world.Lights = lights
	camera := camera.Camera{Position: graphics.Vector{0, 2, 10, 1}, Up: graphics.Vector{0, 1, 0, 0}, Near: 0.1, Far: 100}
	return NewPlanner(MergingStore{}).Plan(world, camera, 640, 480, ShadowSettings{})
}

func TestADrawTakesOnlyTheLightsThatReachIt(test *testing.T) {
	frame := planWithLights([]scene.PointLight{lamp(3, 0, 0, 2), lamp(1.5, 0, 0, 2), lamp(50, 0, 0, 100)})
	crate := frame.Commands[0].Lights
	if crate.Count != 2 || crate.Indices[0] != 1 || crate.Indices[1] != 2 {
		test.Fatalf("the crate takes %+v; wanted the near lamp and the far one that reaches it", crate)
	}
}

func TestADrawTakesAtMostItsNearestLights(test *testing.T) {
	var lights []scene.PointLight
	for index := range 10 {
		lights = append(lights, lamp(float32(10-index), 0, 0, 20)) // the last is the nearest
	}
	crate := planWithLights(lights).Commands[0].Lights
	if crate.Count != MaxLightsPerDraw || crate.Indices != [MaxLightsPerDraw]uint16{6, 7, 8, 9} {
		test.Fatalf("the crate takes %+v; wanted the four nearest, by their index", crate)
	}
}

func TestFrameLightsCarryTheirIntensityAndTheVerticalScale(test *testing.T) {
	world := scene.New()
	world.Environment.VerticalScale = 2
	world.Lights = []scene.PointLight{{Position: graphics.Vector{0, 3, 0, 1}, Color: graphics.Vector{1, 0.5, 0, 2}, Inner: 5, Outer: 4}}
	frame := NewPlanner(MergingStore{}).Plan(world, camera.Camera{Up: graphics.Vector{0, 1, 0, 0}, Near: 0.1, Far: 10}, 64, 64, ShadowSettings{})
	light := frame.Lights[0]
	if light.Position[1] != 6 || light.Color != (graphics.Vector{2, 1, 0, 1}) || light.Inner >= 4 || light.Inner < 3.99 {
		test.Fatalf("the frame's light is %+v; wanted its height scaled, its colour times 2, its inner radius within the outer", light)
	}
}

func TestChoosingLightsAllocatesNothing(test *testing.T) {
	lights := []FrameLight{{Position: graphics.Vector{1, 0, 0, 1}, Outer: 3}, {Position: graphics.Vector{2, 0, 0, 1}, Outer: 3}}
	bounds := box{minimum: graphics.Vector{-1, -1, -1, 1}, maximum: graphics.Vector{1, 1, 1, 1}}
	allocations := testing.AllocsPerRun(100, func() { nearestLights(lights, bounds) })
	if allocations != 0 {
		test.Fatalf("choosing lights allocates %v times", allocations)
	}
}

func TestInstancesTakeTheLightsThatReachAnyOfThem(test *testing.T) {
	world := scene.New()
	far := graphics.Translate(graphics.Vector{40, 0, 0, 0})
	instances := &scene.Instances{Transforms: []graphics.Matrix{graphics.Identity(), far}, Colors: []graphics.Vector{{1, 1, 1, 1}, {1, 1, 1, 1}}}
	world.Nodes = []*scene.Node{{Name: "forest", Mesh: unitCube(), Instances: instances}}
	world.Lights = []scene.PointLight{lamp(41, 0, 0, 3), lamp(0, 30, 0, 3)} // by the far tree; above, out of reach
	camera := camera.Camera{Position: graphics.Vector{0, 2, 10, 1}, Up: graphics.Vector{0, 1, 0, 0}, Near: 0.1, Far: 100}
	forest := NewPlanner(MergingStore{}).Plan(world, camera, 640, 480, ShadowSettings{}).Commands[0].Lights
	if forest.Count != 1 || forest.Indices[0] != 0 {
		test.Fatalf("the forest takes %+v; wanted the lamp by its far tree only", forest)
	}
}
