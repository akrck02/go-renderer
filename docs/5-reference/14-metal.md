# metal

```go
import "github.com/akrck02/go-renderer/metal"
```

### Constants and variables

```go
var (
	// Unavailable is Metal unable to start on this computer: no Metal device, shaders that do not
	// compile, or a system other than macOS. A program can fall back to OpenGL.
	Unavailable = failure.Define("metal.unavailable", "Metal cannot start on this computer")
	// ShaderNotCompiled is a Metal shader the device would not compile.
	ShaderNotCompiled = failure.Define("metal.shader-not-compiled", "Metal could not compile a shader")
	// NothingCaptured is a capture to a picture that drew nothing.
	NothingCaptured = failure.Define("metal.nothing-captured", "the capture drew nothing")
	// NotImplemented is what the immediate-mode Window and Renderer return until they are built.
	NotImplemented = failure.Define("metal.not-implemented", "the Metal immediate-mode renderer is not implemented yet")
)
```

Codes of the errors the Metal renderers return.

### FlatWindow

FlatWindow draws through Metal, which only macOS has; elsewhere Run says so and a program uses the OpenGL backend.

```go
type FlatWindow struct {
	// Fonts are what text would be drawn with, as on macOS.
	Fonts overlay.Fonts
}
```

| Function or method | |
|---|---|
| `func (flat *FlatWindow) Run(app *window.Application) error` | Run fails: Metal is only on macOS. |

### Renderer

Renderer will draw the immediate-mode shapes (immediate.Renderer) with Metal. Until it is built, every method returns NotImplemented.

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

### SceneWindow

SceneWindow draws a scene through Metal, which only macOS has; elsewhere Run says so.

```go
type SceneWindow struct {
	// Fonts are what the overlay's text would be drawn with, as on macOS.
	Fonts overlay.Fonts
}
```

| Function or method | |
|---|---|
| `func (scene *SceneWindow) Run(app *window.Application) error` | Run fails: Metal is only on macOS. |

### Window

Window will open a window drawn with the Metal immediate-mode renderer. Until it is built, every method returns NotImplemented.

```go
type Window struct{}
```

| Function or method | |
|---|---|
| `func (*Window) Draw(app *window.Application) error` | Draw returns NotImplemented. |
| `func (*Window) Run(app *window.Application) error` | Run returns NotImplemented. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
