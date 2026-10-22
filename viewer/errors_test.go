package viewer

import (
	"errors"
	"testing"
)

func TestAPointThatIsNotTwoNumbersIsInvalidPoint(test *testing.T) {
	var point *[2]float32
	for _, text := range []string{"1", "1,2,3", "one,2"} {
		if err := (pointFlag{&point}).Set(text); !errors.Is(err, InvalidPoint) {
			test.Errorf("%s: %v, not %s", text, err, InvalidPoint.Name())
		}
	}
}

func TestABackendNotKnownIsUnknownBackend(test *testing.T) {
	var backend Backend
	if err := backend.Set("vulkan"); !errors.Is(err, UnknownBackend) {
		test.Fatalf("%v, not %s", err, UnknownBackend.Name())
	}
}
