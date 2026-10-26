//go:build android

package android

/*
#include <GLES2/gl2.h>
#include <stdlib.h>

// shader_source sets one shader's source from a string of a known length.
static void shader_source(GLuint shader, const char *text, GLint length) {
	glShaderSource(shader, 1, &text, &length);
}

// info_log reads a shader's or a program's log into a buffer.
static void shader_log(GLuint shader, char *buffer, GLsizei size) { glGetShaderInfoLog(shader, size, NULL, buffer); }
static void program_log(GLuint program, char *buffer, GLsizei size) { glGetProgramInfoLog(program, size, NULL, buffer); }

static GLuint one_buffer(void) { GLuint name = 0; glGenBuffers(1, &name); return name; }
static GLuint one_texture(void) { GLuint name = 0; glGenTextures(1, &name); return name; }
static GLuint one_framebuffer(void) { GLuint name = 0; glGenFramebuffers(1, &name); return name; }
static void delete_texture(GLuint name) { glDeleteTextures(1, &name); }
static void delete_framebuffer(GLuint name) { glDeleteFramebuffers(1, &name); }
static GLint shader_status(GLuint shader, GLenum which) { GLint value = 0; glGetShaderiv(shader, which, &value); return value; }
static GLint program_status(GLuint program, GLenum which) { GLint value = 0; glGetProgramiv(program, which, &value); return value; }

// OpenGL ES takes offsets into the bound buffer as pointers: C makes them, not Go.
static void attribute_layout(GLuint index, GLint size, GLenum kind, GLboolean normalized, GLsizei stride, GLintptr offset) {
	glVertexAttribPointer(index, size, kind, normalized, stride, (const void *)offset);
}
static void draw_elements(GLenum mode, GLsizei count, GLenum kind, GLintptr offset) {
	glDrawElements(mode, count, kind, (const void *)offset);
}
*/
import "C"

import (
	"unsafe"

	"github.com/akrck02/go-renderer/gles"
)

// GL is OpenGL ES 2, called directly on the drawing thread, for go-renderer's gles renderers.
// Use it only from that thread, inside the drawing function Run calls.
type GL struct{}

// CreateShader implements gles.Functions.
func (GL) CreateShader(stage gles.Enum) gles.Shader {
	return gles.Shader(C.glCreateShader(C.GLenum(stage)))
}

// SetShaderSource implements gles.Functions.
func (GL) SetShaderSource(shader gles.Shader, source string) {
	text := C.CString(source)
	defer C.free(unsafe.Pointer(text))
	C.shader_source(C.GLuint(shader), text, C.GLint(len(source)))
}

// CompileShader implements gles.Functions.
func (GL) CompileShader(shader gles.Shader) { C.glCompileShader(C.GLuint(shader)) }

// ShaderCompiled implements gles.Functions.
func (GL) ShaderCompiled(shader gles.Shader) (bool, string) {
	if C.shader_status(C.GLuint(shader), C.GL_COMPILE_STATUS) != 0 {
		return true, ""
	}
	size := C.shader_status(C.GLuint(shader), C.GL_INFO_LOG_LENGTH)
	return false, readLog(int(size), func(buffer *C.char) { C.shader_log(C.GLuint(shader), buffer, C.GLsizei(size)) })
}

// DeleteShader implements gles.Functions.
func (GL) DeleteShader(shader gles.Shader) { C.glDeleteShader(C.GLuint(shader)) }

// CreateProgram implements gles.Functions.
func (GL) CreateProgram() gles.Program { return gles.Program(C.glCreateProgram()) }

// AttachShader implements gles.Functions.
func (GL) AttachShader(program gles.Program, shader gles.Shader) {
	C.glAttachShader(C.GLuint(program), C.GLuint(shader))
}

// LinkProgram implements gles.Functions.
func (GL) LinkProgram(program gles.Program) { C.glLinkProgram(C.GLuint(program)) }

// ProgramLinked implements gles.Functions.
func (GL) ProgramLinked(program gles.Program) (bool, string) {
	if C.program_status(C.GLuint(program), C.GL_LINK_STATUS) != 0 {
		return true, ""
	}
	size := C.program_status(C.GLuint(program), C.GL_INFO_LOG_LENGTH)
	return false, readLog(int(size), func(buffer *C.char) { C.program_log(C.GLuint(program), buffer, C.GLsizei(size)) })
}

// readLog reads a log of a given size through a function that fills a C buffer.
func readLog(size int, read func(buffer *C.char)) string {
	if size <= 0 {
		return "(no log)"
	}
	buffer := (*C.char)(C.malloc(C.size_t(size)))
	defer C.free(unsafe.Pointer(buffer))
	read(buffer)
	return C.GoString(buffer)
}

// BindAttributeLocation implements gles.Functions.
func (GL) BindAttributeLocation(program gles.Program, index uint32, name string) {
	text := C.CString(name)
	defer C.free(unsafe.Pointer(text))
	C.glBindAttribLocation(C.GLuint(program), C.GLuint(index), text)
}

