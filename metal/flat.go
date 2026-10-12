package metal

import "errors"

// ErrUnavailable is returned (wrapped) when Metal cannot start on this computer: no Metal device,
// shaders that do not compile, or a system other than macOS. A program can fall back to OpenGL.
var ErrUnavailable = errors.New("metal unavailable")
