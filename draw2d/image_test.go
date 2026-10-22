package draw2d

import (
	"bytes"
	"hash/maphash"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestAnImageKeepsStraightColours(test *testing.T) {
	premultiplied := image.NewRGBA(image.Rect(10, 10, 12, 12))
	premultiplied.Set(10, 10, color.NRGBA{R: 200, G: 100, B: 50, A: 128})
	picture := NewImage(premultiplied)
	if width, height := picture.Size(); width != 2 || height != 2 {
		test.Fatalf("size %d×%d; want the picture's 2×2", width, height)
	}
	if got := picture.Pixels.NRGBAAt(0, 0); got.A != 128 || got.R < 198 || got.R > 202 {
		test.Fatalf("first pixel %v; want the straight colour back, at the origin", got)
	}
	if NewImage(premultiplied).ID() == picture.ID() {
		test.Fatal("two images have the same id")
	}
}

func TestAChangedImageChangesTheFingerprint(test *testing.T) {
	picture := NewBlankImage(4, 4)
	var list List
	list.Image(picture, Rect{X: 0, Y: 0, Width: 40, Height: 40}, 8, Color{1, 1, 1, 1})
	seed := maphash.MakeSeed()
	before := list.Fingerprint(seed)
	if list.Fingerprint(seed) != before {
		test.Fatal("the same list has two fingerprints")
	}
	picture.Changed()
	if list.Fingerprint(seed) == before {
		test.Fatal("a picture with new pixels left the fingerprint as it was, so the frame would not be drawn")
	}
}

func TestNothingToShowDrawsNoImage(test *testing.T) {
	var list List
	area := Rect{Width: 40, Height: 40}
	list.Image(nil, area, 0, Color{1, 1, 1, 1})
	list.Image(NewBlankImage(0, 0), area, 0, Color{1, 1, 1, 1})
	list.Image(NewBlankImage(4, 4), area, 0, Color{1, 1, 1, 0})
	if !list.IsEmpty() {
		test.Fatalf("%d commands for no picture, an empty one and an invisible one; want none", len(list.Commands))
	}
}

func TestAPictureIsDecodedFromItsBytes(test *testing.T) {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 3, 2))); err != nil {
		test.Fatal(err)
	}
	picture, err := DecodeImage(&encoded)
	if err != nil {
		test.Fatal(err)
	}
	if width, height := picture.Size(); width != 3 || height != 2 {
		test.Fatalf("decoded %d×%d; want 3×2", width, height)
	}
	if _, err := DecodeImage(strings.NewReader("not a picture")); err == nil {
		test.Fatal("bytes that are no picture decoded")
	}
}
