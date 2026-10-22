package ctr

import (
	"testing"

	"github.com/akrck02/go-renderer/failure"
)

func TestEachCodeHasItsOwnName(test *testing.T) {
	codes := map[*failure.Code]string{
		ScreenNotOpened:   "ctr.screen-not-opened",
		InvalidShader:     "ctr.invalid-shader",
		InvalidVertices:   "ctr.invalid-vertices",
		OutOfLinearMemory: "ctr.out-of-linear-memory",
	}
	for code, want := range codes {
		if code.Name() != want {
			test.Errorf("a code is named %s, want %s", code.Name(), want)
		}
	}
}
