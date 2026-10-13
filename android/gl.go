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

func (GL) CreateShader(stage gles.Enum) gles.Shader {
	return gles.Shader(C.glCreateShader(C.GLenum(stage)))
}

func (GL) ShaderSource(shader gles.Shader, source string) {
	text := C.CString(source)
	defer C.free(unsafe.Pointer(text))
	C.shader_source(C.GLuint(shader), text, C.GLint(len(source)))
}

func (GL) CompileShader(shader gles.Shader) { C.glCompileShader(C.GLuint(shader)) }

func (GL) ShaderCompiled(shader gles.Shader) (bool, string) {
	if C.shader_status(C.GLuint(shader), C.GL_COMPILE_STATUS) != 0 {
		return true, ""
	}
	size := C.shader_status(C.GLuint(shader), C.GL_INFO_LOG_LENGTH)
	return false, readLog(int(size), func(buffer *C.char) { C.shader_log(C.GLuint(shader), buffer, C.GLsizei(size)) })
}

func (GL) DeleteShader(shader gles.Shader) { C.glDeleteShader(C.GLuint(shader)) }

func (GL) CreateProgram() gles.Program { return gles.Program(C.glCreateProgram()) }

func (GL) AttachShader(program gles.Program, shader gles.Shader) {
	C.glAttachShader(C.GLuint(program), C.GLuint(shader))
}

func (GL) LinkProgram(program gles.Program) { C.glLinkProgram(C.GLuint(program)) }

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

func (GL) BindAttribLocation(program gles.Program, index uint32, name string) {
	text := C.CString(name)
	defer C.free(unsafe.Pointer(text))
	C.glBindAttribLocation(C.GLuint(program), C.GLuint(index), text)
}

func (GL) UseProgram(program gles.Program) { C.glUseProgram(C.GLuint(program)) }

func (GL) UniformLocation(program gles.Program, name string) gles.Uniform {
	text := C.CString(name)
	defer C.free(unsafe.Pointer(text))
	return gles.Uniform(C.glGetUniformLocation(C.GLuint(program), text))
}

func (GL) Uniform1i(location gles.Uniform, value int32) {
	C.glUniform1i(C.GLint(location), C.GLint(value))
}

func (GL) Uniform1f(location gles.Uniform, value float32) {
	C.glUniform1f(C.GLint(location), C.GLfloat(value))
}

func (GL) Uniform2f(location gles.Uniform, first, second float32) {
	C.glUniform2f(C.GLint(location), C.GLfloat(first), C.GLfloat(second))
}

func (GL) Uniform3f(location gles.Uniform, first, second, third float32) {
	C.glUniform3f(C.GLint(location), C.GLfloat(first), C.GLfloat(second), C.GLfloat(third))
}

func (GL) Uniform4f(location gles.Uniform, first, second, third, fourth float32) {
	C.glUniform4f(C.GLint(location), C.GLfloat(first), C.GLfloat(second), C.GLfloat(third), C.GLfloat(fourth))
}

func (GL) UniformMatrix3(location gles.Uniform, matrix [9]float32) {
	C.glUniformMatrix3fv(C.GLint(location), 1, C.GL_FALSE, (*C.GLfloat)(unsafe.Pointer(&matrix[0])))
}

func (GL) UniformMatrix4(location gles.Uniform, matrix [16]float32) {
	C.glUniformMatrix4fv(C.GLint(location), 1, C.GL_FALSE, (*C.GLfloat)(unsafe.Pointer(&matrix[0])))
}

func (GL) CreateBuffer() gles.Buffer { return gles.Buffer(C.one_buffer()) }

func (GL) BindBuffer(target gles.Enum, buffer gles.Buffer) {
	C.glBindBuffer(C.GLenum(target), C.GLuint(buffer))
}

func (GL) BufferData(target gles.Enum, data []byte, usage gles.Enum) {
	C.glBufferData(C.GLenum(target), C.GLsizeiptr(len(data)), bytesOf(data), C.GLenum(usage))
}

