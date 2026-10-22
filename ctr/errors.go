package ctr

import "github.com/akrck02/go-renderer/failure"

// Codes of the errors the 3DS screens and renderers return.
var (
	// ScreenNotOpened is a screen citro3d could not prepare for drawing.
	ScreenNotOpened = failure.Define("ctr.screen-not-opened", "citro3d could not open the screen")
	// InvalidShader is a shader binary that is empty or that the GPU's library cannot read.
	InvalidShader = failure.Define("ctr.invalid-shader", "the shader binary cannot be read")
	// InvalidVertices is a list of values that does not make whole vertices.
	InvalidVertices = failure.Define("ctr.invalid-vertices", "the values do not make whole vertices")
	// OutOfLinearMemory is the GPU's memory (linear memory) full.
	OutOfLinearMemory = failure.Define("ctr.out-of-linear-memory", "the GPU's linear memory is full")
)
