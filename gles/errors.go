package gles

import "github.com/akrck02/go-renderer/failure"

// Codes of the errors the renderers return when they are made. Their details carry the driver's
// own log.
var (
	// ShaderNotCompiled is a shader the driver would not compile.
	ShaderNotCompiled = failure.Define("gles.shader-not-compiled", "the driver could not compile a shader")
	// ProgramNotLinked is a pair of shaders the driver would not link into a program.
	ProgramNotLinked = failure.Define("gles.program-not-linked", "the driver could not link a program")
)
