// Package sample builds a small made-up island that uses every feature the renderers draw: relief
// with snow, the sea with a tide, a river, instanced trees with levels of detail that sway, a
// house with stone, shingle and plank patterns, a paved square, a glowing lamp, a waterfall, an
// unlit flag, a glass box, the sky with clouds, moon and a constellation, and a label. It needs no
// files, so tests and backend comparisons can draw the same scene anywhere.
package sample

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// islandRadius is the island's radius in world units; the sea around it is larger.
const islandRadius = 100

// Island returns the scene, with the sun in the afternoon. Night returns it after dark.
func Island() *scene.Scene {
	world := &scene.Scene{Environment: scene.DefaultEnvironment()}
	environment := &world.Environment
	environment.FogNear, environment.FogFar = 250, 900
	environment.Snow = &scene.Snow{Level: 31.5, Blend: 1.5, Color: graphics.Vector{0.92, 0.94, 0.98, 1}}
	environment.FoliageTint = graphics.Vector{0.75, 0.55, 0.2, 0.25}
	environment.Water = &scene.Water{Level: 0, Size: 2000, Variation: tide()}
	environment.Sky.Clouds.Coverage = 0.45
	environment.Sky.Constellations = []scene.Constellation{plough()}

	terrain := terrainMesh()
	world.Nodes = append(world.Nodes,
		&scene.Node{Name: "terrain", Transform: graphics.Identity(), Mesh: terrain},
		&scene.Node{Name: "river", Transform: graphics.Translate(graphics.Vector{-30, 0, 0, 0}), Mesh: riverMesh()},
		&scene.Node{Name: "trees", Transform: graphics.Identity(), Mesh: treeMesh(), Instances: forest(terrain)},
	)
	house := houseNode()
	world.Nodes = append(world.Nodes, house, squareNode(), lampNode(), waterfallNode(), flagNode(), glassNode())
	world.Overlay.Labels = append(world.Overlay.Labels,
		&scene.Label{Node: house, Height: 9, Text: "Harbour house", Icon: "house", Badge: "open"})
	return world
}

// Night returns the island an hour before midnight, under the moon and the stars.
func Night() *scene.Scene {
	world := Island()
	world.Environment.SunDirection = graphics.Vector{0.3, -0.45, -0.8, 0}.Normalize()
	return world
}

// The village stands on a flat terrace around this point.
const villageX, villageZ, villageRadius = 20, 15, 20

// heightAt is the island's relief: a cone with a few ridges, below the sea far from the centre,
// flattened into a terrace where the village stands.
func heightAt(x, z float64) float64 {
	terrace := naturalHeightAt(villageX, villageZ)
	blend := smoothStep(villageRadius, villageRadius+10, math.Hypot(x-villageX, z-villageZ))
	return terrace + (naturalHeightAt(x, z)-terrace)*blend
}

func naturalHeightAt(x, z float64) float64 {
	distance := math.Hypot(x, z)
	cone := 38 * (1 - distance/islandRadius)
	ridges := 3*math.Sin(x*0.08)*math.Cos(z*0.06) + 1.5*math.Sin((x+z)*0.15)
	return cone + ridges - 4
}

func smoothStep(low, high, value float64) float64 {
	share := math.Max(0, math.Min(1, (value-low)/(high-low)))
	return share * share * (3 - 2*share)
}

// terrainMesh is a grid over the island, coloured by height: sand, grass, rock.
func terrainMesh() *scene.Mesh {
	const divisions, size = 96, 2.4 * islandRadius
	mesh := &scene.Mesh{Name: "terrain", Ground: true,
		Material: &scene.Material{Name: "ground", BaseColor: graphics.Vector{1, 1, 1, 1}, Pattern: scene.PatternGrass, PatternScale: 1.5}}
	for row := 0; row <= divisions; row++ {
		for column := 0; column <= divisions; column++ {
			x := (float64(column)/divisions - 0.5) * size
			z := (float64(row)/divisions - 0.5) * size
			height := heightAt(x, z)
			mesh.Positions = append(mesh.Positions, float32(x), float32(height), float32(z))
			colour := groundColour(height)
			mesh.Colors = append(mesh.Colors, colour[0], colour[1], colour[2], 1)
		}
	}
	for row := 0; row < divisions; row++ {
		for column := 0; column < divisions; column++ {
			corner := uint32(row*(divisions+1) + column)
			below := corner + divisions + 1
			mesh.Indices = append(mesh.Indices, corner, below, corner+1, corner+1, below, below+1)
		}
	}
	mesh.ComputeNormals()
	return mesh
}

