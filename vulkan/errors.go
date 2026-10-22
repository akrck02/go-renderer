// Package vulkan is the beginning of a Vulkan backend for the immediate-mode Renderer contract.
// It will be built later: until then every function returns NotImplemented.
package vulkan

import "github.com/akrck02/go-renderer/failure"

// NotImplemented is what every function of the Vulkan backend returns until it is built.
var NotImplemented = failure.Define("vulkan.not-implemented", "the Vulkan backend is not implemented yet")
