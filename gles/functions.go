// Package gles draws go-renderer's flat interfaces and a simple 3D scene through OpenGL ES, the
// graphics interface both browsers (as WebGL2) and Android phones offer. It makes no graphics
// calls of its own: it talks to a small interface, Functions, that each platform implements (the
// browser package through syscall/js, an Android window through golang.org/x/mobile/gl). So the
// renderers are written once, in plain Go with no cgo, and tested on any computer with a fake.
//
// It uses only what OpenGL ES 2.0 has, plus 32-bit indices (an extension every current phone
// has): golang.org/x/mobile/gl knows nothing newer on Android, and WebGL2 accepts it all. So the
// shaders are GLSL ES 1.00, there are no vertex array objects (each draw describes its vertices)
// and the atlas is a LUMINANCE texture.
//
// Desktop programs keep using the opengl and metal packages; nothing here replaces them.
package gles

import "unsafe"

// Enum is an OpenGL ES constant. The values are the ones the OpenGL ES 3.0 and WebGL2
// specifications give, so every implementation passes them through unchanged.
type Enum uint32

// The constants the renderers use.
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

// Handles name objects the graphics driver keeps. 0 is never a valid one.
type (
	Shader  uint32
	Program uint32
	Buffer  uint32
	Texture uint32
	// Framebuffer 0 is the screen's own: it can be bound, though no call made it.
	Framebuffer uint32
)

// Uniform is where a program reads a uniform value; -1 when the program has none of that name
// (the compiler drops uniforms it does not use).
type Uniform int32

// Functions is the part of OpenGL ES 2.0 the renderers need. An implementation must be used from
// the goroutine (and thread) that owns the graphics context. Its methods are named in words
// (SetUniformInteger, SetTextureImage), so a reader new to OpenGL can follow the renderers; each
// comment gives the OpenGL ES function it stands for (glUniform1i, glTexImage2D), for a reader
// who knows OpenGL or looks it up.
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

// floatBytes views float32 values as the bytes the GPU reads, without copying. Every platform this
// package runs on (WebAssembly, arm64, amd64) is little-endian, like the GPUs.
func floatBytes(values []float32) []byte {
	if len(values) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&values[0])), len(values)*4)
}

// indexBytes views uint32 indices as bytes, without copying.
func indexBytes(values []uint32) []byte {
	if len(values) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&values[0])), len(values)*4)
}
