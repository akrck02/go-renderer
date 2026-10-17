# metal

```go
import "github.com/akrck02/go-renderer/metal"
```

### Constants and variables

```go
var ErrUnavailable = errors.New("metal unavailable")
```

ErrUnavailable is returned (wrapped) when Metal cannot start on this computer: no Metal device, shaders that do not compile, or a system other than macOS.

### FlatWindow

FlatWindow draws through Metal, which only macOS has; elsewhere StartLoop says so and a program uses the OpenGL backend.

```go
type FlatWindow struct{}
```

| Function or method | |
|---|---|
| `func (flat *FlatWindow) StartLoop(app *models.Application) error` | StartLoop fails: Metal is only on macOS. |

### Metal

```go
type Metal struct{}
```

| Function or method | |
|---|---|
| `func (metal *Metal) Draw(app *models.Application) error` |  |
| `func (metal *Metal) StartLoop(app *models.Application) error` |  |

### MetalRenderer

```go
type MetalRenderer struct{}
```

| Function or method | |
|---|---|
| `func (MetalRenderer) Render3dModel(space graphics.CoordinateSpace, model *models.Model, shader *string) error` |  |
| `func (MetalRenderer) Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vec4, shader *string) error` |  |
| `func (MetalRenderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, coordinates graphics.Vec4, width float32, height float32, shader *string) error` |  |
| `func (MetalRenderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error` |  |
| `func (MetalRenderer) RenderRectangle(space graphics.CoordinateSpace, coordinates graphics.Vec4, width float32, height float32, shader *string, fill bool) error` |  |
| `func (MetalRenderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error` |  |
| `func (MetalRenderer) SetMatrices(projection, view, model graphics.Mat4)` |  |

### SceneWindow

SceneWindow draws a scene through Metal, which only macOS has; elsewhere StartLoop says so.

```go
type SceneWindow struct{}
```

| Function or method | |
|---|---|
| `func (scene *SceneWindow) StartLoop(app *models.Application) error` | StartLoop fails: Metal is only on macOS. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
