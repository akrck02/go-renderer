# opengl

```go
import "github.com/akrck02/go-renderer/opengl"
```

### Functions

| Function | |
|---|---|
| `func Wake()` | Wake makes a window that draws on demand draw a frame now: for a program whose data changes on another goroutine (a download finishing, a message arriving). |

### Constants and variables

```go
var (
	// FramebufferIncomplete is a framebuffer OpenGL cannot draw into, such as the shadow map's on
	// a driver that lacks depth textures. Its details give OpenGL's status.
	FramebufferIncomplete = failure.Define("opengl.framebuffer-incomplete", "OpenGL cannot draw into the framebuffer")
	// WindowNotOpened is a window GLFW could not open: no display, or no OpenGL 4.1.
	WindowNotOpened = failure.Define("opengl.window-not-opened", "the window could not be opened")
	// ContextNotStarted is OpenGL's functions failing to load in the window's context.
	ContextNotStarted = failure.Define("opengl.context-not-started", "OpenGL could not start")
	// ShaderNotCompiled is a shader the driver would not compile. Its details give the driver's log.
	ShaderNotCompiled = failure.Define("opengl.shader-not-compiled", "OpenGL could not compile a shader")
	// ProgramNotLinked is two shaders the driver would not link. Its details give the driver's log.
	ProgramNotLinked = failure.Define("opengl.program-not-linked", "OpenGL could not link a shader program")
	// FrameNotSaved is a captured frame that could not be written to its file.
	FrameNotSaved = failure.Define("opengl.frame-not-saved", "the frame could not be saved")
	// NotImplemented is an immediate-mode shape the OpenGL renderer cannot draw yet.
	NotImplemented = failure.Define("opengl.not-implemented", "the OpenGL renderer cannot draw this shape yet")
)
```

Codes of the errors the OpenGL backend returns.

### Area

Area is a rectangle of the framebuffer, in its pixels, from its bottom-left corner as OpenGL counts.

```go
type Area struct{ X, Y, Width, Height int }
```

### FlatRenderer

FlatRenderer draws a 2D draw list on a plain background, with no scene, sky or camera: the window of an application that is only an interface. It uses the overlay's atlas and shaders, so a whole frame is one clear and one draw call. Create and use it on the thread that owns the OpenGL context.

```go
type FlatRenderer struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewFlatRenderer(fonts overlay.Fonts) (*FlatRenderer, error)` | NewFlatRenderer prepares the atlas and shaders; text is drawn in fonts (overlay.Fonts{} for the default ones). |
| `func (renderer *FlatRenderer) Draw(list *draw2d.List, background draw2d.Color, width, height int, scale float32)` | Draw clears the framebuffer (width × height pixels) to the background colour and paints the list. |

### FrameStatistics

FrameStatistics describes the last frame drawn (see sceneplan.FrameStatistics).

```go
type FrameStatistics = sceneplan.FrameStatistics
```

### PassTimes

PassTimes are GPU durations of the passes of one frame.

```go
type PassTimes = sceneplan.PassTimes
```

### Renderer

Renderer draws immediate-mode shapes (immediate.Renderer) with OpenGL, in the window's current program. Its zero value is ready; it makes its buffers on the first shape and reuses them, so a shape allocates nothing on the Go side.

```go
type Renderer struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (renderer *Renderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, centre graphics.Vector, width float32, height float32, shader *string) error` | RenderImage returns NotImplemented: OpenGL cannot draw pictures in immediate mode yet. |
| `func (renderer *Renderer) RenderModel(space graphics.CoordinateSpace, model *immediate.Model, shader *string) error` | RenderModel draws every mesh of the model. |
| `func (renderer *Renderer) RenderObject(space graphics.CoordinateSpace, vertices []graphics.Vector, shader *string) error` | RenderObject draws the triangles of a 3D object, three vertices each. |
| `func (renderer *Renderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error` | RenderPolygon returns NotImplemented: OpenGL cannot draw polygons yet. |
| `func (renderer *Renderer) RenderRectangle(space graphics.CoordinateSpace, centre graphics.Vector, width float32, height float32, shader *string, fill bool) error` | RenderRectangle draws a rectangle of width × height centred on centre, in the plane z = centre's z. |
| `func (renderer *Renderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error` | RenderTriangle draws triangles, three coordinates each. |
| `func (renderer *Renderer) SetMatrices(projection, view, model graphics.Matrix)` | SetMatrices sets the matrices the next shapes are drawn with. |

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
| `func NewSceneRenderer(fonts overlay.Fonts) (*SceneRenderer, error)` | NewSceneRenderer compiles the scene shaders; the overlay's text is drawn in fonts (overlay.Fonts{} for the default ones). |
| `func (renderer *SceneRenderer) Draw(world *scene.Scene, camera camera.Lens, width, height int, seconds float64)` | Draw renders the scene from the camera into the current framebuffer of the given size, then its overlay on top. |
| `func (renderer *SceneRenderer) DrawIn(world *scene.Scene, camera camera.Lens, area Area, seconds float64)` | DrawIn renders the scene from the camera into an area of the current framebuffer, clearing only that area, and without the scene's overlay: several scenes can share a frame (a preview of a console's two screens), and an overlay over all of them is drawn once after (DrawOverlay). |
| `func (renderer *SceneRenderer) DrawOver(world *scene.Scene, camera camera.Lens, area Area, seconds float64)` | DrawOver renders the scene from the camera into an area over what the framebuffer holds there: it clears only the depth and draws no sky, so the scene floats over the one behind it (a radar in a corner of a game's view). |
| `func (renderer *SceneRenderer) DrawOverlay(world *scene.Scene, width, height int)` | DrawOverlay draws a scene's overlay (its labels, notices and drawing) over the whole framebuffer, with the camera of the last DrawIn. |

### ShadowSettings

ShadowSettings controls the sun shadows (see sceneplan.ShadowSettings).

```go
type ShadowSettings = sceneplan.ShadowSettings
```

| Function or method | |
|---|---|
| `func DefaultShadowSettings() ShadowSettings` | DefaultShadowSettings returns enabled shadows with automatic sizes. |

### Window

Window runs a window.Application in an OpenGL 4.1 window. Its zero value is ready to use.

```go
type Window struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (opengl *Window) Draw(app *window.Application) error` | Draw draws a frame: the immediate-mode matrices from the application's camera, a clear to its background, then its Draw callback. |
| `func (opengl *Window) Run(app *window.Application) error` | Run opens the window and runs the application until it closes or sets Stop; a Headless application draws one frame into a hidden window and saves it. |
| `func (opengl *Window) SaveFrame(app *window.Application, path string) error` | SaveFrame saves the frame just drawn (and not yet shown) as a PNG at path. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
