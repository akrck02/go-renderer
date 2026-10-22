# webgpu

```go
import "github.com/akrck02/go-renderer/webgpu"
```

Package webgpu is the beginning of a WebGPU backend for the immediate-mode
Renderer contract. It will be built later: until then every function returns
NotImplemented.

### Constants and variables

```go
var NotImplemented = failure.Define("webgpu.not-implemented", "the WebGPU backend is not implemented yet")
```

NotImplemented is what every function of the WebGPU backend returns until it is built.

### WebGPU

```go
type WebGPU struct{}
```

| Function or method | |
|---|---|
| `func (webgpu *WebGPU) Draw(app *models.Application) error` |  |
| `func (webgpu *WebGPU) StartLoop(app *models.Application) error` |  |

### WebGPURenderer

```go
type WebGPURenderer struct{}
```

| Function or method | |
|---|---|
| `func (WebGPURenderer) Render3dModel(space graphics.CoordinateSpace, model *models.Model, shader *string) error` |  |
| `func (WebGPURenderer) Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vec4, shader *string) error` |  |
| `func (WebGPURenderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, coordinates graphics.Vec4, width float32, height float32, shader *string) error` |  |
| `func (WebGPURenderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error` |  |
| `func (WebGPURenderer) RenderRectangle(space graphics.CoordinateSpace, coordinates graphics.Vec4, width float32, height float32, shader *string, fill bool) error` |  |
| `func (WebGPURenderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error` |  |
| `func (WebGPURenderer) SetMatrices(projection, view, model graphics.Mat4)` |  |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
