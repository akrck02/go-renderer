package scene

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
)

func TestNoticesFadeAndGo(test *testing.T) {
	overlay := &Overlay{NoticeDuration: 4}
	overlay.AddNotice("anchor", "first boat in", graphics.Vector{})
	overlay.Advance(1)
	overlay.AddNotice("sailboat", "second boat in", graphics.Vector{})
	visible := overlay.VisibleNotices()
	if len(visible) != 2 || visible[0].Text != "second boat in" || visible[0].Opacity != 1 {
		test.Fatalf("newest first and fully shown: %+v", visible)
	}
	overlay.Advance(2.5) // the first is 3.5 s old: half way through its last second
	visible = overlay.VisibleNotices()
	if visible[1].Opacity < 0.45 || visible[1].Opacity > 0.55 {
		test.Fatalf("the older notice should be fading, opacity %v", visible[1].Opacity)
	}
	overlay.Advance(1)
	if len(overlay.Notices) != 1 || overlay.Notices[0].Text != "second boat in" {
		test.Fatalf("the first notice should be gone: %+v", overlay.Notices)
	}
}

func TestOnlyTheNewestNoticesShow(test *testing.T) {
	overlay := &Overlay{MaximumNotices: 2}
	for _, text := range []string{"one", "two", "three"} {
		overlay.AddNotice("", text, graphics.Vector{})
	}
	visible := overlay.VisibleNotices()
	if len(visible) != 2 || visible[0].Text != "three" || visible[1].Text != "two" {
		test.Fatalf("expected three and two, got %+v", visible)
	}
}

func TestLabelsComeAndGo(test *testing.T) {
	overlay := &Overlay{}
	lighthouse, boat := &Label{Text: "lighthouse"}, &Label{Icon: "sailboat"}
	overlay.AddLabel(lighthouse)
	overlay.AddLabel(boat)
	overlay.RemoveLabel(lighthouse)
	if len(overlay.Labels) != 1 || overlay.Labels[0] != boat {
		test.Fatalf("only the boat should stay")
	}
	if (&Label{}).IsEmpty() == false || boat.IsEmpty() {
		test.Fatalf("a label with an icon is not empty")
	}
}

func TestColorFromHex(test *testing.T) {
	color, valid := ColorFromHex("#ff8000")
	if !valid || color != (graphics.Vector{1, float32(0x80) / 255, 0, 1}) {
		test.Fatalf("got %v", color)
	}
	if short, _ := ColorFromHex("#fff"); short != (graphics.Vector{1, 1, 1, 1}) {
		test.Fatalf("short form: %v", short)
	}
	if _, valid := ColorFromHex("blue"); valid {
		test.Fatalf("names are not colours")
	}
}

func TestWorldTransformsFollowParentsAndHiding(test *testing.T) {
	mast := &Node{Name: "mast", Transform: graphics.Translate(graphics.Vector{0, 2, 0, 0})}
	boat := &Node{Name: "boat", Transform: graphics.Translate(graphics.Vector{10, 0, 0, 0}), Children: []*Node{mast}}
	hidden := &Node{Name: "buoy", Hidden: true}
	harbour := &Scene{Nodes: []*Node{boat, hidden}}
	transforms := harbour.WorldTransforms(map[*Node]bool{mast: true, hidden: true})
	origin := transforms[mast].Apply(graphics.Vector{0, 0, 0, 1})
	if origin[0] != 10 || origin[1] != 2 {
		test.Fatalf("the mast should be at (10, 2), got %v", origin)
	}
	if _, found := transforms[hidden]; found {
		test.Fatalf("hidden nodes have no anchor")
	}
}

func TestMeshTop(test *testing.T) {
	mesh := &Mesh{Positions: []float32{0, 1, 0, 0, 3, 0, 0, -2, 0}}
	if mesh.Top() != 3 || (*Mesh)(nil).Top() != 0 {
		test.Fatalf("top %v", mesh.Top())
	}
}
