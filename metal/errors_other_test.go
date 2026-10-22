//go:build !darwin

package metal

import (
	"errors"
	"testing"
)

func TestOutsideMacOSEveryWindowIsUnavailable(test *testing.T) {
	for name, err := range map[string]error{
		"FlatWindow":  (&FlatWindow{}).Run(nil),
		"SceneWindow": (&SceneWindow{}).Run(nil),
	} {
		if !errors.Is(err, Unavailable) {
			test.Errorf("%s returned %v, not %s", name, err, Unavailable.Name())
		}
	}
}
