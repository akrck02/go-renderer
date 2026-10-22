package gles

import "strings"

// recorder is a fake graphics context: it hands out handles and remembers the calls the tests
// look at, so the renderers can be checked on any computer, with no display.
type recorder struct {
	nextHandle         uint32
	sources            []string
	failLink           bool
	uniforms           map[string]Uniform
	matrixUploads      int
	textureUploads     int
	textureAreaUploads int
	bufferUploads      map[Enum]int
	draws              []recordedDraw
	failCompile        bool
	attributes         []string
	uniformValues      int
	enables            int
	snapshots          int // frames copied into a texture: one per backdrop snapshot
	bound              Framebuffer
}

type recordedDraw struct {
	elements    bool
	count       int
	framebuffer Framebuffer // the one bound when it was drawn
}

func newRecorder() *recorder {
	return &recorder{uniforms: map[string]Uniform{}, bufferUploads: map[Enum]int{}}
}

func (fake *recorder) handle() uint32 {
	fake.nextHandle++
	return fake.nextHandle
}

func (fake *recorder) CreateShader(Enum) Shader { return Shader(fake.handle()) }
func (fake *recorder) SetShaderSource(_ Shader, source string) {
	fake.sources = append(fake.sources, source)
}
func (fake *recorder) CompileShader(Shader) {}
func (fake *recorder) ShaderCompiled(Shader) (bool, string) {
	if fake.failCompile {
		return false, "0:1: broken on purpose"
	}
	return true, ""
}
func (fake *recorder) DeleteShader(Shader)          {}
func (fake *recorder) CreateProgram() Program       { return Program(fake.handle()) }
func (fake *recorder) AttachShader(Program, Shader) {}
func (fake *recorder) LinkProgram(Program)          {}
func (fake *recorder) ProgramLinked(Program) (bool, string) {
	if fake.failLink {
		return false, "linking broken on purpose"
	}
	return true, ""
}
func (fake *recorder) BindAttributeLocation(_ Program, index uint32, name string) {
	fake.attributes = append(fake.attributes, name)
}
func (fake *recorder) UseProgram(Program) {}
func (fake *recorder) UniformLocation(_ Program, name string) Uniform {
	location, known := fake.uniforms[name]
	if !known {
		location = Uniform(len(fake.uniforms))
		fake.uniforms[name] = location
	}
	return location
}
func (fake *recorder) SetUniformInteger(Uniform, int32)                         { fake.uniformValues++ }
func (fake *recorder) SetUniformFloat(Uniform, float32)                         { fake.uniformValues++ }
func (fake *recorder) SetUniformTwoFloats(Uniform, float32, float32)            { fake.uniformValues++ }
func (fake *recorder) SetUniformThreeFloats(Uniform, float32, float32, float32) { fake.uniformValues++ }
func (fake *recorder) SetUniformFourFloats(Uniform, float32, float32, float32, float32) {
	fake.uniformValues++
}
func (fake *recorder) SetUniformMatrix3By3(Uniform, [9]float32)                  {}
func (fake *recorder) SetUniformMatrix4By4(Uniform, [16]float32)                 { fake.matrixUploads++ }
func (fake *recorder) CreateBuffer() Buffer                                      { return Buffer(fake.handle()) }
func (fake *recorder) BindBuffer(Enum, Buffer)                                   {}
func (fake *recorder) SetBufferData(target Enum, _ []byte, _ Enum)               { fake.bufferUploads[target]++ }
func (fake *recorder) EnableVertexAttribute(uint32)                              { fake.enables++ }
func (fake *recorder) DisableVertexAttribute(uint32)                             {}
func (fake *recorder) DescribeVertexAttribute(uint32, int, Enum, bool, int, int) {}
func (fake *recorder) CreateTexture() Texture                                    { return Texture(fake.handle()) }
func (fake *recorder) SelectTextureUnit(Enum)                                    {}
func (fake *recorder) BindTexture(Enum, Texture)                                 {}
func (fake *recorder) SetTextureParameter(Enum, Enum, int32)                     {}
func (fake *recorder) SetPixelStorage(Enum, int32)                               {}
func (fake *recorder) SetTextureImage(Enum, int, Enum, int, int, Enum, Enum, []byte) {
	fake.textureUploads++
}
func (fake *recorder) ReplaceTexturePart(Enum, int, int, int, int, int, Enum, Enum, []byte) {
	fake.textureAreaUploads++
}
func (fake *recorder) CopyFramebufferToTexture(Enum, int, int, int, int, int, int, int) {
	fake.snapshots++
}
func (fake *recorder) DeleteTexture(Texture) {}
func (fake *recorder) CreateFramebuffer() Framebuffer {
	fake.nextHandle++
	return Framebuffer(fake.nextHandle)
}
func (fake *recorder) BindFramebuffer(_ Enum, framebuffer Framebuffer)           { fake.bound = framebuffer }
func (fake *recorder) AttachTextureToFramebuffer(Enum, Enum, Enum, Texture, int) {}
func (fake *recorder) DeleteFramebuffer(Framebuffer)                             {}
func (fake *recorder) SetViewport(int, int, int, int)                            {}
func (fake *recorder) SetClearColor(float32, float32, float32, float32)          {}
func (fake *recorder) Clear(Enum)                                                {}
func (fake *recorder) Enable(Enum)                                               {}
func (fake *recorder) Disable(Enum)                                              {}
func (fake *recorder) SetBlendFactors(Enum, Enum, Enum, Enum)                    {}
func (fake *recorder) SetDepthWrite(bool)                                        {}
func (fake *recorder) SetDepthFunction(Enum)                                     {}
func (fake *recorder) DrawArrays(_ Enum, _ int, count int) {
	fake.draws = append(fake.draws, recordedDraw{count: count, framebuffer: fake.bound})
}
func (fake *recorder) DrawElements(_ Enum, count int, _ Enum, _ int) {
	fake.draws = append(fake.draws, recordedDraw{elements: true, count: count, framebuffer: fake.bound})
}

// forget clears what the last frame recorded.
func (fake *recorder) forget() {
	fake.draws = nil
	fake.textureUploads, fake.textureAreaUploads, fake.matrixUploads, fake.uniformValues, fake.enables = 0, 0, 0, 0, 0
	fake.bufferUploads = map[Enum]int{}
}

// sourcesWith returns the shader sources that contain a piece of text.
func (fake *recorder) sourcesWith(text string) []string {
	var found []string
	for _, source := range fake.sources {
		if strings.Contains(source, text) {
			found = append(found, source)
		}
	}
	return found
}