// UseProgram implements gles.Functions.
func (GL) UseProgram(program gles.Program) { C.glUseProgram(C.GLuint(program)) }

// UniformLocation implements gles.Functions.
func (GL) UniformLocation(program gles.Program, name string) gles.Uniform {
	text := C.CString(name)
	defer C.free(unsafe.Pointer(text))
	return gles.Uniform(C.glGetUniformLocation(C.GLuint(program), text))
}

// SetUniformInteger implements gles.Functions.
func (GL) SetUniformInteger(location gles.Uniform, value int32) {
	C.glUniform1i(C.GLint(location), C.GLint(value))
}

// SetUniformFloat implements gles.Functions.
func (GL) SetUniformFloat(location gles.Uniform, value float32) {
	C.glUniform1f(C.GLint(location), C.GLfloat(value))
}

// SetUniformTwoFloats implements gles.Functions.
func (GL) SetUniformTwoFloats(location gles.Uniform, first, second float32) {
	C.glUniform2f(C.GLint(location), C.GLfloat(first), C.GLfloat(second))
}

// SetUniformThreeFloats implements gles.Functions.
func (GL) SetUniformThreeFloats(location gles.Uniform, first, second, third float32) {
	C.glUniform3f(C.GLint(location), C.GLfloat(first), C.GLfloat(second), C.GLfloat(third))
}

// SetUniformFourFloats implements gles.Functions.
func (GL) SetUniformFourFloats(location gles.Uniform, first, second, third, fourth float32) {
	C.glUniform4f(C.GLint(location), C.GLfloat(first), C.GLfloat(second), C.GLfloat(third), C.GLfloat(fourth))
}

// SetUniformMatrix3By3 implements gles.Functions.
func (GL) SetUniformMatrix3By3(location gles.Uniform, matrix [9]float32) {
	C.glUniformMatrix3fv(C.GLint(location), 1, C.GL_FALSE, (*C.GLfloat)(unsafe.Pointer(&matrix[0])))
}

// SetUniformMatrix4By4 implements gles.Functions.
func (GL) SetUniformMatrix4By4(location gles.Uniform, matrix [16]float32) {
	C.glUniformMatrix4fv(C.GLint(location), 1, C.GL_FALSE, (*C.GLfloat)(unsafe.Pointer(&matrix[0])))
}

// CreateBuffer implements gles.Functions.
func (GL) CreateBuffer() gles.Buffer { return gles.Buffer(C.one_buffer()) }

// BindBuffer implements gles.Functions.
func (GL) BindBuffer(target gles.Enum, buffer gles.Buffer) {
	C.glBindBuffer(C.GLenum(target), C.GLuint(buffer))
}

// SetBufferData implements gles.Functions.
func (GL) SetBufferData(target gles.Enum, data []byte, usage gles.Enum) {
	C.glBufferData(C.GLenum(target), C.GLsizeiptr(len(data)), bytesOf(data), C.GLenum(usage))
}

// bytesOf points C at a slice's bytes, for the length of a call; nil for an empty slice.
func bytesOf(data []byte) unsafe.Pointer {
	if len(data) == 0 {
		return nil
	}
	return unsafe.Pointer(&data[0])
}

// EnableVertexAttribute implements gles.Functions.
func (GL) EnableVertexAttribute(index uint32) { C.glEnableVertexAttribArray(C.GLuint(index)) }

// DisableVertexAttribute implements gles.Functions.
func (GL) DisableVertexAttribute(index uint32) { C.glDisableVertexAttribArray(C.GLuint(index)) }

// DescribeVertexAttribute implements gles.Functions.
func (GL) DescribeVertexAttribute(index uint32, size int, kind gles.Enum, normalized bool, stride, offset int) {
	flag := C.GLboolean(C.GL_FALSE)
	if normalized {
		flag = C.GL_TRUE
	}
	C.attribute_layout(C.GLuint(index), C.GLint(size), C.GLenum(kind), flag, C.GLsizei(stride), C.GLintptr(offset))
}

// CreateTexture implements gles.Functions.
func (GL) CreateTexture() gles.Texture { return gles.Texture(C.one_texture()) }

// SelectTextureUnit implements gles.Functions.
func (GL) SelectTextureUnit(unit gles.Enum) { C.glActiveTexture(C.GLenum(unit)) }

// BindTexture implements gles.Functions.
func (GL) BindTexture(target gles.Enum, texture gles.Texture) {
	C.glBindTexture(C.GLenum(target), C.GLuint(texture))
}

// SetTextureParameter implements gles.Functions.
func (GL) SetTextureParameter(target, name gles.Enum, value int32) {
	C.glTexParameteri(C.GLenum(target), C.GLenum(name), C.GLint(value))
}

// SetPixelStorage implements gles.Functions.
func (GL) SetPixelStorage(name gles.Enum, value int32) {
	C.glPixelStorei(C.GLenum(name), C.GLint(value))
}

