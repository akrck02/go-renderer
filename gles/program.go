package gles

import (
	"github.com/akrck02/go-renderer/failure"
	"github.com/akrck02/go-renderer/graphics"
)

// program is a linked shader program with its uniform locations and values remembered: a frame
// does not ask the driver for a location again, nor send a value the uniform already holds. Each
// call is costly where the bindings cross to another thread (golang.org/x/mobile/gl, a few calls at
// a time) or into JavaScript.
type program struct {
	functions Functions
	handle    Program
	uniforms  map[string]Uniform
	values    map[Uniform][16]float32 // what each uniform holds, padded with zeros
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
		functions.BindAttributeLocation(handle, uint32(index), name)
	}
	functions.LinkProgram(handle)
	if linked, log := functions.ProgramLinked(handle); !linked {
		return nil, failure.Wrap(ProgramNotLinked, log, nil)
	}
	return &program{functions: functions, handle: handle, uniforms: map[string]Uniform{}, values: map[Uniform][16]float32{}}, nil
}

func compileShader(functions Functions, stage Enum, source string) (Shader, error) {
	shader := functions.CreateShader(stage)
	functions.SetShaderSource(shader, source)
	functions.CompileShader(shader)
	if compiled, log := functions.ShaderCompiled(shader); !compiled {
		functions.DeleteShader(shader)
		return 0, failure.Wrap(ShaderNotCompiled, log, nil)
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

// changed tells whether a uniform must be sent: it does not hold this value yet (or the program
// does not use it, location -1, and nothing need be sent). It remembers the value as sent.
func (program *program) changed(name string, values ...float32) (Uniform, bool) {
	location := program.location(name)
	if location < 0 {
		return location, false
	}
	var padded [16]float32
	copy(padded[:], values)
	if held, known := program.values[location]; known && held == padded {
		return location, false
	}
	program.values[location] = padded
	return location, true
}

func (program *program) setInteger(name string, value int32) {
	if location, send := program.changed(name, float32(value)); send {
		program.functions.SetUniformInteger(location, value)
	}
}

func (program *program) setFloat(name string, value float32) {
	if location, send := program.changed(name, value); send {
		program.functions.SetUniformFloat(location, value)
	}
}

func (program *program) setVector2(name string, first, second float32) {
	if location, send := program.changed(name, first, second); send {
		program.functions.SetUniformTwoFloats(location, first, second)
	}
}

func (program *program) setVector3(name string, vector graphics.Vector) {
	if location, send := program.changed(name, vector[0], vector[1], vector[2]); send {
		program.functions.SetUniformThreeFloats(location, vector[0], vector[1], vector[2])
	}
}

func (program *program) setVector4(name string, vector graphics.Vector) {
	if location, send := program.changed(name, vector[:]...); send {
		program.functions.SetUniformFourFloats(location, vector[0], vector[1], vector[2], vector[3])
	}
}

func (program *program) setMatrix3(name string, matrix [9]float32) {
	if location, send := program.changed(name, matrix[:]...); send {
		program.functions.SetUniformMatrix3By3(location, matrix)
	}
}

func (program *program) setMatrix(name string, matrix graphics.Matrix) {
	if location, send := program.changed(name, matrix[:]...); send {
		program.functions.SetUniformMatrix4By4(location, matrix)
	}
}
