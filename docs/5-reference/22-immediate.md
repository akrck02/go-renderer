# immediate

```go
import "github.com/akrck02/go-renderer/immediate"
```

Package immediate is the immediate-mode drawing contract: a program asks a
Renderer for a shape every frame, and the backend (OpenGL today; software,
Vulkan, Metal and WebGPU as stubs) draws it at once. Nothing is kept between
frames, which makes it the simplest way to put a triangle on the screen. Scenes
that last (a glTF world, an interface) use the retained renderers instead.

### Mesh

Mesh is a list of vertices and, optionally, the indices that join them into triangles: three indices per triangle. Without indices, every three vertices make a triangle.

```go
type Mesh struct {
	Vertices []graphics.Vector
	Indices  []uint32
}
```

### Model

Model is a set of meshes drawn together, as a file loaded with loaders.LoadModel.

```go
type Model struct {
	Meshes []Mesh
}
```

### Renderer

Renderer draws shapes at once, with the matrices last given to SetMatrices. shader names the shader to draw with; nil is the backend's own. fill draws the inside of a shape, not only its edges. A backend that cannot draw a shape yet returns its not-implemented error code.

```go
type Renderer interface {
	// RenderPolygon draws a flat polygon through the coordinates.
	RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error
	// RenderTriangle draws triangles, three coordinates each.
	RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error
	// RenderRectangle draws a rectangle of width × height centred on centre.
	RenderRectangle(space graphics.CoordinateSpace, centre graphics.Vector, width float32, height float32, shader *string, fill bool) error
	// RenderImage draws an encoded picture (PNG, JPEG) of width × height centred on centre.
	RenderImage(space graphics.CoordinateSpace, bytes []byte, centre graphics.Vector, width float32, height float32, shader *string) error
	// RenderObject draws triangles of a 3D object, three vertices each.
	RenderObject(space graphics.CoordinateSpace, vertices []graphics.Vector, shader *string) error
	// RenderModel draws every mesh of a model.
	RenderModel(space graphics.CoordinateSpace, model *Model, shader *string) error
	// SetMatrices sets the projection, view and model matrices the next shapes are drawn with.
	SetMatrices(projection, view, model graphics.Matrix)
}
```

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
