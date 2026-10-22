package overlay

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/scene"
)

var white = draw2d.Hex("#ffffff")

func paintList(test *testing.T, list *draw2d.List, scale float32) []Quad {
	test.Helper()
	painter := newTestPainter(test)
	return painter.Paint(&scene.Overlay{Drawing: list}, nil, Frame{Width: 800, Height: 600, Scale: scale})
}

// covered returns the smallest rectangle holding every quad.
func covered(quads []Quad) [4]float32 {
	bounds := [4]float32{math.MaxFloat32, math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
	for _, quad := range quads {
		bounds[0], bounds[1] = min(bounds[0], quad.Screen[0]), min(bounds[1], quad.Screen[1])
		bounds[2], bounds[3] = max(bounds[2], quad.Screen[2]), max(bounds[3], quad.Screen[3])
	}
	return bounds
}

func TestMeasuredTextIsPaintedText(test *testing.T) {
	measurer, err := NewMeasurer()
	if err != nil {
		test.Fatal(err)
	}
	for _, scale := range []float32{1, 2} {
		for _, weight := range []draw2d.Weight{draw2d.Regular, draw2d.Medium, draw2d.Bold} {
			text := "Faro del Norte · 18 mn"
			metrics := measurer.MeasureText(text, 13, weight, scale)
			var list draw2d.List
			list.Text(text, 10, 30, 13, weight, white)
			bounds := covered(paintList(test, &list, scale))
			painted := (bounds[2] - bounds[0]) / scale
			// the painted ink is a little narrower than the advances (side bearings), never wider
			if painted > metrics.Width+1 || painted < metrics.Width-3 {
				test.Fatalf("scale %v weight %v: measured %.1f, painted %.1f", scale, weight, metrics.Width, painted)
			}
			if metrics.Ascent <= 0 || metrics.Descent <= 0 {
				test.Fatalf("a line has an ascent and a descent: %+v", metrics)
			}
		}
	}
}

func TestPlainRectangleFillsItsArea(test *testing.T) {
	var list draw2d.List
	list.Rectangle(draw2d.Rect{X: 10, Y: 20, Width: 100, Height: 30}, 0, white, draw2d.Transparent, 0)
	quads := paintList(test, &list, 2)
	if bounds := covered(quads); bounds != [4]float32{20, 40, 220, 100} {
		test.Fatalf("the rectangle covers its area at twice the density, got %v", bounds)
	}
}

func TestRoundedRectangleWithBorder(test *testing.T) {
	var list draw2d.List
	list.Rectangle(draw2d.Rect{X: 0, Y: 0, Width: 80, Height: 28}, 6, white, draw2d.Hex("#dde3ea"), 1)
	quads := paintList(test, &list, 1)
	if len(quads) != 9+8 {
		test.Fatalf("a fill is nine patches and a border eight, got %d quads", len(quads))
	}
	if bounds := covered(quads); bounds != [4]float32{0, 0, 80, 28} {
		test.Fatalf("got %v", bounds)
	}
}

func TestClipCutsQuadsAndTheirAtlas(test *testing.T) {
	var list draw2d.List
	list.PushClip(draw2d.Rect{X: 0, Y: 0, Width: 50, Height: 50})
	list.Rectangle(draw2d.Rect{X: 25, Y: 25, Width: 100, Height: 100}, 0, white, draw2d.Transparent, 0)
	list.PopClip()
	quads := paintList(test, &list, 1)
	if bounds := covered(quads); bounds != [4]float32{25, 25, 50, 50} {
		test.Fatalf("the rectangle is cut to the clip, got %v", bounds)
	}
	quad := Quad{Screen: [4]float32{0, 0, 10, 10}, Atlas: [4]float32{100, 100, 110, 110}}
	cut, visible := clipQuad(quad, rectangle{5, 0, 20, 20})
	if !visible || cut.Screen[0] != 5 || cut.Atlas[0] != 105 {
		test.Fatalf("the atlas moves with the edge: %+v", cut)
	}
	if _, visible := clipQuad(quad, rectangle{20, 20, 30, 30}); visible {
		test.Fatal("a quad outside the clip is dropped")
	}
}

func TestHigherLayersAreDrawnLater(test *testing.T) {
	var list draw2d.List
	list.SetLayer(1)
	list.Rectangle(draw2d.Rect{X: 0, Y: 0, Width: 10, Height: 10}, 0, draw2d.Hex("#ff0000"), draw2d.Transparent, 0)
	list.SetLayer(0)
	list.Rectangle(draw2d.Rect{X: 0, Y: 0, Width: 10, Height: 10}, 0, draw2d.Hex("#0000ff"), draw2d.Transparent, 0)
	quads := paintList(test, &list, 1)
	if last := quads[len(quads)-1]; last.Color[0] != 1 {
		test.Fatalf("the red rectangle is on the higher layer, so it is drawn last: %v", last.Color)
	}
}

func TestIconsAreDrawnByName(test *testing.T) {
	var list draw2d.List
	list.Icon("anchor", 10, 10, 16, white)
	quads := paintList(test, &list, 1)
	if len(quads) != 1 {
		test.Fatalf("an icon is one glyph, got %d quads", len(quads))
	}
	bounds := covered(quads)
	if bounds[0] < 10 || bounds[1] < 10 || bounds[2] > 26 || bounds[3] > 26 {
		test.Fatalf("the icon stays inside its square: %v", bounds)
	}
}
