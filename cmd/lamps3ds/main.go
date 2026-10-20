//go:build nintendo3ds

// Three lamps over a floor at night on the Nintendo 3DS: an orange, a blue and a green point light
// beside three grey cubes. The 3DS bakes their light into the meshes' colours with the sun's, so
// the lamps cost nothing a frame while they stand still; the blue one moves when A is held, to
// show what baking again costs.
//
//	tinygo build -target=nintendo3ds -o lamps3ds.3dsx ./cmd/lamps3ds
package main

import (
	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
)

// lampStep is how far the blue lamp moves each frame A is held, in world units.
const lampStep = 0.05

// The floor is twenty units across in cells of half a unit: the 3DS lights by vertex, so a lamp's
// pool of light shows on a floor only where it has vertices to light.
const (
	floorSide  = 20
	floorCells = 40
)

// gridFloor is a flat square of side units, facing up, cut into cells × cells squares.
func gridFloor(side float32, cells int, colour graphics.Vec4) *scene.Mesh {
	mesh := &scene.Mesh{Material: &scene.Material{BaseColor: colour}}
	step := side / float32(cells)
	for row := 0; row <= cells; row++ {
		for column := 0; column <= cells; column++ {
			mesh.Positions = append(mesh.Positions, -side/2+float32(column)*step, 0, -side/2+float32(row)*step)
			mesh.Normals = append(mesh.Normals, 0, 1, 0)
		}
	}
	for row := range cells {
		for column := range cells {
			corner := uint32(row*(cells+1) + column)
			below := corner + uint32(cells+1)
			mesh.Indices = append(mesh.Indices, corner, below, corner+1, corner+1, below, below+1)
		}
	}
	return mesh
}

func main() {
	screen, err := ctr.Open()
	if err != nil {
		println(err.Error())
		return
	}
	ctr.OpenBottomConsole()
	renderer, err := ctr.NewSceneRenderer(screen)
	if err != nil {
		println(err.Error())
		return
	}
	world, camera := lampWorld()
	println("lamps: hold A to move the blue one")
	for frame := 0; ctr.Running(); frame++ {
		reading := ctr.ReadControls()
		if reading.Held&ctr.ButtonA != 0 {
			world.Lights[1].Position[0] = wrapped(world.Lights[1].Position[0] + lampStep)
		}
		renderer.Draw(world, camera)
		if frame%60 == 0 {
			println("bakes so far:", renderer.Bakes())
		}
	}
}

// wrapped keeps x between -5 and 5.
func wrapped(x float32) float32 {
	if x > 5 {
		return -5
	}
	return x
}
func cubeMesh(colour graphics.Vec4) *scene.Mesh {
	mesh := &scene.Mesh{Material: &scene.Material{BaseColor: colour}}
	faces := [6][3]float32{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
	for _, normal := range faces {
		var tangent, bitangent [3]float32
		if normal[1] != 0 {
			tangent, bitangent = [3]float32{1, 0, 0}, [3]float32{0, 0, 1}
		} else {
			tangent, bitangent = [3]float32{0, 1, 0}, [3]float32{normal[2], 0, -normal[0]}
		}
		base := uint32(len(mesh.Positions) / 3)
		for _, corner := range [4][2]float32{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
			for axis := range 3 {
				mesh.Positions = append(mesh.Positions, 0.5*(normal[axis]+corner[0]*tangent[axis]+corner[1]*bitangent[axis]))
				mesh.Normals = append(mesh.Normals, normal[axis])
			}
		}
		mesh.Indices = append(mesh.Indices, base, base+2, base+1, base, base+3, base+2)
	}
	return mesh
}

func lampWorld() (*scene.Scene, models.Camera) {
	world := scene.New()
	world.Environment.SunDirection = graphics.Vec4{0.3, -1, 0.2, 0}.Normalize()
	world.Environment.Ambient = 0.15
	floor := gridFloor(floorSide, floorCells, graphics.Vec4{0.55, 0.55, 0.55, 1})
	world.Nodes = []*scene.Node{{Name: "floor", Mesh: floor}}
	for index, x := range []float32{-3, 0, 3} {
		cube := cubeMesh(graphics.Vec4{0.8, 0.8, 0.8, 1})
		world.Nodes = append(world.Nodes, &scene.Node{Name: "cube" + string(rune('a'+index)), Mesh: cube,
			Transform: graphics.Translate(graphics.Vec4{x, 0.5, -1, 0})})
	}
	{
		world.Lights = []scene.PointLight{
			{Position: graphics.Vec4{-4, 1.2, 1, 1}, Color: graphics.Vec4{1, 0.55, 0.2, 2}, Inner: 0.5, Outer: 5},
			{Position: graphics.Vec4{0, 1.5, 1.5, 1}, Color: graphics.Vec4{0.3, 0.5, 1, 2}, Inner: 0.5, Outer: 4},
			{Position: graphics.Vec4{4, 1.2, 0.5, 1}, Color: graphics.Vec4{0.3, 1, 0.4, 2}, Inner: 0.5, Outer: 4.5},
		}
	}
	camera := models.Camera{Position: graphics.Vec4{0, 6, 9, 1}, Target: graphics.Vec4{0, 0, 0, 1},
		Up: graphics.Vec4{0, 1, 0, 0}, FieldOfView: 0.8, Near: 0.1, Far: 100}
	return world, camera
}
