package overlay

import (
	"errors"
	"testing"
)

func TestAFontThatIsNotAFontIsFontNotRead(test *testing.T) {
	broken := Fonts{Bold: []byte("not a font")}
	if _, err := NewPainter(broken); !errors.Is(err, FontNotRead) {
		test.Fatalf("NewPainter returned %v, not %s", err, FontNotRead.Name())
	}
	if _, err := NewMeasurer(broken); !errors.Is(err, FontNotRead) {
		test.Fatalf("NewMeasurer returned %v, not %s", err, FontNotRead.Name())
	}
}

func TestABrokenColourFontIsColourFontInvalid(test *testing.T) {
	broken := map[string][]byte{
		"too short":         []byte("short"),
		"no colour tables":  make([]byte, 12),
		"directory too big": {0, 1, 0, 0, 0, 9, 0, 0, 0, 0, 0, 0},
	}
	for name, data := range broken {
		if _, err := parseColourFont(data); !errors.Is(err, colourFontInvalid) {
			test.Errorf("%s: %v, not %s", name, err, colourFontInvalid.Name())
		}
	}
}
