# Lesson 6: scenes from files

Building meshes in Go is good for learning and for things a program generates. Real islands are
modelled in a 3D editor, or built from a map by a tool. In this lesson you load a **glTF** file,
add the renderer's own information to it through **extras**, and carry the scene inside the
program.

## glTF in one paragraph

[glTF 2.0](https://www.khronos.org/gltf/) is the open format for 3D scenes: nodes, meshes and
materials, in a `.gltf` (JSON beside its binary buffers) or a single `.glb`. Blender exports it
(*File → Export → glTF 2.0*), and so does go-scene-builder, which turns a 2D map and a relief into
a whole scene.

## Opening a file

```go
world, err := loaders.LoadScene("island.glb")
if err != nil {
	log.Fatal(err)
}
viewer.Run(world, options, "Island", nil, 0)
```

That is the whole of `cmd/visor`, give or take its flags: `go run ./cmd/visor island.glb` opens
any file without writing a line.

The loader reads the node tree, positions, normals, vertex colours (`COLOR_0`), indices, the
base colour of materials with `doubleSided` and `alphaMode`, compressed attributes
(`KHR_mesh_quantization`) and instancing (`EXT_mesh_gpu_instancing`). Textures are not read: colour
comes from materials, vertex colours and the procedural patterns.

## Extras: what glTF cannot say

glTF has no word for the tide, the walkable ground or a label. It lets any object carry free JSON,
its **extras**, and go-renderer reads its own there. In Blender, *custom properties* become extras
when *Include → Custom Properties* is ticked in the exporter.

| On | Extra | Becomes |
|---|---|---|
| The scene | `environment`: `sunDirection`, `fog`, `water`, `wind`, `sky`, `snow`… | `world.Environment` (lesson 3) |
| A mesh, a primitive or a node | `"ground": true` | `Mesh.Ground`: walkable, found by `GroundHeight` |
| A material | `"kind": "glow"`, `"sway": 0.3`, `"pattern": "cobbles"`, `"patternScale": 0.5` | `Material.Kind`, `Sway`, `Pattern` (lessons 1–3) |
| A mesh | `"detail": [{"mesh": 3, "screenSize": 0.05}]` | Simpler versions drawn far away |
| A node | `"label"`, `"icon"`, `"badge"`, `"label_color"`, `"pinned"` | A label, or a line of the header (lesson 4) |

A node's extras, all of them, are also in `node.Tags`, for your program's own data: a lighthouse's
range, a building's owner. For example, a node exported like this:

```json
{
  "name": "harbour",
  "mesh": 0,
  "extras": { "label": "Harbour", "icon": "anchor", "berths": 12 }
}
```

gets a label with an anchor, and `node.Tags["berths"]` holds `12` (a `float64`, as JSON numbers
are). The full list is in [glTF extras](../gltf-extras.md).

## Finding nodes

A loaded scene is the same `scene.Scene` you built by hand, so everything in lessons 2 to 5 works
on it. To animate a node, find it by name:

```go
func findNode(nodes []*scene.Node, name string) *scene.Node {
	for _, node := range nodes {
		if node.Name == name {
			return node
		}
		if found := findNode(node.Children, name); found != nil {
			return found
		}
	}
	return nil
}

lamp := findNode(world.Nodes, "lamp")
```

## Carrying the scene inside the program

A program for a web page or a phone has no files beside it. Embed the `.glb` and parse its bytes:

```go
//go:embed island.glb
var islandFile []byte

world, err := loaders.ParseSceneGLB(islandFile)
```

The buffers must be inside the file, as a `.glb`'s are; a `.gltf` with separate `.bin` files needs
`LoadScene`.

## What you learnt

- `loaders.LoadScene` opens a `.glb` or `.gltf`; `ParseSceneGLB` reads embedded bytes.
- Extras carry what glTF cannot: the environment, the ground, material kinds, labels.
- A loaded scene is an ordinary `scene.Scene`: find nodes by name and change them.

## Where next

You have used every part of go-renderer a program usually needs. From here:

- **Put an interface over the island**: panels, buttons, sliders, with
  [go-ui](https://github.com/akrck02/go-ui). Its own tutorial starts where this one ends.
- **Run it elsewhere**: [in a web browser](../web.md) and [on Android](../android.md).
- **Understand the inside**: [architecture](../architecture.md), the [overlay](../overlay.md), and
  how a frame is planned and drawn.
- **Look things up** in [the reference](../reference/README.md).

---

[← Lesson 5](05-editing-with-the-pointer.md) · [Tutorial](README.md) · [Reference →](../reference/README.md)
