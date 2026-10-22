package opengl

import (
	"errors"
	"testing"

	"github.com/akrck02/go-renderer/failure"
)

func TestEachCodeIsNamedForItsPackage(test *testing.T) {
	names := map[*failure.Code]string{
		FramebufferIncomplete: "opengl.framebuffer-incomplete",
		WindowNotOpened:       "opengl.window-not-opened",
		ContextNotStarted:     "opengl.context-not-started",
		ShaderNotCompiled:     "opengl.shader-not-compiled",
		ProgramNotLinked:      "opengl.program-not-linked",
		FrameNotSaved:         "opengl.frame-not-saved",
		NotImplemented:        "opengl.not-implemented",
	}
	for code, want := range names {
		if code.Name() != want {
			test.Errorf("%s is named %s", want, code.Name())
		}
	}
}

func TestAWrappedCodeKeepsItsCode(test *testing.T) {
	wrapped := failure.Wrap(FramebufferIncomplete, "shadow map, status 0x0", nil)
	if !errors.Is(wrapped, FramebufferIncomplete) {
		test.Fatal("a wrapped FramebufferIncomplete lost its code")
	}
}
