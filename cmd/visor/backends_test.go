package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMetalDrawsTheSceneLikeOpenGL captures the sample island through both backends, by day and by
// night, and compares them. They draw the same plan with translated shaders, so nearly every pixel
// matches or differs by rounding (1/255); a few may differ where two surfaces lie in the same plane
// and depth precision decides which shows (Metal's depth is 32-bit float, OpenGL's 24-bit). It opens
// hidden windows, so it needs macOS with a display; -short skips it.
func TestMetalDrawsTheSceneLikeOpenGL(test *testing.T) {
	if runtime.GOOS != "darwin" || testing.Short() {
		test.Skip("needs macOS and a display")
	}
	directory := test.TempDir()
	program := filepath.Join(directory, "visor")
	if output, err := exec.Command("go", "build", "-o", program, ".").CombinedOutput(); err != nil {
		test.Fatalf("building the visor: %v\n%s", err, output)
	}
	for _, sample := range []string{"island", "night"} {
		metal := capture(test, program, directory, "metal", sample)
		opengl := capture(test, program, directory, "opengl", sample)
		beyondRounding := countBeyondRounding(test, metal, opengl)
		if limit := len(metal.Pix) / 4 / 1000; beyondRounding > limit {
			test.Errorf("%s: %d pixels differ by more than rounding, more than 0.1 %%", sample, beyondRounding)
		}
	}
}

func capture(test *testing.T, program, directory, backend, sample string) *image.NRGBA {
	test.Helper()
	path := filepath.Join(directory, backend+"-"+sample+".png")
	command := exec.Command(program, "-backend", backend, "-sample", sample, "-time", "12.5", "-width", "640", "-height", "400", "-capture", path)
	if output, err := command.CombinedOutput(); err != nil {
		test.Fatalf("capturing through %s: %v\n%s", backend, err, output)
	}
	file, err := os.Open(path)
	if err != nil {
		test.Fatal(err)
	}
	defer file.Close()
	picture, err := png.Decode(file)
	if err != nil {
		test.Fatal(err)
	}
	converted := image.NewNRGBA(picture.Bounds())
	draw.Draw(converted, converted.Bounds(), picture, picture.Bounds().Min, draw.Src)
	return converted
}

// countBeyondRounding counts the pixels where a colour channel differs by more than 1/255.
func countBeyondRounding(test *testing.T, first, second *image.NRGBA) int {
	test.Helper()
	if first.Bounds() != second.Bounds() {
		test.Fatalf("sizes differ: %v and %v", first.Bounds(), second.Bounds())
	}
	count := 0
	for index := 0; index < len(first.Pix); index += 4 {
		for channel := 0; channel < 3; channel++ {
			difference := int(first.Pix[index+channel]) - int(second.Pix[index+channel])
			if difference > 1 || difference < -1 {
				count++
				break
			}
		}
	}
	return count
}
