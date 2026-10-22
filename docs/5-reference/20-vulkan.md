# vulkan

```go
import "github.com/akrck02/go-renderer/vulkan"
```

Package vulkan is the beginning of a Vulkan backend for the immediate-mode
Renderer contract. It will be built later: until then every function returns
NotImplemented.

### Constants and variables

```go
var NotImplemented = failure.Define("vulkan.not-implemented", "the Vulkan backend is not implemented yet")
```

NotImplemented is what every function of the Vulkan backend returns until it is built.

### Renderer

Renderer will draw the immediate-mode shapes (immediate.Renderer) with Vulkan. Until it is built, every method returns NotImplemented.

```go
type Renderer struct{}
```

| Function or method | |
|---|---|
| `func (Renderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, centre graphics.Vector, width float32, height float32, shader *string) error` | RenderImage returns NotImplemented. |
| `func (Renderer) RenderModel(space graphics.CoordinateSpace, model *immediate.Model, shader *string) error` | RenderModel returns NotImplemented. |
| `func (Renderer) RenderObject(space graphics.CoordinateSpace, vertices []graphics.Vector, shader *string) error` | RenderObject returns NotImplemented. |
| `func (Renderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error` | RenderPolygon returns NotImplemented. |
| `func (Renderer) RenderRectangle(space graphics.CoordinateSpace, centre graphics.Vector, width float32, height float32, shader *string, fill bool) error` | RenderRectangle returns NotImplemented. |
| `func (Renderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error` | RenderTriangle returns NotImplemented. |
| `func (Renderer) SetMatrices(projection, view, model graphics.Matrix)` | SetMatrices does nothing yet: it has no error to return. |

### Window

Window will open a window drawn with the Vulkan immediate-mode renderer. Until it is built, every method returns NotImplemented.

```go
type Window struct{}
```

| Function or method | |
|---|---|
| `func (*Window) Draw(app *window.Application) error` | Draw returns NotImplemented. |
| `func (*Window) Run(app *window.Application) error` | Run returns NotImplemented. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
