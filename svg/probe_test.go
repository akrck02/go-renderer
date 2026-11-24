//go:build probe

// The probe reads real icon sets, which CI does not have: run it with
//
//	SVG_SETS=/folder/with/sets SVG_PROBE_OUT=/folder/for/pictures go test -tags probe -run Probe -v ./svg
//
// The comparison needs ImageMagick (magick) with librsvg, the renderer GNOME draws SVG with.
package svg

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// setFiles are the SVG files under the folder SVG_SETS names; the test is skipped without it.
func setFiles(test *testing.T) []string {
	folder := os.Getenv("SVG_SETS")
	if folder == "" {
		test.Skip("SVG_SETS names no folder of icon sets")
	}
	var files []string
	err := filepath.WalkDir(folder, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && filepath.Ext(path) == ".svg" {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		test.Fatal(err)
	}
	return files
}

func TestProbeEveryIconOfTheSetsReads(test *testing.T) {
	files, failed := setFiles(test), 0
	for _, path := range files {
		source, err := os.ReadFile(path)
		if err != nil {
			test.Fatal(err)
		}
		if _, err := Read(source); err != nil {
			failed++
			if failed <= 20 {
				test.Errorf("%s: %v", path, err)
			}
		}
	}
	test.Logf("%d icons read, %d failed", len(files), failed)
}

// probeSize is the size icons are compared at, in pixels: twice the sidebar's 16, as a dense
// screen draws them.
const probeSize = 32

// difference is how far an icon's picture is from the reference one.
type difference struct {
	path   string
	amount float64
	ours   *image.NRGBA
	theirs image.Image
}

// TestProbeIconsLookAsLibrsvgDrawsThem compares every fifth icon with librsvg's picture, and
// writes the 20 most different, ours beside theirs, for a person to look at.
func TestProbeIconsLookAsLibrsvgDrawsThem(test *testing.T) {
	files, out := setFiles(test), os.Getenv("SVG_PROBE_OUT")
	var differences []difference
	for index := 0; index < len(files); index += 5 {
		source, _ := os.ReadFile(files[index])
		drawing, err := Read(source)
		if err != nil {
			continue
		}
		theirs, err := librsvgPicture(files[index])
		if err != nil {
			test.Fatal(err)
		}
		ours := drawing.Picture(probeSize, probeSize).Pixels
		differences = append(differences, difference{files[index], meanDifference(ours, theirs), ours, theirs})
	}
	slices.SortFunc(differences, func(one, other difference) int { return int(math.Copysign(1, other.amount-one.amount)) })
	total := 0.0
	for _, each := range differences {
		total += each.amount
	}
	test.Logf("%d icons compared, mean difference %.4f", len(differences), total/float64(len(differences)))
	for rank, each := range differences[:min(20, len(differences))] {
		test.Logf("%2d. %.4f %s", rank+1, each.amount, each.path)
		if out != "" {
			writeSideBySide(test, filepath.Join(out, fmt.Sprintf("%02d.png", rank+1)), each)
		}
	}
}

// librsvgPicture is librsvg's picture of an SVG file, drawn straight at the size: one drawn larger
// and shrunk would be blurred by the shrinking, not by librsvg.
func librsvgPicture(path string) (image.Image, error) {
	size := fmt.Sprintf("%dx%d", probeSize, probeSize)
	command := exec.Command("magick", "-background", "none", "-size", size, path, "-gravity", "center", "-extent", size, "png:-")
	output, err := command.Output()
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(output))
}

// meanDifference is the mean of each pixel's difference in colour and opacity, from 0 to 1, the
// colours weighed by their opacity so that an invisible colour counts for nothing.
func meanDifference(ours *image.NRGBA, theirs image.Image) float64 {
	total := 0.0
	for y := range probeSize {
		for x := range probeSize {
			red, green, blue, alpha := ours.At(x, y).RGBA()
			otherRed, otherGreen, otherBlue, otherAlpha := theirs.At(x, y).RGBA()
			for _, pair := range [][2]uint32{{red, otherRed}, {green, otherGreen}, {blue, otherBlue}, {alpha, otherAlpha}} {
				total += math.Abs(float64(pair[0])-float64(pair[1])) / 65535
			}
		}
	}
	return total / (probeSize * probeSize * 4)
}

// writeSideBySide writes ours and theirs, four times larger, side by side on grey.
func writeSideBySide(test *testing.T, path string, each difference) {
	const zoom = 4
	sheet := image.NewNRGBA(image.Rect(0, 0, probeSize*zoom*2+8, probeSize*zoom))
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(color.Gray{Y: 128}), image.Point{}, draw.Src)
	for half, picture := range []image.Image{each.ours, each.theirs} {
		for y := range probeSize * zoom {
			for x := range probeSize * zoom {
				sheet.Set(half*(probeSize*zoom+8)+x, y, mixOverGrey(picture.At(x/zoom, y/zoom)))
			}
		}
	}
	file, err := os.Create(path)
	if err != nil {
		test.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, sheet); err != nil {
		test.Fatal(err)
	}
}

// mixOverGrey is a colour laid over the sheet's grey, so that see-through parts show.
func mixOverGrey(colour color.Color) color.Color {
	red, green, blue, alpha := colour.RGBA()
	grey := uint32(128 * 257)
	over := func(channel uint32) uint8 { return uint8((channel + grey*(65535-alpha)/65535) >> 8) }
	return color.RGBA{over(red), over(green), over(blue), 255}
}
