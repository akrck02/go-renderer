package metal

import (
	"errors"
	"testing"
)

// TestTheSceneShadersCompile compiles the scene's Metal shaders as the renderer would and checks
// that every function the renderer looks up is there. The struct size checks are static_asserts in
// the source, so a layout that drifts from the Go side fails here too.
func TestTheSceneShadersCompile(test *testing.T) {
	functions := []string{
		"sceneVertex", "sceneFragment",
		"skyVertex", "skyFragment",
		"constellationVertex", "constellationLineFragment", "constellationPointFragment",
		"overlayVertex", "overlayFragment",
	}
	err := compileShaderSource(sceneShaderSource, functions)
	if errors.Is(err, ErrUnavailable) {
		test.Skip(err)
	}
	if err != nil {
		test.Fatal(err)
	}
	// the check itself must catch both kinds of mistake, or a pass above would mean nothing
	if compileShaderSource(sceneShaderSource, []string{"aFunctionThatIsNotThere"}) == nil {
		test.Error("a missing function was not reported")
	}
	if compileShaderSource(sceneShaderSource+"\nstatic_assert(false, \"broken on purpose\");\n", functions) == nil {
		test.Error("a source that does not compile was not reported")
	}
}
