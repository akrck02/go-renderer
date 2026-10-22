package opengl

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
)

// nightGround is a grey floor twenty units across, at night, seen from straight above.
func nightGround() (*scene.Scene, models.Camera) {
	world := scene.New()
	world.Environment.SunDirection = graphics.Vector{0, -1, 0, 0}
	world.Environment.Ambient = 0.05
	floor := &scene.Mesh{Positions: []float32{-10, 0, -10, -10, 0, 10, 10, 0, 10, 10, 0, -10},
		Normals: []float32{0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0}, Indices: []uint32{0, 1, 2, 0, 2, 3},
		Material: &scene.Material{BaseColor: graphics.Vector{0.5, 0.5, 0.5, 1}}}
	world.Nodes = []*scene.Node{{Name: "floor", Mesh: floor}}
	camera := models.Camera{Position: graphics.Vector{0, 12, 0.001, 1}, Target: graphics.Vector{0, 0, 0, 1},
		Up: graphics.Vector{0, 0, -1, 0}, FieldOfView: 1, Near: 0.1, Far: 100}
	return world, camera
}

// TestALampLightsTheGroundAroundIt draws a floor at night with a lamp over its middle: the middle
// is much brighter than without the lamp, and a corner out of its reach is as dark.
func TestALampLightsTheGroundAroundIt(test *testing.T) {
	openTestContext(test, lightTestSide, lightTestSide)
	renderer, err := NewSceneRenderer()
	if err != nil {
		test.Fatal(err)
	}
	renderer.Shadows.Enabled = false
	world, camera := nightGround()
	renderer.Draw(world, camera, lightTestSide, lightTestSide, 0)
	darkMiddle, darkCorner := redAt(lightTestSide/2, lightTestSide/2), redAt(2, 2)
	world.Lights = []scene.PointLight{{Position: graphics.Vector{0, 1, 0, 1}, Color: graphics.Vector{1, 0.9, 0.7, 1.5}, Inner: 1, Outer: 4}}
	renderer.Draw(world, camera, lightTestSide, lightTestSide, 0)
	litMiddle, litCorner := redAt(lightTestSide/2, lightTestSide/2), redAt(2, 2)
	if litMiddle < darkMiddle+60 {
		test.Errorf("the middle went from %d to %d under the lamp; wanted it much brighter", darkMiddle, litMiddle)
	}
	if difference := int(litCorner) - int(darkCorner); difference > 2 || difference < -2 {
		test.Errorf("a corner out of reach went from %d to %d", darkCorner, litCorner)
	}
}

// redAt reads the red of a pixel of the back buffer, counted from the bottom-left.
func redAt(x, y int) uint8 {
	var pixel [4]uint8
	glFinishAndRead(x, y, &pixel)
	return pixel[0]
}
