package openxr

import (
	"strings"
	"testing"

	"github.com/akrck02/go-renderer/failure"
)

func TestErrorsAreNamedForTheirPackage(test *testing.T) {
	codes := []*failure.Code{
		LoaderNotFound, ExtensionMissing, InstanceNotCreated, HeadsetNotFound, GraphicsUnsupported,
		SessionNotCreated, FramebufferIncomplete, ControlsNotCreated, FrameFailed, SessionLost,
	}
	seen := map[string]bool{}
	for _, code := range codes {
		if !strings.HasPrefix(code.Name(), "openxr.") {
			test.Errorf("%s is not named for the package", code.Name())
		}
		if seen[code.Name()] {
			test.Errorf("%s is declared twice", code.Name())
		}
		seen[code.Name()] = true
	}
}

func TestACallErrorKeepsItsCodeAndNamesTheCall(test *testing.T) {
	err := callError(FrameFailed, "xrWaitFrame", "XR_ERROR_RUNTIME_FAILURE")
	if !failure.Is(err, FrameFailed) {
		test.Fatal("the error lost its code")
	}
	if err.Error() != "openxr.frame-failed: xrWaitFrame: XR_ERROR_RUNTIME_FAILURE" {
		test.Fatalf("message: %q", err.Error())
	}
}
