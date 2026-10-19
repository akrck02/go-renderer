package overlay

import (
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/scene"
)

// paintWithSnapshots paints a list and gives its quads and the snapshots the backend must take.
func paintWithSnapshots(t *testing.T, list *draw2d.List, scale float32) ([]Quad, []Snapshot) {
	t.Helper()
	painter := newTestPainter(t)
	quads := painter.Paint(&scene.Overlay{Drawing: list}, nil, Frame{Width: 800, Height: 600, Scale: scale})
	return quads, painter.Snapshots()
}

func TestAListWithoutBackdropsAsksForNoSnapshot(t *testing.T) {
	var list draw2d.List
	list.Rectangle(draw2d.Rect{X: 10, Y: 10, Width: 100, Height: 50}, 8, white, draw2d.Transparent, 0)
	quads, snapshots := paintWithSnapshots(t, &list, 1)
	if len(snapshots) != 0 {
		t.Fatalf("nothing frosted, no snapshot: got %v", snapshots)
	}
	for _, quad := range quads {
		if quad.Backdrop {
			t.Fatal("no quad reads a backdrop")
		}
	}
}

func TestABackdropIsARoundedFillThatReadsTheSnapshot(t *testing.T) {
	var list draw2d.List
	area := draw2d.Rect{X: 20, Y: 30, Width: 200, Height: 100}
	list.Backdrop(area, 12, 24)
	quads, snapshots := paintWithSnapshots(t, &list, 1)
	if len(quads) == 0 {
		t.Fatal("a backdrop is laid out")
	}
	for _, quad := range quads {
		if !quad.Backdrop {
			t.Fatal("every quad of a backdrop reads the snapshot")
		}
	}
	if bounds := covered(quads); bounds != [4]float32{20, 30, 220, 130} {
		t.Fatalf("a backdrop covers its area, got %v", bounds)
	}
	if len(snapshots) != 1 || snapshots[0].At != 0 || snapshots[0].Blur != 24 {
		t.Fatalf("one snapshot, taken before it, with its blur: got %v", snapshots)
	}
}

func TestTheBackdropsOfALayerShareOneSnapshotTakenBeforeTheFirst(t *testing.T) {
	var list draw2d.List
	list.Rectangle(draw2d.Rect{X: 0, Y: 0, Width: 800, Height: 600}, 0, white, draw2d.Transparent, 0)
	for row := float32(0); row < 3; row++ {
		panel := draw2d.Rect{X: 20, Y: 20 + row*120, Width: 300, Height: 100}
		list.Backdrop(panel, 12, 16+row*4)
		list.Rectangle(panel, 12, draw2d.Hex("#ffffff20"), draw2d.Transparent, 0)
	}
	quads, snapshots := paintWithSnapshots(t, &list, 2)
	if len(snapshots) != 1 {
		t.Fatalf("three panels of one layer cost one snapshot, got %d", len(snapshots))
	}
	if !quads[snapshots[0].At].Backdrop || quads[snapshots[0].At-1].Backdrop {
		t.Fatal("the snapshot is taken after the ground and just before the first backdrop")
	}
	if snapshots[0].Blur != 24*2 {
		t.Fatalf("the snapshot takes the widest blur of its layer, in screen pixels: got %v", snapshots[0].Blur)
	}
}

func TestEachLayerWithBackdropsTakesItsOwnSnapshot(t *testing.T) {
	var list draw2d.List
	list.Backdrop(draw2d.Rect{X: 10, Y: 10, Width: 100, Height: 100}, 8, 24)
	list.SetLayer(1) // a dialog over the page: its glass blurs the page, panels included
	list.Backdrop(draw2d.Rect{X: 50, Y: 50, Width: 100, Height: 100}, 8, 24)
	_, snapshots := paintWithSnapshots(t, &list, 1)
	if len(snapshots) != 2 || snapshots[1].At <= snapshots[0].At {
		t.Fatalf("a layer over another takes a snapshot of its own, later: got %v", snapshots)
	}
}

func TestABackdropWithoutBlurIsLeftOut(t *testing.T) {
	var list draw2d.List
	list.Backdrop(draw2d.Rect{X: 10, Y: 10, Width: 100, Height: 100}, 8, 0)
	if !list.IsEmpty() {
		t.Fatal("no blur, no backdrop: a system asking for less transparency costs nothing")
	}
}