// GenerateMipmap implements gles.Functions.
func (GL) GenerateMipmap(target gles.Enum) {
	C.glGenerateMipmap(C.GLenum(target))
}

// SetTextureImage implements gles.Functions.
func (GL) SetTextureImage(target gles.Enum, level int, internalFormat gles.Enum, width, height int, format, kind gles.Enum, pixels []byte) {
	C.glTexImage2D(C.GLenum(target), C.GLint(level), C.GLint(internalFormat), C.GLsizei(width), C.GLsizei(height), 0,
		C.GLenum(format), C.GLenum(kind), bytesOf(pixels))
}

// CopyFramebufferToTexture implements gles.Functions.
func (GL) CopyFramebufferToTexture(target gles.Enum, level, xOffset, yOffset, x, y, width, height int) {
	C.glCopyTexSubImage2D(C.GLenum(target), C.GLint(level), C.GLint(xOffset), C.GLint(yOffset), C.GLint(x), C.GLint(y),
		C.GLsizei(width), C.GLsizei(height))
}

// DeleteTexture implements gles.Functions.
func (GL) DeleteTexture(texture gles.Texture) { C.delete_texture(C.GLuint(texture)) }

// CreateFramebuffer implements gles.Functions.
func (GL) CreateFramebuffer() gles.Framebuffer { return gles.Framebuffer(C.one_framebuffer()) }

// BindFramebuffer implements gles.Functions.
func (GL) BindFramebuffer(target gles.Enum, framebuffer gles.Framebuffer) {
	C.glBindFramebuffer(C.GLenum(target), C.GLuint(framebuffer))
}

// AttachTextureToFramebuffer implements gles.Functions.
func (GL) AttachTextureToFramebuffer(target, attachment, textureTarget gles.Enum, texture gles.Texture, level int) {
	C.glFramebufferTexture2D(C.GLenum(target), C.GLenum(attachment), C.GLenum(textureTarget), C.GLuint(texture), C.GLint(level))
}

// DeleteFramebuffer implements gles.Functions.
func (GL) DeleteFramebuffer(framebuffer gles.Framebuffer) {
	C.delete_framebuffer(C.GLuint(framebuffer))
}

// ReplaceTexturePart implements gles.Functions.
func (GL) ReplaceTexturePart(target gles.Enum, level, x, y, width, height int, format, kind gles.Enum, pixels []byte) {
	C.glTexSubImage2D(C.GLenum(target), C.GLint(level), C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height),
		C.GLenum(format), C.GLenum(kind), bytesOf(pixels))
}

// SetViewport implements gles.Functions.
func (GL) SetViewport(x, y, width, height int) {
	C.glViewport(C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height))
}

// SetClearColor implements gles.Functions.
func (GL) SetClearColor(red, green, blue, alpha float32) {
	C.glClearColor(C.GLfloat(red), C.GLfloat(green), C.GLfloat(blue), C.GLfloat(alpha))
}

// Clear implements gles.Functions.
func (GL) Clear(mask gles.Enum) { C.glClear(C.GLbitfield(mask)) }

// Enable implements gles.Functions.
func (GL) Enable(capability gles.Enum) { C.glEnable(C.GLenum(capability)) }

// Disable implements gles.Functions.
func (GL) Disable(capability gles.Enum) { C.glDisable(C.GLenum(capability)) }

// SetBlendFactors implements gles.Functions.
func (GL) SetBlendFactors(sourceColor, destinationColor, sourceAlpha, destinationAlpha gles.Enum) {
	C.glBlendFuncSeparate(C.GLenum(sourceColor), C.GLenum(destinationColor), C.GLenum(sourceAlpha), C.GLenum(destinationAlpha))
}

// SetDepthWrite implements gles.Functions.
func (GL) SetDepthWrite(write bool) {
	flag := C.GLboolean(C.GL_FALSE)
	if write {
		flag = C.GL_TRUE
	}
	C.glDepthMask(flag)
}

// SetDepthFunction implements gles.Functions.
func (GL) SetDepthFunction(function gles.Enum) { C.glDepthFunc(C.GLenum(function)) }

// DrawArrays implements gles.Functions.
func (GL) DrawArrays(mode gles.Enum, first, count int) {
	C.glDrawArrays(C.GLenum(mode), C.GLint(first), C.GLsizei(count))
}

// DrawElements implements gles.Functions.
func (GL) DrawElements(mode gles.Enum, count int, kind gles.Enum, offset int) {
	C.draw_elements(C.GLenum(mode), C.GLsizei(count), C.GLenum(kind), C.GLintptr(offset))
}

// description names the OpenGL ES version and the GPU, as the driver gives them.
func description() string {
	return C.GoString((*C.char)(unsafe.Pointer(C.glGetString(C.GL_VERSION)))) + " · " +
		C.GoString((*C.char)(unsafe.Pointer(C.glGetString(C.GL_RENDERER))))
}
