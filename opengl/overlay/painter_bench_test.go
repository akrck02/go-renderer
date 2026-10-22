package overlay

// Benchmarks of the 2D drawing path (draw2d.List -> Painter -> quads -> vertices). They need
// no OpenGL context: the GL-side loop of opengl/overlay.go fillVertices is reproduced here verbatim
// (minus the gl.BufferData call) so its CPU cost can be measured.
//
//	go test ./opengl/overlay -run xxx -bench . -benchmem

import (
	"fmt"
	"hash/maphash"
	"testing"
	"unsafe"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// busyEditorList builds a busy editor-like interface: panels with a shadow, a bordered body, a title,
// a clip, and rows of label + bordered button + button text + icon, scrolled so most rows fall
// outside the clip (texts are kept by draw2d and cut by clipQuads). 8 panels x 100 rows = ~2000 commands.
func busyEditorList(list *draw2d.List, panels, rows int) {
	list.Reset()
	ink, surface, line := draw2d.Hex("#18212e"), draw2d.Hex("#ffffff"), draw2d.Hex("#18212e22")
	accent := draw2d.Hex("#2f6feb")
	for panel := 0; panel < panels; panel++ {
		area := draw2d.Rect{X: 8 + float32(panel)*158, Y: 40, Width: 150, Height: 660}
		list.Shadow(area, 8, 12, 4, draw2d.Hex("#0000003a"))
		list.Rectangle(area, 8, surface, line, 1)
		list.Text(fmt.Sprintf("Panel %d", panel), area.X+12, area.Y+20, 14, draw2d.Bold, ink)
		list.PushClip(area.Inset(4))
		for row := 0; row < rows; row++ {
			y := area.Y + 32 + float32(row)*24 // rows run past the panel bottom: some are clipped away
			list.Text(fmt.Sprintf("Item %d", row), area.X+12, y+16, 13, draw2d.Regular, ink)
			button := draw2d.Rect{X: area.X + 64, Y: y + 2, Width: 78, Height: 20}
			list.Rectangle(button, 4, surface.Mix(accent, 0.08), line, 1)
			list.Text("Button", button.X+22, button.Y+14, 13, draw2d.Medium, ink)
			list.Icon("settings", button.X+4, button.Y+3, 14, accent)
		}
		list.PopClip()
	}
}

const busyWidth, busyHeight, busyScale = 2560, 1440, 2

func busyFrame() Frame { return Frame{Width: busyWidth, Height: busyHeight, Scale: busyScale} }

// fillVerticesCopy is opengl/overlay.go:120-133 without the upload.
func fillVerticesCopy(vertices []float32, quads []Quad) []float32 {
	vertices = vertices[:0]
	for _, quad := range quads {
		left, top, right, bottom := quad.Screen[0], quad.Screen[1], quad.Screen[2], quad.Screen[3]
		atlasLeft, atlasTop, atlasRight, atlasBottom := quad.Atlas[0], quad.Atlas[1], quad.Atlas[2], quad.Atlas[3]
		corners := [6][4]float32{
			{left, top, atlasLeft, atlasTop}, {left, bottom, atlasLeft, atlasBottom}, {right, bottom, atlasRight, atlasBottom},
			{left, top, atlasLeft, atlasTop}, {right, bottom, atlasRight, atlasBottom}, {right, top, atlasRight, atlasTop},
		}
		for _, corner := range corners {
			vertices = append(vertices, corner[0], corner[1], corner[2], corner[3], quad.Color[0], quad.Color[1], quad.Color[2], quad.Color[3])
		}
	}
	return vertices
}

// packedInstance is a candidate compact format: one instance per quad (drawn with
// glDrawArraysInstanced, 4 vertices generated in the vertex shader), 24 bytes instead of 192.
type packedInstance struct {
	screen [4]int16  // pixels (fits 2560x1440 and up to 32767)
	atlas  [4]uint16 // atlas pixels (atlas <= 8192)
	colour [4]uint8  // RGBA8
	_      [4]byte   // pad to 24 for alignment
}

func fillPacked(instances []packedInstance, quads []Quad) []packedInstance {
	instances = instances[:0]
	for _, quad := range quads {
		var instance packedInstance
		for index := range 4 {
			instance.screen[index] = int16(quad.Screen[index])
			instance.atlas[index] = uint16(quad.Atlas[index])
			instance.colour[index] = uint8(quad.Color[index]*255 + 0.5)
		}
		instances = append(instances, instance)
	}
	return instances
}

// TestARepeatedFrameLeavesTheAtlasAlone logs the size of a busy frame: commands, quads, vertex bytes per upload,
// overdraw at 2560x1440 and the atlas state. It asserts nothing beyond a sane run.
func TestARepeatedFrameLeavesTheAtlasAlone(test *testing.T) {
	var list draw2d.List
	busyEditorList(&list, 8, 100)
	painter := newTestPainter(test)
	layer := &scene.Overlay{Drawing: &list}
	painter.Paint(layer, nil, busyFrame()) // warm the atlas
	versionAfterWarm := painter.atlas.Version
	quads := painter.Paint(layer, nil, busyFrame())
	if painter.atlas.Version != versionAfterWarm {
		test.Fatalf("a repeated frame changed the atlas")
	}
	area := 0.0
	for _, quad := range quads {
		area += float64(quad.Screen[2]-quad.Screen[0]) * float64(quad.Screen[3]-quad.Screen[1])
	}
	screen := float64(busyWidth * busyHeight)
	vertexBytes := len(quads) * 6 * floatsPerTestVertex * 4
	test.Logf("commands=%d quads=%d triangles=%d vertexBytes/frame=%d (%.1f KiB) packed=%d B",
		len(list.Commands), len(quads), 2*len(quads), vertexBytes, float64(vertexBytes)/1024, len(quads)*int(unsafe.Sizeof(packedInstance{})))
	test.Logf("blended quad area=%.2f Mpx = %.2fx the %dx%d screen (plus 1x clear)", area/1e6, area/screen, busyWidth, busyHeight)
	test.Logf("atlas %dx%d R8 = %d KiB, entries=%d, version=%d", painter.atlas.Size(), painter.atlas.Size(),
		painter.atlas.Size()*painter.atlas.Size()/1024, len(painter.atlas.entries), painter.atlas.Version)

	// a move to a screen of another density (or the window dragged between displays) asks for new
	// sizes: every glyph, corner and shadow is drawn again into the same, growing atlas
	before, entries := painter.atlas.Version, len(painter.atlas.entries)
	painter.Paint(layer, nil, Frame{Width: busyWidth / 2, Height: busyHeight / 2, Scale: 1})
	test.Logf("scale 2 -> 1: +%d entries, %d atlas version bumps (=> one full TexImage2D of %d KiB on the next drawQuads), atlas now %d",
		len(painter.atlas.entries)-entries, painter.atlas.Version-before, painter.atlas.Size()*painter.atlas.Size()/1024, painter.atlas.Size())
}

const floatsPerTestVertex = 8

// BenchmarkPaintingTwoThousandCommands is the per-frame CPU of turning ~2000 commands into quads, atlas warm
// (the steady state when nothing new is drawn).
func BenchmarkPaintingTwoThousandCommands(benchmark *testing.B) {
	var list draw2d.List
	busyEditorList(&list, 8, 100)
	painter := newTestPainter(benchmark)
	layer := &scene.Overlay{Drawing: &list}
	quads := painter.Paint(layer, nil, busyFrame())
	benchmark.ReportMetric(float64(len(list.Commands)), "commands")
	benchmark.ReportMetric(float64(len(quads)), "quads")
	benchmark.ReportAllocs()
	benchmark.ResetTimer()
	for range benchmark.N {
		painter.Paint(layer, nil, busyFrame())
	}
}

// BenchmarkFillingTheVerticesOfAFrame is the GL-side expansion of the quads into 6 vertices x 8 floats.
func BenchmarkFillingTheVerticesOfAFrame(benchmark *testing.B) {
	var list draw2d.List
	busyEditorList(&list, 8, 100)
	painter := newTestPainter(benchmark)
	quads := painter.Paint(&scene.Overlay{Drawing: &list}, nil, busyFrame())
	vertices := fillVerticesCopy(nil, quads)
	benchmark.SetBytes(int64(len(vertices) * 4))
	benchmark.ReportAllocs()
	benchmark.ResetTimer()
	for range benchmark.N {
		vertices = fillVerticesCopy(vertices, quads)
	}
}

// BenchmarkFillingPackedInstances is the same quads as 24-byte instances.
func BenchmarkFillingPackedInstances(benchmark *testing.B) {
	var list draw2d.List
	busyEditorList(&list, 8, 100)
	painter := newTestPainter(benchmark)
	quads := painter.Paint(&scene.Overlay{Drawing: &list}, nil, busyFrame())
	instances := fillPacked(nil, quads)
	benchmark.SetBytes(int64(len(instances)) * int64(unsafe.Sizeof(packedInstance{})))
	benchmark.ReportAllocs()
	benchmark.ResetTimer()
	for range benchmark.N {
		instances = fillPacked(instances, quads)
	}
}

// BenchmarkSortingCommandsByLayer is the stable sort by layer done every frame (drawing.go:32).
func BenchmarkSortingCommandsByLayer(benchmark *testing.B) {
	var list draw2d.List
	busyEditorList(&list, 8, 100)
	painter := newTestPainter(benchmark)
	benchmark.ReportAllocs()
	benchmark.ResetTimer()
	for range benchmark.N {
		painter.drawingOrder(&list)
	}
}

// BenchmarkClippingQuads cuts 10 000 quads, a third of them crossing the clip edge.
func BenchmarkClippingQuads(benchmark *testing.B) {
	painter := newTestPainter(benchmark)
	source := make([]Quad, 10000)
	for index := range source {
		x := float32(index%100) * 12
		y := float32(index/100) * 12
		source[index] = Quad{Screen: [4]float32{x, y, x + 10, y + 10}, Atlas: [4]float32{0, 0, 10, 10}, Color: [4]float32{1, 1, 1, 1}}
	}
	clip := rectangle{5, 5, 800, 800}
	benchmark.ReportAllocs()
	benchmark.ResetTimer()
	for range benchmark.N {
		painter.quads = append(painter.quads[:0], source...)
		painter.clipQuads(0, clip)
	}
}

// BenchmarkHashingAList measures what a "skip the frame when the list did not change" check
// costs: hashing every command (fixed fields and text) with maphash.
func BenchmarkHashingAList(benchmark *testing.B) {
	var list draw2d.List
	busyEditorList(&list, 8, 100)
	seed := maphash.MakeSeed()
	benchmark.ReportAllocs()
	benchmark.ResetTimer()
	var sink uint64
	for range benchmark.N {
		sink ^= hashList(seed, &list)
	}
	_ = sink
}

func hashList(seed maphash.Seed, list *draw2d.List) uint64 {
	var hash maphash.Hash
	hash.SetSeed(seed)
	for index := range list.Commands {
		command := &list.Commands[index]
		fixed := *command
		fixed.Text = ""
		hash.Write(unsafe.Slice((*byte)(unsafe.Pointer(&fixed)), unsafe.Sizeof(fixed)))
		hash.WriteString(command.Text)
	}
	return hash.Sum64()
}

// BenchmarkPrebuildingAFace is the cold cost of a new (face, size) pair: the ~200 prebuilt
// characters rasterised and copied into the atlas. Every new text size, weight or screen scale
// pays it once.
func BenchmarkPrebuildingAFace(benchmark *testing.B) {
	painter := newTestPainter(benchmark)
	benchmark.ReportAllocs()
	benchmark.ResetTimer()
	for iteration := range benchmark.N {
		if painter.atlas.Size() >= 4096 { // keep inserting into a non-full atlas
			benchmark.StopTimer()
			painter = newTestPainter(benchmark)
			benchmark.StartTimer()
		}
		painter.prebuild(regularFace, 10+iteration%40)
		delete(painter.prebuilt, faceKey{regularFace, 10 + iteration%40})
		// forget the entries so the next iteration rasterises again
		for key := range painter.atlas.entries {
			if key.kind == glyphEntry {
				delete(painter.atlas.entries, key)
			}
		}
	}
	benchmark.ReportMetric(float64(len(PrebuiltCharacters)), "glyphs/op")
}

// BenchmarkRasterisingAGlyph is one glyph rasterised (no atlas copy).
func BenchmarkRasterisingAGlyph(benchmark *testing.B) {
	painter := newTestPainter(benchmark)
	face := painter.fonts.face(regularFace, 26)
	benchmark.ReportAllocs()
	benchmark.ResetTimer()
	for iteration := range benchmark.N {
		rasteriseGlyph(face, rune('A'+iteration%26))
	}
}

// BenchmarkMakingAShadowMask is a shadow corner picture drawn (circleMask with math.Hypot), the
// cost of a new shadow radius/blur pair.
func BenchmarkMakingAShadowMask(benchmark *testing.B) {
	benchmark.ReportAllocs()
	for range benchmark.N {
		radius, blur := 16, 24
		inner, outer := float64(radius-blur), float64(radius+blur)
		circleMask(radius+blur, func(distance float64) float64 { return 1 - smoothStep(inner, outer, distance) })
	}
}

// BenchmarkGrowingTheAtlas is the CPU copy when the atlas doubles 512 -> 1024 -> 2048 (the GPU then
// receives the whole 4 MiB image again).
func BenchmarkGrowingTheAtlas(benchmark *testing.B) {
	benchmark.ReportAllocs()
	for range benchmark.N {
		atlas := NewAtlas() // 512
		atlas.grow()        // 1024
		atlas.grow()        // 2048
	}
}

func TestPaintingAFrameAfterTheFirstAllocatesNothing(test *testing.T) {
	var list draw2d.List
	busyEditorList(&list, 8, 100)
	painter := newTestPainter(test)
	layer := &scene.Overlay{Drawing: &list, Labels: []*scene.Label{
		{Text: "Near boat", Icon: "sailboat", Badge: "2", Point: graphics.Vec4{0, 0, 2, 1}},
		{Text: "Far boat", Point: graphics.Vec4{3, 0, -2, 1}},
	}}
	painter.Paint(layer, nil, busyFrame()) // the first frame fills the atlas
	allocations := testing.AllocsPerRun(20, func() { painter.Paint(layer, nil, busyFrame()) })
	if allocations != 0 {
		test.Fatalf("a frame allocates %v times", allocations)
	}
}
