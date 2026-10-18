# Lesson 1: meshes and nodes

In this lesson you build the ground of the island, a hill, and stand a tower on it. You learn what
a mesh is made of and how a node places it.

| Mission | Pieces | Time |
|---|---|---|
| A tower on a hill | `scene.Mesh`, `scene.Node`, materials, `viewer.Run` | 15 min |

#### The idea

A mesh is **a tent made of triangles**: a list of poles (the vertices) and which three of them hold up each piece of cloth (the indices). The node is **where you pitch the tent**: a position, a turn and a size. Build the tent once; pitch it anywhere, as often as you like.

## Coordinates

Go Renderer's world has **y up**. The ground lies in the x–z plane, and north is towards −z (the
viewer's map shows it at the top). Units are whatever you decide: in this tutorial one unit is a
metre, so the tower is eight metres tall.

## A mesh is a list of triangles

A `scene.Mesh` keeps its numbers in flat slices, the way the graphics card wants them:

| Field | Holds | Example |
|---|---|---|
| `Positions` | x, y, z of every vertex, one after another | `{0,0,0, 1,0,0, 0,0,1}` is three vertices |
| `Indices` | Three vertex numbers per triangle (optional: without them, every three vertices are one) | `{0, 2, 1}` |
| `Normals` | The direction each vertex faces, for the light | Filled by `ComputeNormals()` |
| `Colors` | r, g, b, a per vertex (optional) | Multiplied with the material's colour |
| `Material` | How the surface is shaded | `&scene.Material{Kind: scene.KindLit, BaseColor: …}` |
| `Ground` | Whether people can walk on it | `true` for the hill |

Triangles are **counter-clockwise when seen from the side they face**: the order of the three
indices decides which side is the front. Get it backwards and the light falls on the wrong side
(or `Material.DoubleSided` shades both).

### The hill

A grid of squares, each made of two triangles, whose height falls from the middle to the edge:

```go
// hill is a round island: a grid of side × side cells, highest in the middle and under the sea at
// its edges. Its surface is ground, so the walker and GroundHeight find it.
func hill(side int, height float32) *scene.Mesh {
	mesh := &scene.Mesh{
		Name:   "hill",
		Ground: true,
		Material: &scene.Material{
			Kind:         scene.KindLit,
			BaseColor:    graphics.Vec4{0.45, 0.6, 0.3, 1},
			Pattern:      scene.PatternGrass,
			PatternScale: 1,
		},
	}
	half := float32(side) / 2
	for row := 0; row <= side; row++ {
		for column := 0; column <= side; column++ {
			x, z := float32(column)-half, float32(row)-half
			distance := math.Hypot(float64(x), float64(z)) / float64(half) // 0 in the middle, 1 at the edge
			y := height * float32(math.Cos(math.Min(distance, 1.3)*math.Pi/2))
			mesh.Positions = append(mesh.Positions, x, y, z)
		}
	}
	for row := range side {
		for column := range side {
			corner := uint32(row*(side+1) + column)
			below := corner + uint32(side+1)
			mesh.Indices = append(mesh.Indices, corner, below, corner+1, corner+1, below, below+1)
		}
	}
	mesh.ComputeNormals()
	return mesh
}
```

`ComputeNormals` works out smooth normals from the triangles, so the hill is shaded as one rounded
surface. `PatternGrass` mottles its colour from the world position, with no texture to load; the
other patterns are cobbles, masonry, plaster, shingles and planks.

### The tower

A box needs sharp edges, so each face gets vertices of its own (a shared corner would have one
normal, halfway between two faces, and the box would look like a blurred pebble):

