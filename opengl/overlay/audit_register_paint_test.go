package overlay

// Performance audit of the painter on a draw list shaped like a desktop application's frame (the
// go-ui register example, cmd/app: about 47 rectangles, 110 lines of text of about 9 characters
// and 27 icons, half of them clipped), turned into quads at 2× density, headless. The
// "hiddenRows" variant adds table rows scrolled out of their clip rectangle, which go-ui still
// emits: draw2d culls rectangles and icons that are wholly clipped away, but never text, so the
// painter lays out every glyph and clipQuads throws the quads away. Run with:
//
//	go test ./opengl/overlay -run AuditRegister -bench AuditRegister -benchmem -v

import (
	"fmt"
	"sort"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/scene"
)

var (
	registerInk     = draw2d.Hex("#14202f")
	registerSurface = draw2d.Hex("#ffffff")
	registerLine    = draw2d.Hex("#dde3ea")
)

// registerList builds a frame like the register's, plus hiddenRows table rows (5 texts each)
// scrolled outside the table's clip.
func registerList(hiddenRows int) *draw2d.List {
	var list draw2d.List
	texts := []string{"Lighthouses", "North Point", "Rocky island, north shore", "1850", "20 nm", "Lit", "Sort by name", "All", "Export"}
	icons := []string{"lamp", "save", "undo-2", "redo-2", "sun", "moon", "radar", "circle", "settings"}
	for index := range 17 { // separators and bars
		list.Rectangle(draw2d.Rect{X: float32(index * 70), Y: 0, Width: 1280, Height: 1}, 0, registerLine, draw2d.Transparent, 0)
	}
	for index := range 20 { // controls
		list.Rectangle(draw2d.Rect{X: float32(20 + index*50), Y: 100, Width: 120, Height: 30}, 6, registerSurface, registerLine, 1)
	}
	for index := range 10 { // cards and chips
		list.Rectangle(draw2d.Rect{X: float32(20 + index*90), Y: 200, Width: 220, Height: 110}, 10, registerSurface, draw2d.Transparent, 0)
	}
	for index := range 27 {
		list.Icon(icons[index%len(icons)], float32(10+index*40), 40, 16, registerInk)
	}
	for index := range 110 { // visible texts, half of them in clipped table cells
		clipped := index%2 == 0
		if clipped {
			list.PushClip(draw2d.Rect{X: 240, Y: 300, Width: 700, Height: 380})
		}
		list.Text(texts[index%len(texts)], float32(20+index%10*100), float32(320+index/10*30), 13, draw2d.Weight(index%3), registerInk)
		if clipped {
			list.PopClip()
		}
	}
	list.PushClip(draw2d.Rect{X: 240, Y: 300, Width: 700, Height: 380})
	for row := range hiddenRows { // each cell clips to its own box inside the table's clip
		y := float32(700 + row*40)
		for column := range 5 {
			list.PushClip(draw2d.Rect{X: float32(256 + column*130), Y: y, Width: 120, Height: 40})
			list.Text(fmt.Sprintf("Lighthouse %d", row), float32(256+column*130), y+25, 13, draw2d.Regular, registerInk)
			list.PopClip()
		}
	}
	list.PopClip()
	return &list
}

func BenchmarkAuditRegisterPaint(b *testing.B) {
	for _, hiddenRows := range []int{0, 1000} {
		b.Run(fmt.Sprintf("hiddenRows=%d", hiddenRows), func(b *testing.B) {
			painter, err := NewPainter()
			if err != nil {
				b.Fatal(err)
			}
			list := registerList(hiddenRows)
			layer := &scene.Overlay{Drawing: list}
			frame := Frame{Width: 2560, Height: 1440, Scale: 2}
			quads := painter.Paint(layer, nil, frame)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				quads = painter.Paint(layer, nil, frame)
			}
			b.StopTimer()
			b.ReportMetric(float64(len(list.Commands)), "commands/frame")
			b.ReportMetric(float64(len(quads)), "quads/frame")
			b.ReportMetric(float64(len(quads)*6*8*4), "vertexbytes/frame")
		})
	}
}

// BenchmarkAuditRegisterDrawingOrder is the per-frame stable sort of the commands by layer, even
// when every command is already in layer order.
func BenchmarkAuditRegisterDrawingOrder(b *testing.B) {
	for _, hiddenRows := range []int{0, 1000} {
		b.Run(fmt.Sprintf("hiddenRows=%d", hiddenRows), func(b *testing.B) {
			painter, err := NewPainter()
			if err != nil {
				b.Fatal(err)
			}
			list := registerList(hiddenRows)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				painter.drawingOrder(list)
			}
		})
	}
}

// BenchmarkAuditRegisterTextLine is painting one 12-character line: per rune a glyph presence
// check through the font (drawable), an atlas map lookup and a quad.
func BenchmarkAuditRegisterTextLine(b *testing.B) {
	painter, err := NewPainter()
	if err != nil {
		b.Fatal(err)
	}
	command := draw2d.Command{Kind: draw2d.TextKind, Text: "North Point!", Size: 13, Baseline: 30, Fill: registerInk}
	painter.paintText(&command, 2)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		painter.quads = painter.quads[:0]
		painter.paintText(&command, 2)
	}
}

// BenchmarkAuditRegisterHasGlyph isolates the per-rune font lookup drawable does.
func BenchmarkAuditRegisterHasGlyph(b *testing.B) {
	painter, err := NewPainter()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for index := range b.N {
		painter.fonts.hasGlyph(regularFace, 26, rune('a'+index%26))
	}
}

// TestAuditRegisterLayersAreSorted logs whether a go-ui-like list is already in layer order, in
// which case the sort could be skipped with a linear check.
func TestAuditRegisterLayersAreSorted(t *testing.T) {
	list := registerList(10)
	sorted := sort.SliceIsSorted(list.Commands, func(first, second int) bool {
		return list.Commands[first].Layer < list.Commands[second].Layer
	})
	t.Logf("%d commands, already in layer order: %v", len(list.Commands), sorted)
}
