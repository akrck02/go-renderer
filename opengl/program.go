package opengl

import (
	"strings"

	"github.com/akrck02/go-renderer/failure"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// immediateVertexShader places the immediate-mode shapes with the projection, view and model
// matrices.
const immediateVertexShader = `
    #version 410
    layout(location = 0) in vec3 position;
    uniform mat4 projection;
    uniform mat4 view;
    uniform mat4 model;
    void main() {
        gl_Position = projection * view * model * vec4(position, 1.0);
    }
`

// immediateFragmentShader paints the immediate-mode shapes white.
const immediateFragmentShader = `
    #version 410
    out vec4 fragmentColour;
    uniform vec4 color;
    void main() {
        fragmentColour = vec4(1.0, 1.0, 1.0, 1.0);
    }
`

// newProgram compiles two shader sources and links them into a program.
func newProgram(vertexSource, fragmentSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	fragmentShader, err := compileShader(fragmentSource, gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vertexShader)
		return 0, err
	}
	program, err := linkProgram(vertexShader, fragmentShader)
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
	return program, err
}

// compileShader compiles a shader of kind (gl.VERTEX_SHADER, gl.FRAGMENT_SHADER) from source.
func compileShader(source string, kind uint32) (uint32, error) {
	shader := gl.CreateShader(kind)

	// gl.Strs copies the source without a terminating zero, so the length is given explicitly:
	// with a nil length the driver reads on until the first zero byte in memory and compiles
	// whatever follows the copy. Some sources still end in a zero, which GLSL must not see.
	text := strings.TrimRight(source, "\x00")
	sources, free := gl.Strs(text)
	length := int32(len(text))
	gl.ShaderSource(shader, 1, sources, &length)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status != gl.FALSE {
		return shader, nil
	}
	var logLength int32
	gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
	driverLog := strings.Repeat("\x00", int(logLength+1))
	gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(driverLog))
	gl.DeleteShader(shader)
	return 0, failure.Wrap(ShaderNotCompiled, strings.TrimRight(driverLog, "\x00"), nil)
}

// linkProgram links a vertex and a fragment shader into a program.
func linkProgram(vertexShader, fragmentShader uint32) (uint32, error) {
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status != gl.FALSE {
		return program, nil
	}
	var logLength int32
	gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)
	driverLog := strings.Repeat("\x00", int(logLength+1))
	gl.GetProgramInfoLog(program, logLength, nil, gl.Str(driverLog))
	gl.DeleteProgram(program)
	return 0, failure.Wrap(ProgramNotLinked, strings.TrimRight(driverLog, "\x00"), nil)
}
