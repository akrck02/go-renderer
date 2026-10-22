package draw2d

import (
	"errors"
	"strings"
	"testing"
)

func TestAMalformedColourIsInvalidColour(test *testing.T) {
	for _, text := range []string{"#12345", "#gggggg", "red"} {
		if _, err := ParseHex(text); !errors.Is(err, InvalidColour) {
			test.Errorf("%s: %v, not %s", text, err, InvalidColour.Name())
		}
	}
}

func TestAPictureThatIsNotOneIsImageNotDecoded(test *testing.T) {
	if _, err := DecodeImage(strings.NewReader("not a picture")); !errors.Is(err, ImageNotDecoded) {
		test.Fatalf("%v, not %s", err, ImageNotDecoded.Name())
	}
}