func groundColour(height float64) [3]float32 {
	switch {
	case height < 1.5:
		return [3]float32{0.76, 0.69, 0.5} // sand
	case height < 20:
		return [3]float32{0.3, 0.5, 0.22} // grass
	default:
		return [3]float32{0.45, 0.43, 0.4} // rock
	}
}

// tide raises the sea on one side of the island and lowers it on the other.
func tide() *scene.LevelVariation {
	const cells = 4
	variation := &scene.LevelVariation{MinimumX: -300, MinimumZ: -300, MaximumX: 300, MaximumZ: 300,
		Columns: cells, Rows: cells, Angle: 0.6}
	for row := 0; row < cells; row++ {
		for column := 0; column < cells; column++ {
			variation.InPhase = append(variation.InPhase, float32(column)*0.4-0.6)
			variation.InQuadrature = append(variation.InQuadrature, float32(row)*0.3-0.45)
		}
	}
	return variation
}

// riverMesh is a thin strip of water running down to the sea, drawn a little closer than the ground.
func riverMesh() *scene.Mesh {
	mesh := &scene.Mesh{Name: "river", Material: &scene.Material{Name: "river", Kind: scene.KindWater,
		BaseColor: graphics.Vector{0.12, 0.35, 0.42, 0.8}, Transparent: true, DoubleSided: true, DepthBias: 0.002}}
	for step := 0; step <= 12; step++ {
		x := float64(step) * 5
		height := float32(heightAt(x-30, 0) + 0.3)
		mesh.Positions = append(mesh.Positions, float32(x), height, -2, float32(x), height, 2)
	}
	for step := uint32(0); step < 12; step++ {
		first := step * 2
		mesh.Indices = append(mesh.Indices, first, first+1, first+2, first+2, first+1, first+3)
	}
	mesh.ComputeNormals()
	return mesh
}

// treeMesh is a tree one unit high with its base at the origin, with a coarser version for distance.
func treeMesh() *scene.Mesh {
	material := &scene.Material{Name: "leaves", BaseColor: graphics.Vector{0.22, 0.45, 0.2, 1}, Sway: 0.25, Foliage: true}
	fine := coneMesh("tree", 12, material)
	coarse := coneMesh("tree far", 4, material)
	fine.Details = []scene.DetailLevel{{Mesh: coarse, ScreenSize: 0.02}}
	return fine
}

// coneMesh is a cone of the given number of sides, unindexed.
func coneMesh(name string, sides int, material *scene.Material) *scene.Mesh {
	mesh := &scene.Mesh{Name: name, Material: material}
	for side := 0; side < sides; side++ {
		first := 2 * math.Pi * float64(side) / float64(sides)
		second := 2 * math.Pi * float64(side+1) / float64(sides)
		mesh.Positions = append(mesh.Positions,
			0, 1, 0,
			float32(0.35*math.Cos(first)), 0, float32(0.35*math.Sin(first)),
			float32(0.35*math.Cos(second)), 0, float32(0.35*math.Sin(second)))
	}
	mesh.ComputeNormals()
	return mesh
}

// forest scatters trees over the grass, on a regular pattern disturbed by a hash, each a little
// different in size and colour.
func forest(terrain *scene.Mesh) *scene.Instances {
	instances := &scene.Instances{}
	for row := -20; row <= 20; row++ {
		for column := -20; column <= 20; column++ {
			x := float64(column)*4.5 + 2*math.Sin(float64(row*31+column*17))
			z := float64(row)*4.5 + 2*math.Cos(float64(row*13+column*29))
			height := heightAt(x, z)
			if height < 2 || height > 22 || math.Hypot(x-villageX, z-villageZ) < villageRadius {
				continue // beach, rock, and the village
			}
			size := float32(4 + 2*math.Abs(math.Sin(x*z)))
			transform := graphics.Translate(graphics.Vector{float32(x), float32(height), float32(z), 0}).
				Multiply(graphics.Scale(graphics.Vector{size, size * 1.6, size, 1}))
			shade := float32(0.85 + 0.3*math.Abs(math.Cos(x+z)))
			instances.Transforms = append(instances.Transforms, transform)
			instances.Colors = append(instances.Colors, graphics.Vector{shade, shade, shade * 0.9, 1})
		}
	}
	return instances
}

