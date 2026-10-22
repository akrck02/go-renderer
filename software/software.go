package software

import "github.com/akrck02/go-renderer/window"

// Window will open a window drawn with the software immediate-mode renderer. Until it is built,
// every method returns NotImplemented.
type Window struct{}

// Run returns NotImplemented.
func (*Window) Run(app *window.Application) error { return NotImplemented }

// Draw returns NotImplemented.
func (*Window) Draw(app *window.Application) error { return NotImplemented }
