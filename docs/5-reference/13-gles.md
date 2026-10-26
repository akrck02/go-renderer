# gles

```go
import "github.com/akrck02/go-renderer/gles"
```

Package gles draws go-renderer's flat interfaces and a simple 3D scene through
OpenGL ES, the graphics interface both browsers (as WebGL2) and Android phones
offer. It makes no graphics calls of its own: it talks to a small interface,
Functions, that each platform implements (the browser package through
syscall/js, an Android window through golang.org/x/mobile/gl). So the renderers
are written once, in plain Go with no cgo, and tested on any computer with a
fake.

It uses only what OpenGL ES 2.0 has, plus 32-bit indices (an extension every
current phone has): golang.org/x/mobile/gl knows nothing newer on Android,
and WebGL2 accepts it all. So the shaders are GLSL ES 1.00, there are no vertex
array objects (each draw describes its vertices) and the atlas is a LUMINANCE
texture.

Desktop programs keep using the opengl and metal packages; nothing here replaces
them.

### Constants and variables

```go
var (
	// ShaderNotCompiled is a shader the driver would not compile.
	ShaderNotCompiled = failure.Define("gles.shader-not-compiled", "the driver could not compile a shader")
	// ProgramNotLinked is a pair of shaders the driver would not link into a program.
	ProgramNotLinked = failure.Define("gles.program-not-linked", "the driver could not link a program")
)
```

Codes of the errors the renderers return when they are made.

### Buffer

Handles name objects the graphics driver keeps. 0 is never a valid one.

```go
type Buffer uint32
```

### Enum

Enum is an OpenGL ES constant. The values are the ones the OpenGL ES 3.0 and WebGL2 specifications give, so every implementation passes them through unchanged.

```go
type Enum uint32
```

```go
const (
	Triangles Enum = 0x0004

	ColorBufferBit Enum = 0x00004000
	DepthBufferBit Enum = 0x00000100

	Blend     Enum = 0x0BE2
	DepthTest Enum = 0x0B71

	Zero                Enum = 0
	One                 Enum = 1
	SourceColor         Enum = 0x0300
	DestinationColor    Enum = 0x0306
	SourceAlpha         Enum = 0x0302
	OneMinusSourceAlpha Enum = 0x0303
	LessOrEqual         Enum = 0x0203

	ArrayBuffer        Enum = 0x8892
	ElementArrayBuffer Enum = 0x8893
	StaticDraw         Enum = 0x88E4
	StreamDraw         Enum = 0x88E0

	Float          Enum = 0x1406
	UnsignedByte   Enum = 0x1401
	UnsignedInt    Enum = 0x1405
	VertexShader   Enum = 0x8B31
	FragmentShader Enum = 0x8B30

	Texture2D        Enum = 0x0DE1
	TextureUnit0     Enum = 0x84C0
	TextureMinFilter Enum = 0x2801
	TextureMagFilter Enum = 0x2800
	TextureWrapS     Enum = 0x2802
	TextureWrapT     Enum = 0x2803
	Linear           Enum = 0x2601
	ClampToEdge      Enum = 0x812F
	Luminance        Enum = 0x1909
	RGB              Enum = 0x1907
	RGBA             Enum = 0x1908
	UnpackAlignment  Enum = 0x0CF5
	Nearest          Enum = 0x2600

	FramebufferTarget Enum = 0x8D40
	ColorAttachment0  Enum = 0x8CE0
)
```

The constants the renderers use.

### FlatRenderer

FlatRenderer draws a 2D draw list on a plain background, with no scene, sky or camera: the window of an application that is only an interface. It does what opengl.FlatRenderer does on the desktop, with the same atlas and one draw call per frame. Create and use it where the graphics context lives.

```go
type FlatRenderer struct {
	// Target is the framebuffer Draw draws into: 0, the screen, unless the caller made one to
	// draw a picture offscreen.
	Target Framebuffer
	// LinearOutput makes the colours, which are the screen's (sRGB), be written linear: for a
	// target that encodes sRGB itself, as a headset's panel does, where they would otherwise come
	// out twice as light.
	LinearOutput bool

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewFlatRenderer(functions Functions, fonts overlay.Fonts) (*FlatRenderer, error)` | NewFlatRenderer compiles the shaders and prepares the atlas; text is drawn in fonts (overlay.Fonts{} for the default ones). |
| `func (renderer *FlatRenderer) Draw(list *draw2d.List, background draw2d.Color, width, height int, scale float32)` | Draw clears the target framebuffer (width × height pixels) to the background colour and paints the list. |

### Framebuffer

Framebuffer 0 is the screen's own: it can be bound, though no call made it.

```go
type Framebuffer uint32
```

### Functions

