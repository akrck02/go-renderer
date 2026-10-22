// Package webgpu is the beginning of a WebGPU backend for the immediate-mode Renderer contract.
// It will be built later: until then every function returns NotImplemented.
package webgpu

import "github.com/akrck02/go-renderer/failure"

// NotImplemented is what every function of the WebGPU backend returns until it is built.
var NotImplemented = failure.Define("webgpu.not-implemented", "the WebGPU backend is not implemented yet")
