//go:build !darwin

package metal

import (
	"errors"
	"testing"
)

func TestOutsideMacOSEveryWindowIsUnavailable(test *testing.T) {
	for name, err := range map[string]error{
		"FlatWindow":  (&FlatWindow{}).StartLoop(nil),
		"SceneWindow": (&SceneWindow{}).StartLoop(nil),
	} {
		if !errors.Is(err, Unavailable) || !errors.Is(err, ErrUnavailable) {
			test.Errorf("%s returned %v, not %s", name, err, Unavailable.Name())
		}
	}
}
