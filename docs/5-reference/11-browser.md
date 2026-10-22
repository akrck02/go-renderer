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
var (
	// CanvasNotFound is a page with no canvas of the id asked for (or no canvas at all).
	CanvasNotFound = failure.Define("browser.canvas-not-found", "the page has no canvas with that id")
	// WebGL2Missing is a browser that cannot give the canvas a WebGL2 context.
	WebGL2Missing = failure.Define("browser.webgl2-missing", "this browser has no WebGL2")
)
```

Codes of the errors OpenCanvas and the renderers return.

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
| `func (canvas *Canvas) Attach(state *input.State)` | Attach makes the canvas fill an input.State from now on: pointer, wheel, keyboard, size and the layout of a pad that connects. |
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
| `func (webgl *WebGL) ActiveTexture(unit gles.Enum)` | ActiveTexture implements gles.Functions. |
| `func (webgl *WebGL) AttachShader(program gles.Program, shader gles.Shader)` | AttachShader implements gles.Functions. |
| `func (webgl *WebGL) BindAttribLocation(program gles.Program, index uint32, name string)` | BindAttribLocation implements gles.Functions. |
| `func (webgl *WebGL) BindBuffer(target gles.Enum, buffer gles.Buffer)` | BindBuffer implements gles.Functions. |
| `func (webgl *WebGL) BindFramebuffer(target gles.Enum, framebuffer gles.Framebuffer)` | BindFramebuffer implements gles.Functions. |
| `func (webgl *WebGL) BindTexture(target gles.Enum, texture gles.Texture)` | BindTexture implements gles.Functions. |
| `func (webgl *WebGL) BlendFuncSeparate(sourceColor, destinationColor, sourceAlpha, destinationAlpha gles.Enum)` | BlendFuncSeparate implements gles.Functions. |
| `func (webgl *WebGL) BufferData(target gles.Enum, data []byte, usage gles.Enum)` | BufferData implements gles.Functions. |
| `func (webgl *WebGL) Clear(mask gles.Enum)` | Clear implements gles.Functions. |
| `func (webgl *WebGL) ClearColor(red, green, blue, alpha float32)` | ClearColor implements gles.Functions. |
| `func (webgl *WebGL) CompileShader(shader gles.Shader)` | CompileShader implements gles.Functions. |
| `func (webgl *WebGL) CopyTexSubImage2D(target gles.Enum, level, xOffset, yOffset, x, y, width, height int)` | CopyTexSubImage2D implements gles.Functions. |
| `func (webgl *WebGL) CreateBuffer() gles.Buffer` | CreateBuffer implements gles.Functions. |
| `func (webgl *WebGL) CreateFramebuffer() gles.Framebuffer` | CreateFramebuffer implements gles.Functions. |
| `func (webgl *WebGL) CreateProgram() gles.Program` | CreateProgram implements gles.Functions. |
| `func (webgl *WebGL) CreateShader(stage gles.Enum) gles.Shader` | CreateShader implements gles.Functions. |
| `func (webgl *WebGL) CreateTexture() gles.Texture` | CreateTexture implements gles.Functions. |
| `func (webgl *WebGL) DeleteFramebuffer(framebuffer gles.Framebuffer)` | DeleteFramebuffer implements gles.Functions. |
| `func (webgl *WebGL) DeleteShader(shader gles.Shader)` | DeleteShader implements gles.Functions. |
| `func (webgl *WebGL) DeleteTexture(texture gles.Texture)` | DeleteTexture implements gles.Functions. |
| `func (webgl *WebGL) DepthFunc(function gles.Enum)` | DepthFunc implements gles.Functions. |
| `func (webgl *WebGL) DepthMask(write bool)` | DepthMask implements gles.Functions. |
| `func (webgl *WebGL) Disable(capability gles.Enum)` | Disable implements gles.Functions. |
| `func (webgl *WebGL) DisableVertexAttribArray(index uint32)` | DisableVertexAttribArray implements gles.Functions. |
| `func (webgl *WebGL) DrawArrays(mode gles.Enum, first, count int)` | DrawArrays implements gles.Functions. |
| `func (webgl *WebGL) DrawElements(mode gles.Enum, count int, kind gles.Enum, offset int)` | DrawElements implements gles.Functions. |
| `func (webgl *WebGL) Enable(capability gles.Enum)` | Enable implements gles.Functions. |
| `func (webgl *WebGL) EnableVertexAttribArray(index uint32)` | EnableVertexAttribArray implements gles.Functions. |
| `func (webgl *WebGL) FramebufferTexture2D(target, attachment, textureTarget gles.Enum, texture gles.Texture, level int)` | FramebufferTexture2D implements gles.Functions. |
| `func (webgl *WebGL) LinkProgram(program gles.Program)` | LinkProgram implements gles.Functions. |
| `func (webgl *WebGL) PixelStorei(name gles.Enum, value int32)` | PixelStorei implements gles.Functions. |
| `func (webgl *WebGL) ProgramLinked(program gles.Program) (bool, string)` | ProgramLinked implements gles.Functions. |
| `func (webgl *WebGL) ShaderCompiled(shader gles.Shader) (bool, string)` | ShaderCompiled implements gles.Functions. |
| `func (webgl *WebGL) ShaderSource(shader gles.Shader, source string)` | ShaderSource implements gles.Functions. |
| `func (webgl *WebGL) TexImage2D(target gles.Enum, level int, internalFormat gles.Enum, width, height int, format, kind gles.Enum, pixels []byte)` | TexImage2D implements gles.Functions. |
| `func (webgl *WebGL) TexParameteri(target, name gles.Enum, value int32)` | TexParameteri implements gles.Functions. |
| `func (webgl *WebGL) TexSubImage2D(target gles.Enum, level, x, y, width, height int, format, kind gles.Enum, pixels []byte)` | TexSubImage2D implements gles.Functions. |
| `func (webgl *WebGL) Uniform1f(location gles.Uniform, value float32)` | Uniform1f implements gles.Functions. |
| `func (webgl *WebGL) Uniform1i(location gles.Uniform, value int32)` | Uniform1i implements gles.Functions. |
| `func (webgl *WebGL) Uniform2f(location gles.Uniform, first, second float32)` | Uniform2f implements gles.Functions. |
| `func (webgl *WebGL) Uniform3f(location gles.Uniform, first, second, third float32)` | Uniform3f implements gles.Functions. |
| `func (webgl *WebGL) Uniform4f(location gles.Uniform, first, second, third, fourth float32)` | Uniform4f implements gles.Functions. |
| `func (webgl *WebGL) UniformLocation(program gles.Program, name string) gles.Uniform` | UniformLocation implements gles.Functions. |
| `func (webgl *WebGL) UniformMatrix3(location gles.Uniform, matrix [9]float32)` | UniformMatrix3 implements gles.Functions. |
| `func (webgl *WebGL) UniformMatrix4(location gles.Uniform, matrix [16]float32)` | UniformMatrix4 implements gles.Functions. |
| `func (webgl *WebGL) UseProgram(program gles.Program)` | UseProgram implements gles.Functions. |
| `func (webgl *WebGL) VertexAttribPointer(index uint32, size int, kind gles.Enum, normalized bool, stride, offset int)` | VertexAttribPointer implements gles.Functions. |
| `func (webgl *WebGL) Viewport(x, y, width, height int)` | Viewport implements gles.Functions. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
