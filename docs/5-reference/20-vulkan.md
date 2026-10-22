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

### Vulkan

```go
type Vulkan struct{}
```

| Function or method | |
|---|---|
| `func (vulkan *Vulkan) Draw(app *models.Application) error` |  |
| `func (vulkan *Vulkan) StartLoop(app *models.Application) error` |  |

### VulkanRenderer

```go
type VulkanRenderer struct{}
```

| Function or method | |
|---|---|
| `func (VulkanRenderer) Render3dModel(space graphics.CoordinateSpace, model *models.Model, shader *string) error` |  |
| `func (VulkanRenderer) Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vector, shader *string) error` |  |
| `func (VulkanRenderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, coordinates graphics.Vector, width float32, height float32, shader *string) error` |  |
| `func (VulkanRenderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error` |  |
| `func (VulkanRenderer) RenderRectangle(space graphics.CoordinateSpace, coordinates graphics.Vector, width float32, height float32, shader *string, fill bool) error` |  |
| `func (VulkanRenderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error` |  |
| `func (VulkanRenderer) SetMatrices(projection, view, model graphics.Matrix)` |  |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
