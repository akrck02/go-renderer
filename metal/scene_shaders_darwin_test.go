package metal

import (
	"errors"
	"testing"
)

// TestTheSceneShadersCompile compiles the scene's Metal shaders as the renderer would and checks
// that every function the renderer looks up is there. The struct size checks are static_asserts in
// the source, so a layout that drifts from the Go side fails here too.
func TestTheSceneShadersCompile(t *testing.T) {
	functions := []string{
		"sceneVertex", "sceneFragment",
		"skyVertex", "skyFragment",
		"constellationVertex", "constellationLineFragment", "constellationPointFragment",
		"overlayVertex", "overlayFragment",
	}
	err := compileShaderSource(sceneShaderSource, functions)
	if errors.Is(err, ErrUnavailable) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	// the check itself must catch both kinds of mistake, or a pass above would mean nothing
	if compileShaderSource(sceneShaderSource, []string{"aFunctionThatIsNotThere"}) == nil {
		t.Error("a missing function was not reported")
	}
	if compileShaderSource(sceneShaderSource+"\nstatic_assert(false, \"broken on purpose\");\n", functions) == nil {
		t.Error("a source that does not compile was not reported")
	}
}
