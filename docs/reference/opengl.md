# opengl

```go
import "github.com/akrck02/go-renderer/opengl"
```

### Functions

| Function | |
|---|---|
| `func CompileShader(` |  |
| `func InitWindow(app *models.Application) *glfw.Window` | initGlfw initializes glfw and returns a Window to use. |
| `func LinkProgram(vertexShader, fragmentShader uint32) (uint32, error)` |  |
| `func Wake()` | Wake makes a window that draws on demand draw a frame now: for a program whose data changes on another goroutine (a download finishing, a message arriving). |

### FlatRenderer

FlatRenderer draws a 2D draw list on a plain background, with no scene, sky or camera: the window of an application that is only an interface. It uses the overlay's atlas and shaders, so a whole frame is one clear and one draw call. Create and use it on the thread that owns the OpenGL context.

```go
type FlatRenderer struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewFlatRenderer() (*FlatRenderer, error)` | NewFlatRenderer prepares the atlas and shaders. |
| `func (renderer *FlatRenderer) Draw(list *draw2d.List, background draw2d.Color, width, height int, scale float32)` | Draw clears the framebuffer (width × height pixels) to the background colour and paints the list. |

### FrameStatistics

FrameStatistics describes the last frame drawn (see sceneplan.FrameStatistics).

```go
type FrameStatistics = sceneplan.FrameStatistics
```

### OpenGL

```go
type OpenGL struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (opengl *OpenGL) Draw(app *models.Application) error` | Draw a frame into the display |
| `func (opengl *OpenGL) DrawOnDisk(app *models.Application, filePath string) error` | SaveFrameToDisk captures the current OpenGL framebuffer and saves it as a PNG. |
| `func (opengl *OpenGL) StartLoop(app *models.Application) error` | StartLoop opens the window and runs the application until it closes. |

### OpenGLRenderer

```go
type OpenGLRenderer struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (r *OpenGLRenderer) Render3dModel(space graphics.CoordinateSpace, model *models.Model, shader *string) error` |  |
| `func (r *OpenGLRenderer) Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vec4, shader *string) error` |  |
| `func (r *OpenGLRenderer) RenderImage(` |  |
| `func (r *OpenGLRenderer) RenderPolygon(` |  |
| `func (r *OpenGLRenderer) RenderRectangle(` |  |
| `func (r *OpenGLRenderer) RenderTriangle(` |  |
| `func (r *OpenGLRenderer) SetMatrices(projection, view, model graphics.Mat4)` |  |

### PassTimes

PassTimes are GPU durations of the passes of one frame.

```go
type PassTimes = sceneplan.PassTimes
```

### SceneRenderer

SceneRenderer draws a scene.Scene. Meshes and instances are uploaded once and kept on the GPU; they are uploaded again only when marked Dirty (for simulations). It must be created and used on the thread that owns the OpenGL context.

```go
type SceneRenderer struct {
	Shadows    ShadowSettings
	Profiling  bool            // measure CPU and GPU times into Statistics (waits for the GPU every frame)
	Statistics FrameStatistics // the last frame drawn
	// OverlayScale is how many framebuffer pixels make one pixel of the overlay (labels, header,
	// notices): 2 on a high-density screen, so they keep their size; 0 = 1.
	OverlayScale float32

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewSceneRenderer() (*SceneRenderer, error)` | NewSceneRenderer compiles the scene shaders. |
| `func (renderer *SceneRenderer) Draw(world *scene.Scene, camera models.Camera, width, height int, seconds float64)` | Draw renders the scene from the camera into the current framebuffer of the given size, then its overlay on top. |

### ShadowSettings

ShadowSettings controls the sun shadows (see sceneplan.ShadowSettings).

```go
type ShadowSettings = sceneplan.ShadowSettings
```

| Function or method | |
|---|---|
| `func DefaultShadowSettings() ShadowSettings` | DefaultShadowSettings returns enabled shadows with automatic sizes. |

---

[← Reference](README.md) · generated from the doc comments by `generate.py`
