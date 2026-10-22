package metal

import "testing"

func TestEachCodeHasItsOwnName(test *testing.T) {
	names := map[string]string{
		Unavailable.Name():       "metal.unavailable",
		ShaderNotCompiled.Name(): "metal.shader-not-compiled",
		NothingCaptured.Name():   "metal.nothing-captured",
	}
	for got, want := range names {
		if got != want {
			test.Errorf("a code is named %s, want %s", got, want)
		}
	}
	if len(names) != 3 {
		test.Error("two codes share a name")
	}
}
