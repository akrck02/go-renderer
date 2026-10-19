package opengl

import (
	"image"
	"image/color"
	"os"
	"runtime"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// lightTestSide is the side of the hidden window the light tests draw in, in pixels.
const lightTestSide = 64

// grey is the background the lights fall on: half of each channel.
var grey = draw2d.Color{0.5, 0.5, 0.5, 1}

// TestTheLightLayerLightsTheFrame draws lit lists on OpenGL and reads back the pixels: the ambient
// alone tints the frame, and a point light is full at its centre and gone past its edge. It opens
// a hidden window, so it needs a display; -short skips it.
func TestTheLightLayerLightsTheFrame(test *testing.T) {
	renderer := openLightTestWindow(test, lightTestSide, lightTestSide)
	centre, corner := lightTestSide/2, 2
	cases := []struct {
		name           string
		draw           func(list *draw2d.List)
		at             int
		wanted         uint8
		wantedAtCorner uint8
	}{
		{"a white ambient leaves the colours as they are", func(list *draw2d.List) {
			list.Lighting(draw2d.Color{1, 1, 1, 1})
		}, centre, 128, 128},
		{"an ambient at half intensity halves them", func(list *draw2d.List) {
			list.Lighting(draw2d.Color{1, 1, 1, 0.5})
		}, centre, 64, 64},
		{"an ambient at twice the intensity doubles them", func(list *draw2d.List) {
			list.Lighting(draw2d.Color{1, 1, 1, 2})
		}, centre, 255, 255},
		{"a point light in the dark lights its centre only", func(list *draw2d.List) {
			list.PointLight(lightTestSide/2, lightTestSide/2, 8, 24, draw2d.Color{1, 1, 1, 1})
			list.Lighting(draw2d.Color{0, 0, 0, 1})
		}, centre, 128, 0},
	}
	for _, testCase := range cases {
		list := &draw2d.List{}
		testCase.draw(list)
		renderer.Draw(list, grey, lightTestSide, lightTestSide, 1)
		checkRed(test, testCase.name+", at the centre", centre, testCase.wanted)
		checkRed(test, testCase.name+", at a corner", corner, testCase.wantedAtCorner)
	}
}

// TestALightingLeavesWhatIsDrawnAfterItUnlit checks that an interface drawn after the lighting
// keeps its colours in the dark.
func TestALightingLeavesWhatIsDrawnAfterItUnlit(test *testing.T) {
	renderer := openLightTestWindow(test, lightTestSide, lightTestSide)
	list := &draw2d.List{}
	list.Lighting(draw2d.Color{0, 0, 0, 1})
	list.Rectangle(draw2d.Rect{X: 16, Y: 16, Width: 32, Height: 32}, 0, grey, draw2d.Color{}, 0)
	renderer.Draw(list, grey, lightTestSide, lightTestSide, 1)
	checkRed(test, "the rectangle after the lighting", lightTestSide/2, 128)
	checkRed(test, "the background before it", 2, 0)
}

// TestASpriteLightShinesThroughItsPicture lights the dark with a picture white on its left half and
// clear on its right: only the left is lit.
func TestASpriteLightShinesThroughItsPicture(test *testing.T) {
	renderer := openLightTestWindow(test, lightTestSide, lightTestSide)
	const pictureSide = 8
	pixels := image.NewNRGBA(image.Rect(0, 0, pictureSide, pictureSide))
	for x := range pictureSide / 2 {
		for y := range pictureSide {
			pixels.Set(x, y, color.NRGBA{255, 255, 255, 255})
		}
	}
	list := &draw2d.List{}
	list.SpriteLight(draw2d.NewImage(pixels), draw2d.Rect{Width: lightTestSide, Height: lightTestSide}, draw2d.Color{1, 1, 1, 1})
	list.Lighting(draw2d.Color{0, 0, 0, 1})
	renderer.Draw(list, grey, lightTestSide, lightTestSide, 1)
	checkRedAt(test, "the left, under the white", 8, lightTestSide/2, 128)
	checkRedAt(test, "the right, under the clear", lightTestSide-8, lightTestSide/2, 0)
}

// TestABackdropAndALightingCutTheDrawingInTurn draws a glass panel, lights it in the dark, and
// draws a panel after: the glass goes dark with the rest, the panel after keeps its colour.
func TestABackdropAndALightingCutTheDrawingInTurn(test *testing.T) {
	renderer := openLightTestWindow(test, lightTestSide, lightTestSide)
	list := &draw2d.List{}
	list.Backdrop(draw2d.Rect{X: 4, Y: 4, Width: 24, Height: 24}, 0, 4)
	list.Lighting(draw2d.Color{0, 0, 0, 1})
	list.Rectangle(draw2d.Rect{X: 36, Y: 36, Width: 24, Height: 24}, 0, grey, draw2d.Color{}, 0)
	renderer.Draw(list, grey, lightTestSide, lightTestSide, 1)
	checkRedAt(test, "the glass panel, lit in the dark", 16, lightTestSide-16, 0)
	checkRedAt(test, "the panel after the lighting", 48, lightTestSide-48, 128)
}

// The measured frame's size, in pixels: a laptop's window.
const (
	measuredWidth  = 1280
	measuredHeight = 760
)

// BenchmarkAFrameUnlit and BenchmarkAFrameWithEightLights measure a frame of panels without and
// with eight point lights, the GPU's work waited for. Each opens its own window: OpenGL is bound
// to the thread that made it, and a sub-benchmark would run on another.
func BenchmarkAFrameUnlit(benchmark *testing.B) { measureFrame(benchmark, false) }

func BenchmarkAFrameWithEightLights(benchmark *testing.B) { measureFrame(benchmark, true) }

func measureFrame(benchmark *testing.B, lit bool) {
	renderer := openLightTestWindow(benchmark, measuredWidth, measuredHeight)
	list := measuredFrame(lit)
	benchmark.ResetTimer()
	for range benchmark.N {
		renderer.Draw(list, grey, measuredWidth, measuredHeight, 1)
		gl.Finish()
	}
}

// measuredFrame is twenty panels in rows, and when lit, eight lights over a dim ambient.
func measuredFrame(lit bool) *draw2d.List {
	const panels, columns, panelSide, lights, lightRadius = 20, 5, 200, 8, 160
	list := &draw2d.List{}
	for index := range panels {
		area := draw2d.Rect{X: float32(index%columns) * 250, Y: float32(index/columns) * 180, Width: panelSide, Height: panelSide / 2}
		list.Rectangle(area, 12, draw2d.Color{0.8, 0.7, 0.6, 1}, draw2d.Color{}, 0)
	}
	if !lit {
		return list
	}
	for index := range lights {
		x, y := float32(index%4)*320+160, float32(index/4)*380+190
		list.PointLight(x, y, lightRadius/2, lightRadius, draw2d.Color{1, 0.9, 0.6, 1})
	}
	list.Lighting(draw2d.Color{0.6, 0.6, 0.8, 0.7})
	return list
}

// openLightTestWindow opens a hidden window with an OpenGL context of a size, and its renderer.
func openLightTestWindow(test testing.TB, width, height int) *FlatRenderer {
	test.Helper()
	if testing.Short() || os.Getenv("DISPLAY") == "" && runtime.GOOS == "linux" {
		test.Skip("needs a display")
	}
	runtime.LockOSThread()
	test.Cleanup(runtime.UnlockOSThread)
	if err := glfw.Init(); err != nil {
		test.Skip("no window system: ", err)
	}
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(width, height, "lights", nil, nil)
	if err != nil {
		test.Skip("no OpenGL 4.1 window: ", err)
	}
	test.Cleanup(window.Destroy)
	window.MakeContextCurrent()
	if err := gl.Init(); err != nil {
		test.Fatal(err)
	}
	renderer, err := NewFlatRenderer()
	if err != nil {
		test.Fatal(err)
	}
	return renderer
}

// checkRed reads the red of the pixel at (at, at) in the back buffer, within a rounding's step.
func checkRed(test *testing.T, name string, at int, wanted uint8) {
	test.Helper()
	checkRedAt(test, name, at, at, wanted)
}

// checkRedAt reads the red of the pixel at (x, y), counted from the bottom-left as OpenGL does.
func checkRedAt(test *testing.T, name string, x, y int, wanted uint8) {
	test.Helper()
	const rounding = 2
	var pixel [4]uint8
	gl.Finish()
	gl.ReadPixels(int32(x), int32(y), 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
	if difference := int(pixel[0]) - int(wanted); difference > rounding || difference < -rounding {
		test.Errorf("%s: red is %d, wanted %d", name, pixel[0], wanted)
	}
}
