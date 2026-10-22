# software

```go
import "github.com/akrck02/go-renderer/software"
```

Package software is the beginning of a software backend for the immediate-mode
Renderer contract. It will be built later: until then every function returns
NotImplemented.

### Constants and variables

```go
var NotImplemented = failure.Define("software.not-implemented", "the software backend is not implemented yet")
```

NotImplemented is what every function of the software backend returns until it is built.

### Software

```go
type Software struct{}
```

| Function or method | |
|---|---|
| `func (software *Software) Draw(app *models.Application) error` |  |
| `func (software *Software) StartLoop(app *models.Application) error` |  |

### SoftwareRenderer

```go
type SoftwareRenderer struct{}
```

| Function or method | |
|---|---|
| `func (SoftwareRenderer) Render3dModel(space graphics.CoordinateSpace, model *models.Model, shader *string) error` |  |
| `func (SoftwareRenderer) Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vec4, shader *string) error` |  |
| `func (SoftwareRenderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, coordinates graphics.Vec4, width float32, height float32, shader *string) error` |  |
| `func (SoftwareRenderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error` |  |
| `func (SoftwareRenderer) RenderRectangle(space graphics.CoordinateSpace, coordinates graphics.Vec4, width float32, height float32, shader *string, fill bool) error` |  |
| `func (SoftwareRenderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error` |  |
| `func (SoftwareRenderer) SetMatrices(projection, view, model graphics.Mat4)` |  |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