Functions is the part of OpenGL ES 2.0 the renderers need. An implementation must be used from the goroutine (and thread) that owns the graphics context. Its methods are named in words (SetUniformInteger, SetTextureImage), so a reader new to OpenGL can follow the renderers; each comment gives the OpenGL ES function it stands for (glUniform1i, glTexImage2D), for a reader who knows OpenGL or looks it up.

```go
type Functions interface {
	// CreateShader is glCreateShader.
	CreateShader(stage Enum) Shader
	// SetShaderSource is glShaderSource.
	SetShaderSource(shader Shader, source string)
	// CompileShader is glCompileShader.
	CompileShader(shader Shader)
	// ShaderCompiled reports whether the last compilation worked, and the compiler's log.
	// (OpenGL ES: glGetShaderiv(GL_COMPILE_STATUS) and glGetShaderInfoLog.)
	ShaderCompiled(shader Shader) (bool, string)
	// DeleteShader is glDeleteShader.
	DeleteShader(shader Shader)

	// CreateProgram is glCreateProgram.
	CreateProgram() Program
	// AttachShader is glAttachShader.
	AttachShader(program Program, shader Shader)
	// LinkProgram is glLinkProgram.
	LinkProgram(program Program)
	// ProgramLinked reports whether the last link worked, and the linker's log.
	// (OpenGL ES: glGetProgramiv(GL_LINK_STATUS) and glGetProgramInfoLog.)
	ProgramLinked(program Program) (bool, string)
	// BindAttributeLocation gives an attribute its index; it takes effect at the next link.
	// (OpenGL ES: glBindAttribLocation.)
	BindAttributeLocation(program Program, index uint32, name string)
	// UseProgram is glUseProgram.
	UseProgram(program Program)
	// UniformLocation is glGetUniformLocation.
	UniformLocation(program Program, name string) Uniform

	// SetUniformInteger is glUniform1i.
	SetUniformInteger(location Uniform, value int32)
	// SetUniformFloat is glUniform1f.
	SetUniformFloat(location Uniform, value float32)
	// SetUniformTwoFloats is glUniform2f.
	SetUniformTwoFloats(location Uniform, first, second float32)
	// SetUniformThreeFloats is glUniform3f.
	SetUniformThreeFloats(location Uniform, first, second, third float32)
	// SetUniformFourFloats is glUniform4f.
	SetUniformFourFloats(location Uniform, first, second, third, fourth float32)
	// SetUniformMatrix3By3 is glUniformMatrix3fv.
	SetUniformMatrix3By3(location Uniform, matrix [9]float32)
	// SetUniformMatrix4By4 is glUniformMatrix4fv.
	SetUniformMatrix4By4(location Uniform, matrix [16]float32)

	// CreateBuffer is glGenBuffers.
	CreateBuffer() Buffer
	// BindBuffer is glBindBuffer.
	BindBuffer(target Enum, buffer Buffer)
	// SetBufferData replaces a buffer's contents with these bytes.
	// (OpenGL ES: glBufferData.)
	SetBufferData(target Enum, data []byte, usage Enum)

	// EnableVertexAttribute is glEnableVertexAttribArray.
	EnableVertexAttribute(index uint32)
	// DisableVertexAttribute is glDisableVertexAttribArray.
	DisableVertexAttribute(index uint32)
	// DescribeVertexAttribute describes attribute index in the bound array buffer: size values of
	// type kind each, stride bytes apart, the first offset bytes in.
	// (OpenGL ES: glVertexAttribPointer.)
	DescribeVertexAttribute(index uint32, size int, kind Enum, normalized bool, stride, offset int)

	// CreateTexture is glGenTextures.
	CreateTexture() Texture
	// SelectTextureUnit is glActiveTexture.
	SelectTextureUnit(unit Enum)
	// BindTexture is glBindTexture.
	BindTexture(target Enum, texture Texture)
	// SetTextureParameter is glTexParameteri.
	SetTextureParameter(target, name Enum, value int32)
	// SetPixelStorage is glPixelStorei.
	SetPixelStorage(name Enum, value int32)
	// SetTextureImage is glTexImage2D.
	SetTextureImage(target Enum, level int, internalFormat Enum, width, height int, format, kind Enum, pixels []byte)
	// ReplaceTexturePart replaces a rectangle of the bound texture's pixels, given row after row.
	// (OpenGL ES: glTexSubImage2D.)
	ReplaceTexturePart(target Enum, level, x, y, width, height int, format, kind Enum, pixels []byte)
	// CopyFramebufferToTexture copies a rectangle of the bound framebuffer (x, y from its bottom left)
	// into the bound texture at xOffset, yOffset.
	// (OpenGL ES: glCopyTexSubImage2D.)
	CopyFramebufferToTexture(target Enum, level, xOffset, yOffset, x, y, width, height int)
	// DeleteTexture is glDeleteTextures.
	DeleteTexture(texture Texture)

	// Framebuffers let a pass draw into a texture: the backdrop's blur. SetTextureImage with no pixels
	// makes the texture they draw into.
	// (OpenGL ES: glGenFramebuffers.)
	CreateFramebuffer() Framebuffer
	// BindFramebuffer is glBindFramebuffer.
	BindFramebuffer(target Enum, framebuffer Framebuffer)
	// AttachTextureToFramebuffer is glFramebufferTexture2D.
	AttachTextureToFramebuffer(target, attachment, textureTarget Enum, texture Texture, level int)
	// DeleteFramebuffer is glDeleteFramebuffers.
	DeleteFramebuffer(framebuffer Framebuffer)

	// SetViewport is glViewport.
	SetViewport(x, y, width, height int)
	// SetClearColor is glClearColor.
	SetClearColor(red, green, blue, alpha float32)
	// Clear is glClear.
	Clear(mask Enum)
	// Enable is glEnable.
	Enable(capability Enum)
	// Disable is glDisable.
	Disable(capability Enum)
	// SetBlendFactors is glBlendFuncSeparate.
	SetBlendFactors(sourceColor, destinationColor, sourceAlpha, destinationAlpha Enum)
	// SetDepthWrite is glDepthMask.
	SetDepthWrite(write bool)
	// SetDepthFunction is glDepthFunc.
	SetDepthFunction(function Enum)

	// DrawArrays is glDrawArrays.
	DrawArrays(mode Enum, first, count int)
	// DrawElements is glDrawElements.
	DrawElements(mode Enum, count int, kind Enum, offset int)
}
```

