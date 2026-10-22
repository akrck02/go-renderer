package draw2d

import "testing"

// controlCount is how many controls the benchmark draws: a busy screen of an interface.
const controlCount = 300

// BenchmarkBuildingAListOfThreeHundredControls measures a frame of an interface: each control a
// shadow, a rounded rectangle, a label and an icon, inside a clip, into a list reused every frame.
func BenchmarkBuildingAListOfThreeHundredControls(benchmark *testing.B) {
	list := &List{}
	background, border, text := MustHex("#20242c"), MustHex("#3a4150"), MustHex("#e8eaf0")
	benchmark.ReportAllocs()
	for range benchmark.N {
		list.Reset()
		list.PushClip(Rectangle{X: 0, Y: 0, Width: 1280, Height: 720})
		for index := range controlCount {
			area := Rectangle{X: float32(index%10) * 120, Y: float32(index/10) * 24, Width: 110, Height: 20}
			list.Shadow(area, 4, 6, 2, Color{0, 0, 0, 0.3})
			list.Rectangle(area, 4, background, border, 1)
			list.Text("Button", area.X+24, area.Y+14, 12, Regular, text)
			list.Icon("circle", area.X+4, area.Y+2, 16, text)
		}
		list.PopClip()
	}
}
