package opengl

import (
	"errors"
	"testing"

	"github.com/akrck02/go-renderer/failure"
)

func TestEachCodeIsNamedForItsPackage(test *testing.T) {
	if FramebufferIncomplete.Name() != "opengl.framebuffer-incomplete" {
		test.Fatalf("FramebufferIncomplete is %s", FramebufferIncomplete.Name())
	}
	wrapped := failure.Wrap(FramebufferIncomplete, "shadow map, status 0x0", nil)
	if !errors.Is(wrapped, FramebufferIncomplete) {
		test.Fatal("a wrapped FramebufferIncomplete lost its code")
	}
}
