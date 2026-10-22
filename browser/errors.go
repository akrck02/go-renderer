//go:build js && wasm

package browser

import "github.com/akrck02/go-renderer/failure"

// Codes of the errors OpenCanvas and the renderers return.
var (
	// CanvasNotFound is a page with no canvas of the id asked for (or no canvas at all).
	CanvasNotFound = failure.Define("browser.canvas-not-found", "the page has no canvas with that id")
	// WebGL2Missing is a browser that cannot give the canvas a WebGL2 context.
	WebGL2Missing = failure.Define("browser.webgl2-missing", "this browser has no WebGL2")
)
