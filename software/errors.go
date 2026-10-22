// Package software is the beginning of a software backend for the immediate-mode Renderer contract.
// It will be built later: until then every function returns NotImplemented.
package software

import "github.com/akrck02/go-renderer/failure"

// NotImplemented is what every function of the software backend returns until it is built.
var NotImplemented = failure.Define("software.not-implemented", "the software backend is not implemented yet")
