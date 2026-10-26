package opengl

import (
	"flag"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// updateReferences writes the frames the tests draw as their new reference pictures:
// go test ./opengl -run Texture -update. Look at the pictures before committing them.
var updateReferences = flag.Bool("update", false, "write the drawn frames as the reference pictures in testdata")

// The tolerance of a comparison: a pixel differs when one of its channels is further than
// channelTolerance from the reference's, and a frame matches while at most differingShare of its
// pixels differ (drivers round a little differently).
const (
	channelTolerance = 8
	differingShare   = 0.005
)

// checkReferencePicture reads back the frame just drawn (width × height, the whole back buffer)
// and compares it with testdata/name.png within the tolerance, or writes it there with -update.
func checkReferencePicture(test *testing.T, name string, width, height int) {
	test.Helper()
	drawn := flipRows(readPixels(width, height), width, height)
	path := filepath.Join("testdata", name+".png")
	if *updateReferences {
		writePicture(test, path, drawn)
		return
	}
	reference := readPicture(test, path)
	if reference.Bounds() != drawn.Bounds() {
		test.Fatalf("%s: drawn %v, the reference is %v", name, drawn.Bounds(), reference.Bounds())
	}
	differing := countDifferingPixels(drawn, reference)
	if share := float64(differing) / float64(width*height); share > differingShare {
		failed := filepath.Join(test.TempDir(), name+".png")
		writePicture(test, failed, drawn)
		test.Errorf("%s: %d pixels (%.1f %%) differ from the reference; the frame drawn is in %s", name, differing, 100*share, failed)
	}
}

// countDifferingPixels counts the pixels with a channel further than the tolerance from the other
// picture's.
func countDifferingPixels(drawn *image.NRGBA, reference image.Image) int {
	differing := 0
	bounds := drawn.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			red, green, blue, _ := reference.At(x, y).RGBA()
			pixel := drawn.NRGBAAt(x, y)
			wanted := [3]uint32{red >> 8, green >> 8, blue >> 8}
			got := [3]uint32{uint32(pixel.R), uint32(pixel.G), uint32(pixel.B)}
			for channel := range wanted {
				if max(wanted[channel], got[channel])-min(wanted[channel], got[channel]) > channelTolerance {
					differing++
					break
				}
			}
		}
	}
	return differing
}

func readPicture(test *testing.T, path string) image.Image {
	test.Helper()
	file, err := os.Open(path)
	if err != nil {
		test.Fatalf("no reference picture (go test -update writes it): %v", err)
	}
	defer file.Close() // only read
	picture, err := png.Decode(file)
	if err != nil {
		test.Fatal(err)
	}
	return picture
}

func writePicture(test *testing.T, path string, picture image.Image) {
	test.Helper()
	file, err := os.Create(path)
	if err != nil {
		test.Fatal(err)
	}
	if err := png.Encode(file, picture); err != nil {
		test.Fatal(err)
	}
	if err := file.Close(); err != nil {
		test.Fatal(err)
	}
}
