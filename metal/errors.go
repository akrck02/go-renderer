package metal

import "github.com/akrck02/go-renderer/failure"

// Codes of the errors the Metal renderers return. Their details carry what Metal said.
var (
	// Unavailable is Metal unable to start on this computer: no Metal device, shaders that do not
	// compile, or a system other than macOS. A program can fall back to OpenGL.
	Unavailable = failure.Define("metal.unavailable", "Metal cannot start on this computer")
	// ShaderNotCompiled is a Metal shader the device would not compile.
	ShaderNotCompiled = failure.Define("metal.shader-not-compiled", "Metal could not compile a shader")
	// NothingCaptured is a capture to a picture that drew nothing.
	NothingCaptured = failure.Define("metal.nothing-captured", "the capture drew nothing")
)

// ErrUnavailable is Unavailable under its earlier name, kept so programs that check it still work.
//
// Deprecated: use Unavailable.
var ErrUnavailable error = Unavailable