// boxMesh is an axis-aligned box from minimum to maximum, unindexed, with flat faces.
func boxMesh(name string, minimum, maximum [3]float32, material *scene.Material) *scene.Mesh {
	low, high := minimum, maximum
	corner := func(x, y, z int) [3]float32 {
		pick := func(axis, side int) float32 {
			if side == 0 {
				return low[axis]
			}
			return high[axis]
		}
		return [3]float32{pick(0, x), pick(1, y), pick(2, z)}
	}
	faces := [][4][3]int{
		{{0, 0, 1}, {1, 0, 1}, {1, 1, 1}, {0, 1, 1}}, // front
		{{1, 0, 0}, {0, 0, 0}, {0, 1, 0}, {1, 1, 0}}, // back
		{{0, 0, 0}, {0, 0, 1}, {0, 1, 1}, {0, 1, 0}}, // left
		{{1, 0, 1}, {1, 0, 0}, {1, 1, 0}, {1, 1, 1}}, // right
		{{0, 1, 1}, {1, 1, 1}, {1, 1, 0}, {0, 1, 0}}, // top
		{{0, 0, 0}, {1, 0, 0}, {1, 0, 1}, {0, 0, 1}}, // bottom
	}
	mesh := &scene.Mesh{Name: name, Material: material}
	for _, face := range faces {
		for _, index := range []int{0, 1, 2, 0, 2, 3} {
			point := corner(face[index][0], face[index][1], face[index][2])
			mesh.Positions = append(mesh.Positions, point[0], point[1], point[2])
		}
	}
	mesh.ComputeNormals()
	return mesh
}

// villageGround is the height of the village's square.
func villageGround() float32 { return float32(heightAt(villageX, villageZ)) }

// houseNode is a stone house with a shingle roof and a wooden door, as children of one node.
func houseNode() *scene.Node {
	walls := boxMesh("walls", [3]float32{-4, 0, -3}, [3]float32{4, 5, 3},
		&scene.Material{Name: "stone", BaseColor: graphics.Vector{0.7, 0.66, 0.6, 1}, Pattern: scene.PatternMasonry, PatternScale: 0.5})
	roof := &scene.Mesh{Name: "roof", Material: &scene.Material{Name: "slate",
		BaseColor: graphics.Vector{0.35, 0.3, 0.32, 1}, Pattern: scene.PatternShingles, PatternScale: 0.4, DoubleSided: true}}
	roof.Positions = []float32{
		-4.4, 5, 3.4, 4.4, 5, 3.4, 4.4, 8, 0, -4.4, 5, 3.4, 4.4, 8, 0, -4.4, 8, 0,
		4.4, 5, -3.4, -4.4, 5, -3.4, -4.4, 8, 0, 4.4, 5, -3.4, -4.4, 8, 0, 4.4, 8, 0,
	}
	roof.ComputeNormals()
	door := boxMesh("door", [3]float32{-0.8, 0, 3}, [3]float32{0.8, 2.4, 3.1},
		&scene.Material{Name: "wood", BaseColor: graphics.Vector{0.45, 0.3, 0.18, 1}, Pattern: scene.PatternPlanks, PatternScale: 0.2})
	house := &scene.Node{Name: "house", Transform: graphics.Translate(graphics.Vector{20, villageGround(), 10, 0})}
	for _, mesh := range []*scene.Mesh{walls, roof, door} {
		house.Children = append(house.Children, &scene.Node{Name: mesh.Name, Transform: graphics.Identity(), Mesh: mesh})
	}
	return house
}