```go
// box is a box standing on the origin: width along x, height up y, depth along z. Each face has
// vertices of its own, so its normals stay flat and its edges sharp.
func box(width, height, depth float32, color graphics.Vec4) *scene.Mesh {
	x, z := width/2, depth/2
	corners := [8][3]float32{
		{-x, 0, -z}, {x, 0, -z}, {x, 0, z}, {-x, 0, z}, // bottom
		{-x, height, -z}, {x, height, -z}, {x, height, z}, {-x, height, z}, // top
	}
	faces := [6][4]int{{0, 1, 2, 3}, {7, 6, 5, 4}, {3, 2, 6, 7}, {1, 0, 4, 5}, {2, 1, 5, 6}, {0, 3, 7, 4}}
	mesh := &scene.Mesh{Name: "box", Material: &scene.Material{Kind: scene.KindLit, BaseColor: color}}
	for _, face := range faces {
		first := uint32(len(mesh.Positions) / 3)
		for _, corner := range face {
			mesh.Positions = append(mesh.Positions, corners[corner][:]...)
		}
		mesh.Indices = append(mesh.Indices, first, first+1, first+2, first, first+2, first+3)
	}
	mesh.ComputeNormals()
	return mesh
}
```

## A node places a mesh

A mesh is drawn where a `scene.Node` puts it. The node's `Transform` is a 4×4 matrix: move it with
`graphics.Translate`, turn it with `graphics.RotateY` (or `RotateX`, `Rotate` about any axis),
size it with `graphics.Scale`, and combine them with `Multiply`. The hill is four metres high in
the middle, so the tower stands at `y = 4`:

```go
func main() {
	options := viewer.DefaultOptions()
	options.RegisterFlags(flag.CommandLine) // -capture, -night, -walk and the rest
	flag.Parse()

	world := scene.New()

	island := &scene.Node{Name: "island", Transform: graphics.Identity(), Mesh: hill(60, 4)}
	tower := &scene.Node{
		Name:      "lighthouse",
		Transform: graphics.Translate(graphics.Vec4{0, 4, 0, 1}),
		Mesh:      box(2, 8, 2, graphics.Vec4{0.9, 0.9, 0.85, 1}),
	}
	world.Nodes = append(world.Nodes, island, tower)

	if err := viewer.Run(world, options, "Island", nil, 0); err != nil {
		log.Fatal(err)
	}
}
```

> 📝 **`Vec4`'s fourth number.** A *point* has 1 there and a *direction* 0: a translation moves
> points but not directions. Colours use it as opacity.

`options.RegisterFlags` gives your program every flag of the viewer for free: try
`go run . -walk`, `go run . -map` or `go run . -capture island.png`.

![A white tower on a green hill](../images/tutorial/tutorial-1.jpg)

#### Checkpoint

A cream tower stands on a green hill, its shadow on the grass. Press `Tab` and walk round it, at eye height on the hill.

![Achievement unlocked: Mason](../images/tutorial/quest/achievement-mason.svg)

## What you learnt

- A mesh is positions and indices, counter-clockwise from the front, with normals for the light.
- A material says how it is shaded; `Pattern` adds surface detail with no texture.
- A node places a mesh with a matrix; `Ground: true` makes it walkable.

## Bonus (optional)

Put a small house beside the tower: a box 3 m wide, 2.5 m tall and 4 m deep, painted in planks,
four metres east of it and turned 30°.

<details>
<summary>🔓 Show the solution</summary>

```go
house := box(3, 2.5, 4, graphics.Vec4{0.6, 0.45, 0.3, 1})
house.Material.Pattern, house.Material.PatternScale = scene.PatternPlanks, 0.3
height, _ := world.GroundHeight(4, 0)
world.Nodes = append(world.Nodes, &scene.Node{
	Name:      "house",
	Transform: graphics.Translate(graphics.Vec4{4, height, 0, 1}).Multiply(graphics.RotateY(30 * math.Pi / 180)),
	Mesh:      house,
})
```

`GroundHeight` asks the scene how high the walkable ground is at a point, so the house sits on the
slope. The rightmost matrix of a `Multiply` acts first: the house turns about its own middle, then
moves.

</details>

---

[← Tutorial](0-the-quest.md) · [Lesson 2: making things move →](2-making-things-move.md)
