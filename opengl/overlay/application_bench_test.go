package overlay

// Benchmarks of the painter on a draw list shaped like a desktop application's frame (the
// go-ui register example, cmd/app: about 47 rectangles, 110 lines of text of about 9 characters
// and 27 icons, half of them clipped), turned into quads at 2× density, headless. The
// "hiddenRows" variant adds table rows scrolled out of their clip rectangle, which go-ui still
// emits: draw2d culls rectangles and icons that are wholly clipped away, but never text, so the
// painter lays out every glyph and clipQuads throws the quads away. Run with:
//
//	go test ./opengl/overlay -run xxx -bench Application -benchmem

import (
	"fmt"
	"sort"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/scene"
)

var (
	applicationInk     = draw2d.MustHex("#14202f")
	applicationSurface = draw2d.MustHex("#ffffff")
	applicationLine    = draw2d.MustHex("#dde3ea")
)

// applicationList builds a frame like the register's, plus hiddenRows table rows (5 texts each)
// scrolled outside the table's clip.
func applicationList(hiddenRows int) *draw2d.List {
	var list draw2d.List
	texts := []string{"Lighthouses", "North Point", "Rocky island, north shore", "1850", "20 nm", "Lit", "Sort by name", "All", "Export"}
	icons := []string{"lamp", "save", "undo-2", "redo-2", "sun", "moon", "radar", "circle", "settings"}
	for index := range 17 { // separators and bars
		list.Rectangle(draw2d.Rectangle{X: float32(index * 70), Y: 0, Width: 1280, Height: 1}, 0, applicationLine, draw2d.Transparent, 0)
	}
	for index := range 20 { // controls
		list.Rectangle(draw2d.Rectangle{X: float32(20 + index*50), Y: 100, Width: 120, Height: 30}, 6, applicationSurface, applicationLine, 1)
	}
	for index := range 10 { // cards and chips
		list.Rectangle(draw2d.Rectangle{X: float32(20 + index*90), Y: 200, Width: 220, Height: 110}, 10, applicationSurface, draw2d.Transparent, 0)
	}
	for index := range 27 {
		list.Icon(icons[index%len(icons)], float32(10+index*40), 40, 16, applicationInk)
	}
	for index := range 110 { // visible texts, half of them in clipped table cells
		clipped := index%2 == 0
		if clipped {
			list.PushClip(draw2d.Rectangle{X: 240, Y: 300, Width: 700, Height: 380})
		}
		list.Text(texts[index%len(texts)], float32(20+index%10*100), float32(320+index/10*30), 13, draw2d.Weight(index%3), applicationInk)
		if clipped {
			list.PopClip()
		}
	}
	list.PushClip(draw2d.Rectangle{X: 240, Y: 300, Width: 700, Height: 380})
	for row := range hiddenRows { // each cell clips to its own box inside the table's clip
		y := float32(700 + row*40)
		for column := range 5 {
			list.PushClip(draw2d.Rectangle{X: float32(256 + column*130), Y: y, Width: 120, Height: 40})
			list.Text(fmt.Sprintf("Lighthouse %d", row), float32(256+column*130), y+25, 13, draw2d.Regular, applicationInk)
			list.PopClip()
		}
	}
	list.PopClip()
	return &list
}

func BenchmarkPaintingAnApplicationFrame(benchmark *testing.B) {
	for _, hiddenRows := range []int{0, 1000} {
		benchmark.Run(fmt.Sprintf("hiddenRows=%d", hiddenRows), func(benchmark *testing.B) {
			painter, err := NewPainter()
			if err != nil {
				benchmark.Fatal(err)
			}
			list := applicationList(hiddenRows)
			layer := &scene.Overlay{Drawing: list}
			frame := Frame{Width: 2560, Height: 1440, Scale: 2}
			quads := painter.Paint(layer, nil, frame)
			benchmark.ReportAllocs()
			benchmark.ResetTimer()
			for range benchmark.N {
				quads = painter.Paint(layer, nil, frame)
			}
			benchmark.StopTimer()
			benchmark.ReportMetric(float64(len(list.Commands)), "commands/frame")
			benchmark.ReportMetric(float64(len(quads)), "quads/frame")
			benchmark.ReportMetric(float64(len(quads)*6*8*4), "vertexbytes/frame")
		})
	}
}

// BenchmarkSortingAnApplicationFrameByLayer is the per-frame stable sort of the commands by layer, even
// when every command is already in layer order.
func BenchmarkSortingAnApplicationFrameByLayer(benchmark *testing.B) {
	for _, hiddenRows := range []int{0, 1000} {
		benchmark.Run(fmt.Sprintf("hiddenRows=%d", hiddenRows), func(benchmark *testing.B) {
			painter, err := NewPainter()
			if err != nil {
				benchmark.Fatal(err)
			}
			list := applicationList(hiddenRows)
			benchmark.ReportAllocs()
			benchmark.ResetTimer()
			for range benchmark.N {
				painter.drawingOrder(list)
			}
		})
	}
}

// BenchmarkPaintingALineOfText is painting one 12-character line: per rune a glyph presence
// check through the font (drawable), an atlas map lookup and a quad.
func BenchmarkPaintingALineOfText(benchmark *testing.B) {
	painter, err := NewPainter()
	if err != nil {
		benchmark.Fatal(err)
	}
	command := draw2d.Command{Kind: draw2d.TextKind, Text: "North Point!", Size: 13, Baseline: 30, Fill: applicationInk}
	painter.paintText(&command, 2)
	benchmark.ReportAllocs()
	benchmark.ResetTimer()
	for range benchmark.N {
		painter.quads = painter.quads[:0]
		painter.paintText(&command, 2)
	}
}

// BenchmarkFindingAGlyph isolates the per-rune font lookup drawable does.
func BenchmarkFindingAGlyph(benchmark *testing.B) {
	painter, err := NewPainter()
	if err != nil {
		benchmark.Fatal(err)
	}
	benchmark.ReportAllocs()
	for index := range benchmark.N {
		painter.fonts.hasGlyph(regularFace, 26, rune('a'+index%26))
	}
}

// TestAnApplicationFrameSaysWhetherItIsInLayerOrder logs whether a go-ui-like list is already in layer order, in
// which case the sort could be skipped with a linear check.
func TestAnApplicationFrameSaysWhetherItIsInLayerOrder(test *testing.T) {
	list := applicationList(10)
	sorted := sort.SliceIsSorted(list.Commands, func(first, second int) bool {
		return list.Commands[first].Layer < list.Commands[second].Layer
	})
	test.Logf("%d commands, already in layer order: %v", len(list.Commands), sorted)
}