### Program

Handles name objects the graphics driver keeps. 0 is never a valid one.

```go
type Program uint32
```

### SceneRenderer

SceneRenderer draws a scene.Scene in its simplest form: lit meshes and instances, the light of the time of day, fog, translucent water and, over them, the overlay (labels, header, notices and the program's draw list). What a frame draws (culling, levels of detail, order) is planned by sceneplan, like on the desktop. It is a first step, not the desktop renderer: there are no shadows, sky dome, procedural patterns, waves, wind or snow. OpenGL ES 2 cannot draw instances, so each batch of instances is made into one mesh on the processor, again only when it changes.

```go
type SceneRenderer struct {
	// OverlayScale is framebuffer pixels per layout pixel, so labels and interfaces keep their
	// size on a high-density screen (0 = 1).
	OverlayScale float32
	// Target is the framebuffer Draw draws into: 0, the screen, unless the caller made one to
	// draw a picture offscreen.
	Target Framebuffer
	// LinearOutput makes the shaders and the clear write linear colour instead of turning it to
	// the screen's gamma: for a target that encodes sRGB itself (a headset's swapchain), which
	// would otherwise apply the gamma twice and wash the picture out.
	LinearOutput bool

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewSceneRenderer(functions Functions, fonts overlay.Fonts) (*SceneRenderer, error)` | NewSceneRenderer compiles the shaders and prepares the overlay's atlas; the overlay's text is drawn in fonts (overlay.Fonts{} for the default ones). |
| `func (renderer *SceneRenderer) Draw(world *scene.Scene, camera camera.Lens, width, height int)` | Draw draws a frame of the scene seen by the camera, in the target framebuffer of width × height pixels. |
| `func (renderer *SceneRenderer) DrawViews(world *scene.Scene, covering camera.Lens, aspect float32, views []View)` | DrawViews plans the frame once, from covering (see camera.CoveringLens) and its aspect (width over height), and draws it in every view with the view's own matrices and eye. |

### Shader

Handles name objects the graphics driver keeps. 0 is never a valid one.

```go
type Shader uint32
```

### Texture

Handles name objects the graphics driver keeps. 0 is never a valid one.

```go
type Texture uint32
```

### Uniform

Uniform is where a program reads a uniform value; -1 when the program has none of that name (the compiler drops uniforms it does not use).

```go
type Uniform int32
```

### View

View is one picture DrawViews draws: an eye of a headset.

```go
type View struct {
	// Target is the framebuffer it is drawn into, with a depth buffer. Views may share one, each
	// in its own viewport.
	Target Framebuffer
	// X, Y, Width and Height are its viewport in the target, in pixels from the bottom-left.
	X, Y, Width, Height int
	// View and Projection are the eye's matrices (see camera.Rig and
	// graphics.PerspectiveFromFieldOfView).
	View, Projection graphics.Matrix
	// Eye is the eye in the world: lighting, fog and the water's reflection are worked out from it.
	Eye graphics.Vector
}
```

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