// bytesOf points C at a slice's bytes, for the length of a call; nil for an empty slice.
func bytesOf(data []byte) unsafe.Pointer {
	if len(data) == 0 {
		return nil
	}
	return unsafe.Pointer(&data[0])
}

func (GL) EnableVertexAttribArray(index uint32) { C.glEnableVertexAttribArray(C.GLuint(index)) }

func (GL) VertexAttribPointer(index uint32, size int, kind gles.Enum, normalized bool, stride, offset int) {
	flag := C.GLboolean(C.GL_FALSE)
	if normalized {
		flag = C.GL_TRUE
	}
	C.attribute_layout(C.GLuint(index), C.GLint(size), C.GLenum(kind), flag, C.GLsizei(stride), C.GLintptr(offset))
}

func (GL) CreateTexture() gles.Texture { return gles.Texture(C.one_texture()) }

func (GL) ActiveTexture(unit gles.Enum) { C.glActiveTexture(C.GLenum(unit)) }

func (GL) BindTexture(target gles.Enum, texture gles.Texture) {
	C.glBindTexture(C.GLenum(target), C.GLuint(texture))
}

func (GL) TexParameteri(target, name gles.Enum, value int32) {
	C.glTexParameteri(C.GLenum(target), C.GLenum(name), C.GLint(value))
}

func (GL) PixelStorei(name gles.Enum, value int32) { C.glPixelStorei(C.GLenum(name), C.GLint(value)) }

func (GL) TexImage2D(target gles.Enum, level int, internalFormat gles.Enum, width, height int, format, kind gles.Enum, pixels []byte) {
	C.glTexImage2D(C.GLenum(target), C.GLint(level), C.GLint(internalFormat), C.GLsizei(width), C.GLsizei(height), 0,
		C.GLenum(format), C.GLenum(kind), bytesOf(pixels))
}

func (GL) Viewport(x, y, width, height int) {
	C.glViewport(C.GLint(x), C.GLint(y), C.GLsizei(width), C.GLsizei(height))
}

func (GL) ClearColor(red, green, blue, alpha float32) {
	C.glClearColor(C.GLfloat(red), C.GLfloat(green), C.GLfloat(blue), C.GLfloat(alpha))
}

func (GL) Clear(mask gles.Enum) { C.glClear(C.GLbitfield(mask)) }

func (GL) Enable(capability gles.Enum) { C.glEnable(C.GLenum(capability)) }

func (GL) Disable(capability gles.Enum) { C.glDisable(C.GLenum(capability)) }

func (GL) BlendFuncSeparate(sourceColor, destinationColor, sourceAlpha, destinationAlpha gles.Enum) {
	C.glBlendFuncSeparate(C.GLenum(sourceColor), C.GLenum(destinationColor), C.GLenum(sourceAlpha), C.GLenum(destinationAlpha))
}

func (GL) DepthMask(write bool) {
	flag := C.GLboolean(C.GL_FALSE)
	if write {
		flag = C.GL_TRUE
	}
	C.glDepthMask(flag)
}

func (GL) DepthFunc(function gles.Enum) { C.glDepthFunc(C.GLenum(function)) }

func (GL) DrawArrays(mode gles.Enum, first, count int) {
	C.glDrawArrays(C.GLenum(mode), C.GLint(first), C.GLsizei(count))
}

func (GL) DrawElements(mode gles.Enum, count int, kind gles.Enum, offset int) {
	C.draw_elements(C.GLenum(mode), C.GLsizei(count), C.GLenum(kind), C.GLintptr(offset))
}

// description names the OpenGL ES version and the GPU, as the driver gives them.
func description() string {
	return C.GoString((*C.char)(unsafe.Pointer(C.glGetString(C.GL_VERSION)))) + " · " +
		C.GoString((*C.char)(unsafe.Pointer(C.glGetString(C.GL_RENDERER))))
}
