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
| `func Run(canvas *Canvas, app *window.Application, drawFrame func() error) error` | Run drives an application in the canvas until it sets Stop: update steps at the fixed step and drawFrame to draw one, called from requestAnimationFrame. |
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
| `func (webgl *WebGL) AttachShader(program gles.Program, shader gles.Shader)` | AttachShader implements gles.Functions. |
| `func (webgl *WebGL) AttachTextureToFramebuffer(target, attachment, textureTarget gles.Enum, texture gles.Texture, level int)` | AttachTextureToFramebuffer implements gles.Functions. |
| `func (webgl *WebGL) BindAttributeLocation(program gles.Program, index uint32, name string)` | BindAttributeLocation implements gles.Functions. |
| `func (webgl *WebGL) BindBuffer(target gles.Enum, buffer gles.Buffer)` | BindBuffer implements gles.Functions. |
| `func (webgl *WebGL) BindFramebuffer(target gles.Enum, framebuffer gles.Framebuffer)` | BindFramebuffer implements gles.Functions. |
| `func (webgl *WebGL) BindTexture(target gles.Enum, texture gles.Texture)` | BindTexture implements gles.Functions. |
| `func (webgl *WebGL) Clear(mask gles.Enum)` | Clear implements gles.Functions. |
| `func (webgl *WebGL) CompileShader(shader gles.Shader)` | CompileShader implements gles.Functions. |
| `func (webgl *WebGL) CopyFramebufferToTexture(target gles.Enum, level, xOffset, yOffset, x, y, width, height int)` | CopyFramebufferToTexture implements gles.Functions. |
| `func (webgl *WebGL) CreateBuffer() gles.Buffer` | CreateBuffer implements gles.Functions. |
| `func (webgl *WebGL) CreateFramebuffer() gles.Framebuffer` | CreateFramebuffer implements gles.Functions. |
| `func (webgl *WebGL) CreateProgram() gles.Program` | CreateProgram implements gles.Functions. |
| `func (webgl *WebGL) CreateShader(stage gles.Enum) gles.Shader` | CreateShader implements gles.Functions. |
| `func (webgl *WebGL) CreateTexture() gles.Texture` | CreateTexture implements gles.Functions. |
| `func (webgl *WebGL) DeleteFramebuffer(framebuffer gles.Framebuffer)` | DeleteFramebuffer implements gles.Functions. |
| `func (webgl *WebGL) DeleteShader(shader gles.Shader)` | DeleteShader implements gles.Functions. |
| `func (webgl *WebGL) DeleteTexture(texture gles.Texture)` | DeleteTexture implements gles.Functions. |
| `func (webgl *WebGL) DescribeVertexAttribute(index uint32, size int, kind gles.Enum, normalized bool, stride, offset int)` | DescribeVertexAttribute implements gles.Functions. |
| `func (webgl *WebGL) Disable(capability gles.Enum)` | Disable implements gles.Functions. |
| `func (webgl *WebGL) DisableVertexAttribute(index uint32)` | DisableVertexAttribute implements gles.Functions. |
| `func (webgl *WebGL) DrawArrays(mode gles.Enum, first, count int)` | DrawArrays implements gles.Functions. |
| `func (webgl *WebGL) DrawElements(mode gles.Enum, count int, kind gles.Enum, offset int)` | DrawElements implements gles.Functions. |
| `func (webgl *WebGL) Enable(capability gles.Enum)` | Enable implements gles.Functions. |
| `func (webgl *WebGL) EnableVertexAttribute(index uint32)` | EnableVertexAttribute implements gles.Functions. |
| `func (webgl *WebGL) GenerateMipmap(target gles.Enum)` | GenerateMipmap implements gles.Functions. |
| `func (webgl *WebGL) LinkProgram(program gles.Program)` | LinkProgram implements gles.Functions. |
| `func (webgl *WebGL) ProgramLinked(program gles.Program) (bool, string)` | ProgramLinked implements gles.Functions. |
| `func (webgl *WebGL) ReplaceTexturePart(target gles.Enum, level, x, y, width, height int, format, kind gles.Enum, pixels []byte)` | ReplaceTexturePart implements gles.Functions. |
| `func (webgl *WebGL) SelectTextureUnit(unit gles.Enum)` | SelectTextureUnit implements gles.Functions. |
| `func (webgl *WebGL) SetBlendFactors(sourceColor, destinationColor, sourceAlpha, destinationAlpha gles.Enum)` | SetBlendFactors implements gles.Functions. |
| `func (webgl *WebGL) SetBufferData(target gles.Enum, data []byte, usage gles.Enum)` | SetBufferData implements gles.Functions. |
| `func (webgl *WebGL) SetClearColor(red, green, blue, alpha float32)` | SetClearColor implements gles.Functions. |
| `func (webgl *WebGL) SetDepthFunction(function gles.Enum)` | SetDepthFunction implements gles.Functions. |
| `func (webgl *WebGL) SetDepthWrite(write bool)` | SetDepthWrite implements gles.Functions. |
| `func (webgl *WebGL) SetPixelStorage(name gles.Enum, value int32)` | SetPixelStorage implements gles.Functions. |
| `func (webgl *WebGL) SetShaderSource(shader gles.Shader, source string)` | SetShaderSource implements gles.Functions. |
| `func (webgl *WebGL) SetTextureImage(target gles.Enum, level int, internalFormat gles.Enum, width, height int, format, kind gles.Enum, pixels []byte)` | SetTextureImage implements gles.Functions. |
| `func (webgl *WebGL) SetTextureParameter(target, name gles.Enum, value int32)` | SetTextureParameter implements gles.Functions. |
| `func (webgl *WebGL) SetUniformFloat(location gles.Uniform, value float32)` | SetUniformFloat implements gles.Functions. |
| `func (webgl *WebGL) SetUniformFourFloats(location gles.Uniform, first, second, third, fourth float32)` | SetUniformFourFloats implements gles.Functions. |
| `func (webgl *WebGL) SetUniformInteger(location gles.Uniform, value int32)` | SetUniformInteger implements gles.Functions. |
| `func (webgl *WebGL) SetUniformMatrix3By3(location gles.Uniform, matrix [9]float32)` | SetUniformMatrix3By3 implements gles.Functions. |
| `func (webgl *WebGL) SetUniformMatrix4By4(location gles.Uniform, matrix [16]float32)` | SetUniformMatrix4By4 implements gles.Functions. |
| `func (webgl *WebGL) SetUniformThreeFloats(location gles.Uniform, first, second, third float32)` | SetUniformThreeFloats implements gles.Functions. |
| `func (webgl *WebGL) SetUniformTwoFloats(location gles.Uniform, first, second float32)` | SetUniformTwoFloats implements gles.Functions. |
| `func (webgl *WebGL) SetViewport(x, y, width, height int)` | SetViewport implements gles.Functions. |
| `func (webgl *WebGL) ShaderCompiled(shader gles.Shader) (bool, string)` | ShaderCompiled implements gles.Functions. |
| `func (webgl *WebGL) UniformLocation(program gles.Program, name string) gles.Uniform` | UniformLocation implements gles.Functions. |
| `func (webgl *WebGL) UseProgram(program gles.Program)` | UseProgram implements gles.Functions. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
