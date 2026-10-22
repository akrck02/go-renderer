package nintendo3ds

import (
	"testing"

	"github.com/akrck02/go-renderer/failure"
)

func TestEachCodeHasItsOwnName(test *testing.T) {
	codes := map[*failure.Code]string{
		ScreenNotOpened:   "nintendo3ds.screen-not-opened",
		InvalidShader:     "nintendo3ds.invalid-shader",
		InvalidVertices:   "nintendo3ds.invalid-vertices",
		OutOfLinearMemory: "nintendo3ds.out-of-linear-memory",
	}
	for code, want := range codes {
		if code.Name() != want {
			test.Errorf("a code is named %s, want %s", code.Name(), want)
		}
	}
}
