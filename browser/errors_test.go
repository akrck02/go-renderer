//go:build js && wasm

package browser

import (
	"errors"
	"syscall/js"
	"testing"
)

func TestAPageWithoutTheCanvasIsCanvasNotFound(test *testing.T) {
	if js.Global().Get("document").IsUndefined() {
		test.Skip("needs a page")
	}
	if _, err := OpenCanvas("no-such-canvas"); !errors.Is(err, CanvasNotFound) {
		test.Fatalf("%v, not %s", err, CanvasNotFound.Name())
	}
}

func TestEachCodeHasItsOwnName(test *testing.T) {
	if CanvasNotFound.Name() != "browser.canvas-not-found" || WebGL2Missing.Name() != "browser.webgl2-missing" {
		test.Fatal("a code lost its name")
	}
}
