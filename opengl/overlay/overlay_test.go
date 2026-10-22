package overlay

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay/lucide"
	"github.com/akrck02/go-renderer/scene"
	"golang.org/x/image/font/gofont/gomonobold"
)

func newTestPainter(test testing.TB) *Painter {
	test.Helper()
	painter, err := NewPainter(Fonts{})
	if err != nil {
		test.Fatal(err)
	}
	return painter
}

func TestAtlasHoldsTheCommonCharacters(test *testing.T) {
	painter := newTestPainter(test)
	painter.prebuild(textFace, 13)
	for _, character := range "Aaz09¿¡áéíóúñüÁÉÍÓÚÑ·→" {
		entry := painter.glyph(textFace, 13, character)
		if entry.Advance <= 0 || entry.Bounds.Empty() {
			test.Fatalf("%q should be in the atlas: %+v", character, entry)
		}
	}
	if space := painter.glyph(textFace, 13, ' '); space.Advance <= 0 || !space.Bounds.Empty() {
		test.Fatalf("a space moves the pen and draws nothing: %+v", space)
	}
}

func TestAtlasGrowsAndKeepsItsPictures(test *testing.T) {
	painter := newTestPainter(test)
	first := painter.glyph(textFace, 40, 'W')
	before := painter.atlas.Image.Pix[first.Bounds.Min.Y*painter.atlas.Image.Stride:][:painter.atlas.Image.Stride]
	saved := append([]byte{}, before...)
	for size := 41; painter.atlas.Size() == initialAtlasSize; size++ {
		painter.prebuild(textFace, size)
	}
	again := painter.glyph(textFace, 40, 'W')
	if again.Bounds != first.Bounds {
		test.Fatalf("pictures never move")
	}
	row := painter.atlas.Image.Pix[first.Bounds.Min.Y*painter.atlas.Image.Stride:]
	for column := first.Bounds.Min.X; column < first.Bounds.Max.X; column++ {
		if row[column] != saved[column] {
			test.Fatalf("growing the atlas changed a picture")
		}
	}
}

func TestIconsAreFoundByName(test *testing.T) {
	for _, name := range []string{"anchor", "sailboat", "lighthouse", "clock", "triangle-alert"} {
		if _, found := lucide.Codepoint(name); !found {
			test.Fatalf("no icon %q", name)
		}
	}
	painter := newTestPainter(test)
	character := painter.iconCharacter("anchor")
	if entry := painter.glyph(iconFace, 16, character); entry.Bounds.Empty() {
		test.Fatalf("the anchor icon should draw something")
	}
}

func TestUnknownIconsWarnOnceAndDrawNothing(test *testing.T) {
	painter := newTestPainter(test)
	var warnings []string
	painter.Warn = func(message string) { warnings = append(warnings, message) }
	overlay := &scene.Overlay{}
	overlay.AddLabel(&scene.Label{Icon: "sea-serpent", Point: graphics.Vector{0, 0, 0, 1}})
	frame := testFrame()
	for range 3 {
		if quads := painter.Paint(overlay, nil, frame); len(quads) != 0 {
			test.Fatalf("an unknown icon alone draws nothing, got %d quads", len(quads))
		}
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "sea-serpent") {
		test.Fatalf("expected one warning, got %v", warnings)
	}
}

func TestWithoutWarnAProblemGoesToTheDefaultLogger(test *testing.T) {
	var logged bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logged, nil)))
	defer slog.SetDefault(previous)
	painter := newTestPainter(test)
	painter.Report("the lights are off")
	if !strings.Contains(logged.String(), "level=WARN") || !strings.Contains(logged.String(), "the lights are off") {
		test.Fatalf("logged %q", logged.String())
	}
}

// testFrame looks at the origin from 10 units south, on an 800 × 600 screen.
func testFrame() Frame {
	projection := graphics.PerspectiveOpenGL(0.8, 800.0/600.0, 0.1, 100)
	view := graphics.LookAt(graphics.Vector{0, 0, 10, 1}, graphics.Vector{0, 0, 0, 1}, graphics.Vector{0, 1, 0, 0})
	return Frame{ViewProjection: projection.Multiply(view), Width: 800, Height: 600, Scale: 1}
}

