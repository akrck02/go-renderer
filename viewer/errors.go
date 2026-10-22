package viewer

import "github.com/akrck02/go-renderer/failure"

// Codes of the errors the viewer's flags return.
var (
	// InvalidPoint is a point flag (-look-at) not written as two numbers, "x,z".
	InvalidPoint = failure.Define("viewer.invalid-point", "a point is written as two numbers, x,z")
	// UnknownBackend is a -backend that is not automatic, metal or opengl.
	UnknownBackend = failure.Define("viewer.unknown-backend", "the backend is automatic, metal or opengl")
)
