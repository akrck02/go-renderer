//go:build js && wasm

package browser

import (
	"errors"
	"syscall/js"
	"unsafe"

	"github.com/akrck02/go-renderer/gles"
)

// The WebGL2 parameters this file asks for.
const (
	compileStatus = 0x8B81
	linkStatus    = 0x8B82
)

// WebGL implements gles.Functions over a canvas's WebGL2 context. JavaScript objects (shaders,
// buffers, textures, uniform locations) are kept in tables and handed to the renderers as
// numbers, which is what the gles handles are.
type WebGL struct {
	context  js.Value
	objects  map[uint32]js.Value
	next     uint32
	uniforms []js.Value

	// scratch is reused to copy bytes into JavaScript, so a frame does not allocate a new array
	// for every upload.
	scratch       js.Value
	scratchLength int
	matrix        js.Value // a Float32Array of 16
	matrixBytes   js.Value // the same memory, as bytes
	matrix3       js.Value // a Float32Array of 9
	matrix3Bytes  js.Value
}

// ErrNoWebGL2 is returned when the browser cannot give the canvas a WebGL2 context.
var ErrNoWebGL2 = errors.New("browser: this browser has no WebGL2")

// newWebGL asks the canvas for a WebGL2 context. antialias smooths the edges of 3D geometry; flat
// interfaces do not need it (their shapes are smoothed by the atlas).
func newWebGL(canvas js.Value, antialias bool) (*WebGL, error) {
	options := map[string]any{"antialias": antialias, "alpha": false, "depth": true, "premultipliedAlpha": false}
	context := canvas.Call("getContext", "webgl2", options)
	if context.IsNull() || context.IsUndefined() {
		return nil, ErrNoWebGL2
	}
	matrix := js.Global().Get("Float32Array").New(16)
	matrix3 := js.Global().Get("Float32Array").New(9)
	return &WebGL{context: context, objects: map[uint32]js.Value{}, matrix: matrix,
		matrixBytes: js.Global().Get("Uint8Array").New(matrix.Get("buffer")), matrix3: matrix3,
		matrix3Bytes: js.Global().Get("Uint8Array").New(matrix3.Get("buffer"))}, nil
}

func (webgl *WebGL) keep(object js.Value) uint32 {
	if object.IsNull() || object.IsUndefined() {
		return 0
	}
	webgl.next++
	webgl.objects[webgl.next] = object
	return webgl.next
}

// object returns what a handle names, or null for 0 (WebGL's way to unbind).
func (webgl *WebGL) object(handle uint32) js.Value {
	if handle == 0 {
		return js.Null()
	}
	return webgl.objects[handle]
}

// bytes copies data into a JavaScript Uint8Array of the same length.
func (webgl *WebGL) bytes(data []byte) js.Value {
	if len(data) > webgl.scratchLength {
		webgl.scratchLength = max(len(data), 2*webgl.scratchLength)
		webgl.scratch = js.Global().Get("Uint8Array").New(webgl.scratchLength)
	}
	if webgl.scratchLength == 0 {
		return js.Global().Get("Uint8Array").New(0)
	}
	view := webgl.scratch.Call("subarray", 0, len(data))
	js.CopyBytesToJS(view, data)
	return view
}

func (webgl *WebGL) CreateShader(stage gles.Enum) gles.Shader {
	return gles.Shader(webgl.keep(webgl.context.Call("createShader", uint32(stage))))
}

func (webgl *WebGL) ShaderSource(shader gles.Shader, source string) {
	webgl.context.Call("shaderSource", webgl.object(uint32(shader)), source)
}

func (webgl *WebGL) CompileShader(shader gles.Shader) {
	webgl.context.Call("compileShader", webgl.object(uint32(shader)))
}

func (webgl *WebGL) ShaderCompiled(shader gles.Shader) (bool, string) {
	object := webgl.object(uint32(shader))
	if webgl.context.Call("getShaderParameter", object, compileStatus).Truthy() {
		return true, ""
	}
	return false, webgl.context.Call("getShaderInfoLog", object).String()
}