func TestAPointIsProjectedOntoTheScreen(test *testing.T) {
	frame := testFrame()
	centre := Project(graphics.Vector{0, 0, 0, 1}, frame)
	if !centre.Visible || abs(centre.X-400) > 0.5 || abs(centre.Y-300) > 0.5 {
		test.Fatalf("the target falls in the middle of the screen: %+v", centre)
	}
	above := Project(graphics.Vector{0, 1, 0, 1}, frame)
	if above.Y >= centre.Y {
		test.Fatalf("up in the world is up on screen")
	}
	if behind := Project(graphics.Vector{0, 0, 20, 1}, frame); behind.Visible {
		test.Fatalf("points behind the camera are hidden")
	}
	if aside := Project(graphics.Vector{100, 0, 0, 1}, frame); aside.Visible {
		test.Fatalf("points off screen are hidden")
	}
	frame.VerticalScale = 2
	if exaggerated := Project(graphics.Vector{0, 1, 0, 1}, frame); exaggerated.Y >= above.Y {
		test.Fatalf("the vertical exaggeration lifts the point")
	}
}

func TestLabelsSitOverTheirAnchor(test *testing.T) {
	painter := newTestPainter(test)
	boat := &scene.Node{Name: "boat"}
	overlay := &scene.Overlay{}
	overlay.AddLabel(&scene.Label{Node: boat, Text: "Boat", Icon: "sailboat"})
	anchors := map[*scene.Node]graphics.Matrix{boat: graphics.Translate(graphics.Vector{0, 0, 0, 0})}
	area := boundsOf(painter.Paint(overlay, anchors, testFrame()))
	if abs((area[0]+area[2])/2-400) > 2 {
		test.Fatalf("the label is centred on its anchor: %v", area)
	}
	if area[3] > 300 || area[3] < 300-labelGap-shadowBlur-shadowDrop-1 {
		test.Fatalf("the label stands just above its anchor: %v", area)
	}
	if quads := painter.Paint(overlay, nil, testFrame()); len(quads) != 0 {
		test.Fatalf("a label whose node is not in the scene is not drawn")
	}
}

func TestLabelsKeepTheirSizeOnScreen(test *testing.T) {
	painter := newTestPainter(test)
	near := &scene.Overlay{Labels: []*scene.Label{{Text: "Quay", Point: graphics.Vector{0, 0, 5, 1}}}}
	far := &scene.Overlay{Labels: []*scene.Label{{Text: "Quay", Point: graphics.Vector{0, 0, -50, 1}}}}
	nearArea, farArea := boundsOf(painter.Paint(near, nil, testFrame())), boundsOf(painter.Paint(far, nil, testFrame()))
	if nearArea[2]-nearArea[0] != farArea[2]-farArea[0] || nearArea[3]-nearArea[1] != farArea[3]-farArea[1] {
		test.Fatalf("near %v and far %v should be the same size", nearArea, farArea)
	}
	frame := testFrame()
	frame.Scale = 2
	doubled := boundsOf(painter.Paint(near, nil, frame))
	if width, doubledWidth := nearArea[2]-nearArea[0], doubled[2]-doubled[0]; abs(doubledWidth-2*width) > 6 {
		test.Fatalf("at scale 2 the label is twice as wide: %v then %v", width, doubledWidth)
	}
}

func TestAPillIsSizedToItsText(test *testing.T) {
	painter := newTestPainter(test)
	iconOnly := painter.measurePill(pillContent{icon: "anchor"}, labelStyle, 1)
	if iconOnly.width != iconOnly.height || iconOnly.height != 24 {
		test.Fatalf("an icon alone sits in a circle: %+v", iconOnly)
	}
	short := painter.measurePill(pillContent{text: "Quay"}, labelStyle, 1)
	long := painter.measurePill(pillContent{text: "Quay of the old harbour"}, labelStyle, 1)
	if long.width <= short.width || short.height != long.height {
		test.Fatalf("longer texts make wider pills of the same height")
	}
	withIcon := painter.measurePill(pillContent{text: "Quay", icon: "anchor"}, labelStyle, 1)
	if withIcon.width != short.width+15+5 {
		test.Fatalf("an icon adds its size and a gap: %d vs %d", withIcon.width, short.width)
	}
	tooLong := painter.measurePill(pillContent{text: strings.Repeat("harbour ", 40)}, labelStyle, 1)
	if tooLong.content.textWidth > int(labelStyle.maximumTextWidth) || tooLong.content.text[len(tooLong.content.text)-1] != '…' {
		test.Fatalf("long texts are cut with an ellipsis")
	}
	if empty := painter.measurePill(pillContent{}, labelStyle, 1); empty.width != 0 {
		test.Fatalf("an empty pill has no size")
	}
}

