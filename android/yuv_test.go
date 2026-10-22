package android

import (
	"image"
	"testing"
)

// flatPicture is a 4×2 picture of one luma and one chroma, the chroma planes interleaved as many
// decoders give them.
func flatPicture(luma, blue, red byte) YUVPlanes {
	chroma := []byte{blue, red, blue, red, blue, red, blue, red}
	return YUVPlanes{
		Luma: []byte{luma, luma, luma, luma, luma, luma, luma, luma}, LumaRow: 4,
		Blue: chroma, Red: chroma[1:], BlueRow: 4, RedRow: 4, BluePixel: 2, RedPixel: 2,
		Crop: image.Rect(0, 0, 4, 2),
	}
}

func TestVideoColoursBecomeRGBA(test *testing.T) {
	for _, colour := range []struct {
		name            string
		luma, blue, red byte
		want            [3]byte
	}{
		{"black", 16, 128, 128, [3]byte{0, 0, 0}},
		{"white", 235, 128, 128, [3]byte{255, 255, 255}},
		{"red", 81, 90, 240, [3]byte{255, 0, 0}},
		{"blue", 41, 240, 110, [3]byte{0, 0, 255}},
	} {
		into := image.NewNRGBA(image.Rect(0, 0, 4, 2))
		flatPicture(colour.luma, colour.blue, colour.red).ToRGBA(into)
		got := into.NRGBAAt(3, 1)
		for channel, want := range colour.want {
			value := [3]byte{got.R, got.G, got.B}[channel]
			if difference := int(value) - int(want); difference < -3 || difference > 3 || got.A != 255 {
				test.Errorf("%s turned into %v; want about %v, opaque", colour.name, got, colour.want)
				break
			}
		}
	}
}

func TestOnlyTheCropShows(test *testing.T) {
	planes := flatPicture(235, 128, 128)
	planes.Luma[1] = 16 // a black pixel just inside the crop's start
	planes.Crop = image.Rect(1, 0, 3, 2)
	into := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	planes.ToRGBA(into)
	if first := into.NRGBAAt(0, 0); first.R > 3 {
		test.Fatalf("the picture's first pixel is %v; want the crop's first, black", first)
	}
}