func (webgl *WebGL) DeleteShader(shader gles.Shader) {
	webgl.context.Call("deleteShader", webgl.object(uint32(shader)))
	delete(webgl.objects, uint32(shader))
}

func (webgl *WebGL) CreateProgram() gles.Program {
	return gles.Program(webgl.keep(webgl.context.Call("createProgram")))
}

func (webgl *WebGL) AttachShader(program gles.Program, shader gles.Shader) {
	webgl.context.Call("attachShader", webgl.object(uint32(program)), webgl.object(uint32(shader)))
}

func (webgl *WebGL) LinkProgram(program gles.Program) {
	webgl.context.Call("linkProgram", webgl.object(uint32(program)))
}

func (webgl *WebGL) ProgramLinked(program gles.Program) (bool, string) {
	object := webgl.object(uint32(program))
	if webgl.context.Call("getProgramParameter", object, linkStatus).Truthy() {
		return true, ""
	}
	return false, webgl.context.Call("getProgramInfoLog", object).String()
}

func (webgl *WebGL) BindAttribLocation(program gles.Program, index uint32, name string) {
	webgl.context.Call("bindAttribLocation", webgl.object(uint32(program)), index, name)
}

func (webgl *WebGL) UseProgram(program gles.Program) {
	webgl.context.Call("useProgram", webgl.object(uint32(program)))
}

func (webgl *WebGL) UniformLocation(program gles.Program, name string) gles.Uniform {
	location := webgl.context.Call("getUniformLocation", webgl.object(uint32(program)), name)
	if location.IsNull() {
		return -1
	}
	webgl.uniforms = append(webgl.uniforms, location)
	return gles.Uniform(len(webgl.uniforms) - 1)
}

// uniform returns a location, or null for -1 (WebGL ignores values set at null).
func (webgl *WebGL) uniform(location gles.Uniform) js.Value {
	if location < 0 || int(location) >= len(webgl.uniforms) {
		return js.Null()
	}
	return webgl.uniforms[location]
}

func (webgl *WebGL) Uniform1i(location gles.Uniform, value int32) {
	webgl.context.Call("uniform1i", webgl.uniform(location), value)
}

func (webgl *WebGL) Uniform1f(location gles.Uniform, value float32) {
	webgl.context.Call("uniform1f", webgl.uniform(location), value)
}

func (webgl *WebGL) Uniform2f(location gles.Uniform, first, second float32) {
	webgl.context.Call("uniform2f", webgl.uniform(location), first, second)
}

func (webgl *WebGL) Uniform3f(location gles.Uniform, first, second, third float32) {
	webgl.context.Call("uniform3f", webgl.uniform(location), first, second, third)
}

func (webgl *WebGL) Uniform4f(location gles.Uniform, first, second, third, fourth float32) {
	webgl.context.Call("uniform4f", webgl.uniform(location), first, second, third, fourth)
}

func (webgl *WebGL) UniformMatrix3(location gles.Uniform, matrix [9]float32) {
	if location < 0 {
		return
	}
	js.CopyBytesToJS(webgl.matrix3Bytes, unsafe.Slice((*byte)(unsafe.Pointer(&matrix[0])), 36))
	webgl.context.Call("uniformMatrix3fv", webgl.uniform(location), false, webgl.matrix3)
}

func (webgl *WebGL) UniformMatrix4(location gles.Uniform, matrix [16]float32) {
	if location < 0 {
		return
	}
	js.CopyBytesToJS(webgl.matrixBytes, unsafe.Slice((*byte)(unsafe.Pointer(&matrix[0])), 64))
	webgl.context.Call("uniformMatrix4fv", webgl.uniform(location), false, webgl.matrix)
}

func (webgl *WebGL) CreateBuffer() gles.Buffer {
	return gles.Buffer(webgl.keep(webgl.context.Call("createBuffer")))
}

