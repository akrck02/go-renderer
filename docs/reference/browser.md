# browser

```go
import "github.com/akrck02/go-renderer/browser"
```

> Built only with `GOOS=js`.

Package browser runs go-renderer programs in a web browser, built with GOOS=js
GOARCH=wasm: a canvas stands in for the window, its DOM events become an
input.State, WebGL2 draws through the gles renderers, and requestAnimationFrame
drives the same fixed-step loop as the desktop, drawing only when something
happens.

### Functions

| Function | |
|---|---|
| `func Run(canvas *Canvas, app *models.Application, drawFrame func() error) error` | Run drives an application in the canvas until it sets Stop: update steps at the fixed step and drawFrame to draw one, called from requestAnimationFrame. |
| `func Stop()` | Stop ends the running program's loop after one more frame, so that Run returns: for a page that takes the canvas away (an application of the page's own leaving the screen). |
| `func Wake()` | Wake makes a canvas that draws on demand draw a frame now: for a program whose data changes on another goroutine (a fetch finishing). |

### Constants and variables

```go
var ErrNoCanvas = errors.New("browser: no canvas with that id in the page")
```

ErrNoCanvas is returned when the page has no canvas with the given id.

```go
var ErrNoWebGL2 = errors.New("browser: this browser has no WebGL2")
```

ErrNoWebGL2 is returned when the browser cannot give the canvas a WebGL2 context.

### Canvas

Canvas is the page's drawing surface and the source of its input. It fills its place in the page (its CSS size) and draws at the screen's density, so text stays sharp on high-density screens.

```go
type Canvas struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func OpenCanvas(id string) (*Canvas, error)` | OpenCanvas takes the page's canvas with this id (the first canvas when id is empty). |
| `func (canvas *Canvas) Attach(state *input.State)` | Attach makes the canvas fill an input.State from now on: pointer, wheel, keyboard and size. |
| `func (canvas *Canvas) Detach()` | Detach takes off what Attach added: a program that stops, while the page goes on, leaves no listener behind (the document's would call into a program that has exited). |
| `func (canvas *Canvas) WebGL(antialias bool) (*WebGL, error)` | WebGL returns the canvas's WebGL2 context as gles.Functions. |

### WebGL

WebGL implements gles.Functions over a canvas's WebGL2 context. JavaScript objects (shaders, buffers, textures, uniform locations) are kept in tables and handed to the renderers as numbers, which is what the gles handles are.

```go
type WebGL struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (webgl *WebGL) ActiveTexture(unit gles.Enum)` |  |
| `func (webgl *WebGL) AttachShader(program gles.Program, shader gles.Shader)` |  |
| `func (webgl *WebGL) BindAttribLocation(program gles.Program, index uint32, name string)` |  |
| `func (webgl *WebGL) BindBuffer(target gles.Enum, buffer gles.Buffer)` |  |
| `func (webgl *WebGL) BindTexture(target gles.Enum, texture gles.Texture)` |  |
| `func (webgl *WebGL) BlendFuncSeparate(sourceColor, destinationColor, sourceAlpha, destinationAlpha gles.Enum)` |  |
| `func (webgl *WebGL) BufferData(target gles.Enum, data []byte, usage gles.Enum)` |  |
| `func (webgl *WebGL) Clear(mask gles.Enum)` |  |
| `func (webgl *WebGL) ClearColor(red, green, blue, alpha float32)` |  |
| `func (webgl *WebGL) CompileShader(shader gles.Shader)` |  |
| `func (webgl *WebGL) CreateBuffer() gles.Buffer` |  |
| `func (webgl *WebGL) CreateProgram() gles.Program` |  |
| `func (webgl *WebGL) CreateShader(stage gles.Enum) gles.Shader` |  |
| `func (webgl *WebGL) CreateTexture() gles.Texture` |  |
| `func (webgl *WebGL) DeleteShader(shader gles.Shader)` |  |
| `func (webgl *WebGL) DepthFunc(function gles.Enum)` |  |
| `func (webgl *WebGL) DepthMask(write bool)` |  |
| `func (webgl *WebGL) Disable(capability gles.Enum)` |  |
| `func (webgl *WebGL) DisableVertexAttribArray(index uint32)` |  |
| `func (webgl *WebGL) DrawArrays(mode gles.Enum, first, count int)` |  |
| `func (webgl *WebGL) DrawElements(mode gles.Enum, count int, kind gles.Enum, offset int)` |  |
| `func (webgl *WebGL) Enable(capability gles.Enum)` |  |
| `func (webgl *WebGL) EnableVertexAttribArray(index uint32)` |  |
| `func (webgl *WebGL) LinkProgram(program gles.Program)` |  |
| `func (webgl *WebGL) PixelStorei(name gles.Enum, value int32)` |  |
| `func (webgl *WebGL) ProgramLinked(program gles.Program) (bool, string)` |  |
| `func (webgl *WebGL) ShaderCompiled(shader gles.Shader) (bool, string)` |  |
| `func (webgl *WebGL) ShaderSource(shader gles.Shader, source string)` |  |
| `func (webgl *WebGL) TexImage2D(target gles.Enum, level int, internalFormat gles.Enum, width, height int, format, kind gles.Enum, pixels []byte)` |  |
| `func (webgl *WebGL) TexParameteri(target, name gles.Enum, value int32)` |  |
| `func (webgl *WebGL) TexSubImage2D(target gles.Enum, level, x, y, width, height int, format, kind gles.Enum, pixels []byte)` |  |
| `func (webgl *WebGL) Uniform1f(location gles.Uniform, value float32)` |  |
| `func (webgl *WebGL) Uniform1i(location gles.Uniform, value int32)` |  |
| `func (webgl *WebGL) Uniform2f(location gles.Uniform, first, second float32)` |  |
| `func (webgl *WebGL) Uniform3f(location gles.Uniform, first, second, third float32)` |  |
| `func (webgl *WebGL) Uniform4f(location gles.Uniform, first, second, third, fourth float32)` |  |
| `func (webgl *WebGL) UniformLocation(program gles.Program, name string) gles.Uniform` |  |
| `func (webgl *WebGL) UniformMatrix3(location gles.Uniform, matrix [9]float32)` |  |
| `func (webgl *WebGL) UniformMatrix4(location gles.Uniform, matrix [16]float32)` |  |
| `func (webgl *WebGL) UseProgram(program gles.Program)` |  |
| `func (webgl *WebGL) VertexAttribPointer(index uint32, size int, kind gles.Enum, normalized bool, stride, offset int)` |  |
| `func (webgl *WebGL) Viewport(x, y, width, height int)` |  |

---

[← Reference](README.md) · generated from the doc comments by `generate.py`