// squareNode is a paved square in front of the house, plastered benches around it.
func squareNode() *scene.Node {
	paving := boxMesh("square", [3]float32{-8, 0, -6}, [3]float32{8, 0.2, 6},
		&scene.Material{Name: "cobbles", BaseColor: graphics.Vector{0.6, 0.58, 0.55, 1}, Pattern: scene.PatternCobbles, PatternScale: 0.35})
	bench := boxMesh("bench", [3]float32{-3, 0, -0.4}, [3]float32{3, 0.8, 0.4},
		&scene.Material{Name: "plaster", BaseColor: graphics.Vector{0.85, 0.8, 0.7, 1}, Pattern: scene.PatternPlaster, PatternScale: 0.5})
	square := &scene.Node{Name: "square", Transform: graphics.Translate(graphics.Vector{20, villageGround() - 0.1, 20, 0}),
		Mesh: paving}
	square.Children = append(square.Children, &scene.Node{Name: "bench", Transform: graphics.Translate(graphics.Vector{0, 0.2, 4, 0}), Mesh: bench})
	return square
}

// lampNode is a post with a glowing lantern.
func lampNode() *scene.Node {
	post := boxMesh("post", [3]float32{-0.1, 0, -0.1}, [3]float32{0.1, 3, 0.1},
		&scene.Material{Name: "iron", BaseColor: graphics.Vector{0.15, 0.15, 0.16, 1}})
	lantern := boxMesh("lantern", [3]float32{-0.3, 3, -0.3}, [3]float32{0.3, 3.6, 0.3},
		&scene.Material{Name: "light", Kind: scene.KindGlow, BaseColor: graphics.Vector{1, 0.8, 0.4, 1}})
	lamp := &scene.Node{Name: "lamp", Transform: graphics.Translate(graphics.Vector{28, villageGround(), 20, 0}), Mesh: post}
	lamp.Children = append(lamp.Children, &scene.Node{Name: "lantern", Transform: graphics.Identity(), Mesh: lantern})
	return lamp
}

// waterfallNode is a sheet of falling water down a cliff on the north side.
func waterfallNode() *scene.Node {
	sheet := &scene.Mesh{Name: "waterfall", Material: &scene.Material{Name: "falling water", Kind: scene.KindWaterfall,
		BaseColor: graphics.Vector{0.8, 0.9, 0.95, 0.7}, Transparent: true, DoubleSided: true}}
	top := float32(heightAt(0, -40))
	sheet.Positions = []float32{-3, top, -40, 3, top, -40, 3, 0, -48, -3, top, -40, 3, 0, -48, -3, 0, -48}
	sheet.ComputeNormals()
	return &scene.Node{Name: "waterfall", Transform: graphics.Identity(), Mesh: sheet}
}

// flagNode is a flag in a flat colour on a pole by the harbour.
func flagNode() *scene.Node {
	flag := boxMesh("flag", [3]float32{0, 5, 0}, [3]float32{2, 6.2, 0.05},
		&scene.Material{Name: "flag", Kind: scene.KindUnlit, BaseColor: graphics.Vector{0.8, 0.15, 0.12, 1}})
	return &scene.Node{Name: "flag", Transform: graphics.Translate(graphics.Vector{12, villageGround(), 24, 0}), Mesh: flag}
}

// glassNode is a translucent glass box, a greenhouse.
func glassNode() *scene.Node {
	glass := boxMesh("greenhouse", [3]float32{-2, 0, -2}, [3]float32{2, 3, 2},
		&scene.Material{Name: "glass", BaseColor: graphics.Vector{0.7, 0.85, 0.9, 0.35}, Transparent: true})
	return &scene.Node{Name: "greenhouse", Transform: graphics.Translate(graphics.Vector{34, villageGround(), 8, 0}), Mesh: glass}
}

// plough is a made-up constellation of seven stars.
func plough() scene.Constellation {
	stars := []graphics.Vector{{0.2, 0.9, -0.3, 0}, {0.3, 0.88, -0.32, 0}, {0.38, 0.85, -0.3, 0}, {0.45, 0.82, -0.25, 0},
		{0.5, 0.78, -0.33, 0}, {0.42, 0.78, -0.4, 0}, {0.36, 0.8, -0.38, 0}}
	for index := range stars {
		stars[index] = stars[index].Normalize()
	}
	return scene.Constellation{Name: "Plough", Stars: stars,
		Lines: [][2]int{{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 6}, {6, 3}}, Color: graphics.Vector{0.7, 0.8, 1, 1}}
}