func (webgl *WebGL) BindBuffer(target gles.Enum, buffer gles.Buffer) {
	webgl.context.Call("bindBuffer", uint32(target), webgl.object(uint32(buffer)))
}

func (webgl *WebGL) BufferData(target gles.Enum, data []byte, usage gles.Enum) {
	webgl.context.Call("bufferData", uint32(target), webgl.bytes(data), uint32(usage))
}

func (webgl *WebGL) EnableVertexAttribArray(index uint32) {
	webgl.context.Call("enableVertexAttribArray", index)
}

func (webgl *WebGL) DisableVertexAttribArray(index uint32) {
	webgl.context.Call("disableVertexAttribArray", index)
}

func (webgl *WebGL) VertexAttribPointer(index uint32, size int, kind gles.Enum, normalized bool, stride, offset int) {
	webgl.context.Call("vertexAttribPointer", index, size, uint32(kind), normalized, stride, offset)
}

func (webgl *WebGL) CreateTexture() gles.Texture {
	return gles.Texture(webgl.keep(webgl.context.Call("createTexture")))
}

func (webgl *WebGL) ActiveTexture(unit gles.Enum) {
	webgl.context.Call("activeTexture", uint32(unit))
}

func (webgl *WebGL) BindTexture(target gles.Enum, texture gles.Texture) {
	webgl.context.Call("bindTexture", uint32(target), webgl.object(uint32(texture)))
}

func (webgl *WebGL) TexParameteri(target, name gles.Enum, value int32) {
	webgl.context.Call("texParameteri", uint32(target), uint32(name), value)
}

func (webgl *WebGL) PixelStorei(name gles.Enum, value int32) {
	webgl.context.Call("pixelStorei", uint32(name), value)
}

func (webgl *WebGL) TexImage2D(target gles.Enum, level int, internalFormat gles.Enum, width, height int, format, kind gles.Enum, pixels []byte) {
	webgl.context.Call("texImage2D", uint32(target), level, uint32(internalFormat), width, height, 0,
		uint32(format), uint32(kind), webgl.bytes(pixels))
}

func (webgl *WebGL) TexSubImage2D(target gles.Enum, level, x, y, width, height int, format, kind gles.Enum, pixels []byte) {
	webgl.context.Call("texSubImage2D", uint32(target), level, x, y, width, height,
		uint32(format), uint32(kind), webgl.bytes(pixels))
}

func (webgl *WebGL) Viewport(x, y, width, height int) {
	webgl.context.Call("viewport", x, y, width, height)
}

func (webgl *WebGL) ClearColor(red, green, blue, alpha float32) {
	webgl.context.Call("clearColor", red, green, blue, alpha)
}

func (webgl *WebGL) Clear(mask gles.Enum) { webgl.context.Call("clear", uint32(mask)) }

func (webgl *WebGL) Enable(capability gles.Enum) { webgl.context.Call("enable", uint32(capability)) }

func (webgl *WebGL) Disable(capability gles.Enum) { webgl.context.Call("disable", uint32(capability)) }

func (webgl *WebGL) BlendFuncSeparate(sourceColor, destinationColor, sourceAlpha, destinationAlpha gles.Enum) {
	webgl.context.Call("blendFuncSeparate", uint32(sourceColor), uint32(destinationColor), uint32(sourceAlpha), uint32(destinationAlpha))
}

func (webgl *WebGL) DepthMask(write bool) { webgl.context.Call("depthMask", write) }

func (webgl *WebGL) DepthFunc(function gles.Enum) { webgl.context.Call("depthFunc", uint32(function)) }

func (webgl *WebGL) DrawArrays(mode gles.Enum, first, count int) {
	webgl.context.Call("drawArrays", uint32(mode), first, count)
}

func (webgl *WebGL) DrawElements(mode gles.Enum, count int, kind gles.Enum, offset int) {
	webgl.context.Call("drawElements", uint32(mode), count, uint32(kind), offset)
}
