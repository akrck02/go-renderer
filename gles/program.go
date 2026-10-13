package gles

import (
	"fmt"

	"github.com/akrck02/go-renderer/graphics"
)

// program is a linked shader program with its uniform locations remembered, so a frame does not
// ask the driver for them again (in a browser each question is a call into JavaScript).
type program struct {
	functions Functions
	handle    Program
	uniforms  map[string]Uniform
}

// newProgram compiles and links a vertex and a fragment shader, giving the attributes the indices
// of their place in attributes.
func newProgram(functions Functions, vertexSource, fragmentSource string, attributes []string) (*program, error) {
	vertex, err := compileShader(functions, VertexShader, vertexSource)
	if err != nil {
		return nil, err
	}
	defer functions.DeleteShader(vertex)
	fragment, err := compileShader(functions, FragmentShader, fragmentSource)
	if err != nil {
		return nil, err
	}
	defer functions.DeleteShader(fragment)
	handle := functions.CreateProgram()
	functions.AttachShader(handle, vertex)
	functions.AttachShader(handle, fragment)
	for index, name := range attributes {
		functions.BindAttribLocation(handle, uint32(index), name)
	}
	functions.LinkProgram(handle)
	if linked, log := functions.ProgramLinked(handle); !linked {
		return nil, fmt.Errorf("gles: linking a program: %s", log)
	}
	return &program{functions: functions, handle: handle, uniforms: map[string]Uniform{}}, nil
}

func compileShader(functions Functions, stage Enum, source string) (Shader, error) {
	shader := functions.CreateShader(stage)
	functions.ShaderSource(shader, source)
	functions.CompileShader(shader)
	if compiled, log := functions.ShaderCompiled(shader); !compiled {
		functions.DeleteShader(shader)
		return 0, fmt.Errorf("gles: compiling a shader: %s", log)
	}
	return shader, nil
}

func (program *program) use() { program.functions.UseProgram(program.handle) }

func (program *program) location(name string) Uniform {
	if location, known := program.uniforms[name]; known {
		return location
	}
	location := program.functions.UniformLocation(program.handle, name)
	program.uniforms[name] = location
	return location
}

func (program *program) setInteger(name string, value int32) {
	program.functions.Uniform1i(program.location(name), value)
}

func (program *program) setFloat(name string, value float32) {
	program.functions.Uniform1f(program.location(name), value)
}

func (program *program) setVector2(name string, first, second float32) {
	program.functions.Uniform2f(program.location(name), first, second)
}

func (program *program) setVector3(name string, vector graphics.Vec4) {
	program.functions.Uniform3f(program.location(name), vector[0], vector[1], vector[2])
}

func (program *program) setVector4(name string, vector graphics.Vec4) {
	program.functions.Uniform4f(program.location(name), vector[0], vector[1], vector[2], vector[3])
}

func (program *program) setMatrix3(name string, matrix [9]float32) {
	program.functions.UniformMatrix3(program.location(name), matrix)
}

func (program *program) setMatrix(name string, matrix graphics.Mat4) {
	program.functions.UniformMatrix4(program.location(name), matrix)
}