func TestOverlappingLabelsGiveWayToTheNearest(test *testing.T) {
	painter := newTestPainter(test)
	near := &scene.Label{Text: "Near boat", Point: graphics.Vector{0, 0, 2, 1}}
	far := &scene.Label{Text: "Far boat", Point: graphics.Vector{0.01, 0, -2, 1}}
	overlay := &scene.Overlay{Labels: []*scene.Label{far, near}}
	painter.Paint(overlay, nil, testFrame())
	if len(painter.occupied) != 1 {
		test.Fatalf("only one of two overlapping labels is drawn, got %d", len(painter.occupied))
	}
	alone := &scene.Overlay{Labels: []*scene.Label{near}}
	if count, aloneCount := len(painter.Paint(overlay, nil, testFrame())), len(painter.Paint(alone, nil, testFrame())); count != aloneCount {
		test.Fatalf("the nearer label is the one kept")
	}
}

func TestHeaderAndNoticesStackInTheCorner(test *testing.T) {
	painter := newTestPainter(test)
	overlay := &scene.Overlay{Header: []scene.HeaderLine{{Icon: "clock", Text: "12:00"}, {Text: "10 boats"}}}
	overlay.AddNotice("anchor", "A boat came in", graphics.Vector{})
	painter.Paint(overlay, nil, testFrame())
	if len(painter.occupied) != 2 {
		test.Fatalf("a header card and a notice, got %d areas", len(painter.occupied))
	}
	header, notice := painter.occupied[0], painter.occupied[1]
	if header.left != screenMargin || header.top != screenMargin || notice.top <= header.bottom || notice.left != header.left {
		test.Fatalf("the notice goes under the header: %+v %+v", header, notice)
	}
	if header.height() != 2*headerRowHeight+2*headerPadding {
		test.Fatalf("two rows in the header: %+v", header)
	}
}

func TestFadingNoticesAreTranslucent(test *testing.T) {
	painter := newTestPainter(test)
	overlay := &scene.Overlay{NoticeDuration: 2}
	overlay.AddNotice("", "A boat came in", graphics.Vector{})
	overlay.Advance(1.5)
	for _, quad := range painter.Paint(overlay, nil, testFrame()) {
		if quad.Color[3] > 0.5 {
			test.Fatalf("half faded, alpha %v", quad.Color[3])
		}
	}
}

func boundsOf(quads []Quad) [4]float32 {
	if len(quads) == 0 {
		return [4]float32{}
	}
	area := quads[0].Screen
	for _, quad := range quads[1:] {
		area[0], area[1] = min(area[0], quad.Screen[0]), min(area[1], quad.Screen[1])
		area[2], area[3] = max(area[2], quad.Screen[2]), max(area[3], quad.Screen[3])
	}
	return area
}

func abs(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

// A program can draw text in its own fonts, and codes in a monospaced one.
func TestTextCanUseAProgramsFontsAndAMonospacedOneForCodes(test *testing.T) {
	measurer, err := NewMeasurer(Fonts{})
	if err != nil {
		test.Fatal(err)
	}
	narrow := measurer.MeasureText("iiii", 14, draw2d.Mono, 1)
	wide := measurer.MeasureText("MMMM", 14, draw2d.Mono, 1)
	if narrow.Width != wide.Width {
		test.Errorf("monospaced codes: iiii %v wide, MMMM %v", narrow.Width, wide.Width)
	}
	chosen, err := NewMeasurer(Fonts{Regular: gomonobold.TTF})
	if err != nil {
		test.Fatal(err)
	}
	if text := chosen.MeasureText("iiii", 14, draw2d.Regular, 1); text.Width != narrow.Width {
		test.Errorf("the chosen regular font draws the text: %v, want %v", text.Width, narrow.Width)
	}
}
