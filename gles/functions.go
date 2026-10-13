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

	One              Enum = 1
	SourceAlpha      Enum = 0x0302
	OneMinusSrcAlpha Enum = 0x0303
	LessOrEqual      Enum = 0x0203

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
	Texture0         Enum = 0x84C0
	TextureMinFilter Enum = 0x2801
	TextureMagFilter Enum = 0x2800
	TextureWrapS     Enum = 0x2802
	TextureWrapT     Enum = 0x2803
	Linear           Enum = 0x2601
	ClampToEdge      Enum = 0x812F
	Luminance        Enum = 0x1909
	UnpackAlignment  Enum = 0x0CF5
)

// Handles name objects the graphics driver keeps. 0 is never a valid one.
type (
	Shader  uint32
	Program uint32
	Buffer  uint32
	Texture uint32
)

// Uniform is where a program reads a uniform value; -1 when the program has none of that name
// (the compiler drops uniforms it does not use).
type Uniform int32

// Functions is the part of OpenGL ES 2.0 the renderers need. An implementation must be used from
// the goroutine (and thread) that owns the graphics context.
type Functions interface {
	CreateShader(stage Enum) Shader
	ShaderSource(shader Shader, source string)
	CompileShader(shader Shader)
	// ShaderCompiled reports whether the last compilation worked, and the compiler's log.
	ShaderCompiled(shader Shader) (bool, string)
	DeleteShader(shader Shader)

	CreateProgram() Program
	AttachShader(program Program, shader Shader)
	LinkProgram(program Program)
	// ProgramLinked reports whether the last link worked, and the linker's log.
	ProgramLinked(program Program) (bool, string)
	// BindAttribLocation gives an attribute its index; it takes effect at the next link.
	BindAttribLocation(program Program, index uint32, name string)
	UseProgram(program Program)
	UniformLocation(program Program, name string) Uniform

	Uniform1i(location Uniform, value int32)
	Uniform1f(location Uniform, value float32)
	Uniform2f(location Uniform, first, second float32)
	Uniform3f(location Uniform, first, second, third float32)
	Uniform4f(location Uniform, first, second, third, fourth float32)
	UniformMatrix3(location Uniform, matrix [9]float32)
	UniformMatrix4(location Uniform, matrix [16]float32)

	CreateBuffer() Buffer
	BindBuffer(target Enum, buffer Buffer)
	// BufferData replaces a buffer's contents with these bytes.
	BufferData(target Enum, data []byte, usage Enum)

	EnableVertexAttribArray(index uint32)
	// VertexAttribPointer describes attribute index in the bound array buffer: size values of
	// type kind each, stride bytes apart, the first offset bytes in.
	VertexAttribPointer(index uint32, size int, kind Enum, normalized bool, stride, offset int)

	CreateTexture() Texture
	ActiveTexture(unit Enum)
	BindTexture(target Enum, texture Texture)
	TexParameteri(target, name Enum, value int32)
	PixelStorei(name Enum, value int32)
	TexImage2D(target Enum, level int, internalFormat Enum, width, height int, format, kind Enum, pixels []byte)

	Viewport(x, y, width, height int)
	ClearColor(red, green, blue, alpha float32)
	Clear(mask Enum)
	Enable(capability Enum)
	Disable(capability Enum)
	BlendFuncSeparate(sourceColor, destinationColor, sourceAlpha, destinationAlpha Enum)
	DepthMask(write bool)
	DepthFunc(function Enum)

	DrawArrays(mode Enum, first, count int)
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
