package draw2d

import (
	"hash/maphash"
	"strings"
	"testing"
)

func TestHexReadsColours(t *testing.T) {
	color := Hex("#ff8000")
	if color != (Color{1, 128.0 / 255, 0, 1}) {
		t.Fatalf("got %v", color)
	}
	if Hex("#00000080")[3] != 128.0/255 {
		t.Fatal("the alpha digits should be read")
	}
	if _, err := ParseHex("orange"); err == nil {
		t.Fatal("a name is not a hex colour")
	}
}

func TestRectIntersect(t *testing.T) {
	first := Rect{X: 0, Y: 0, Width: 10, Height: 10}
	second := Rect{X: 5, Y: 6, Width: 10, Height: 10}
	if shared := first.Intersect(second); shared != (Rect{X: 5, Y: 6, Width: 5, Height: 4}) {
		t.Fatalf("got %v", shared)
	}
	if !first.Intersect(Rect{X: 20, Y: 20, Width: 1, Height: 1}).IsEmpty() {
		t.Fatal("rectangles apart share nothing")
	}
}

func TestClipsNest(t *testing.T) {
	var list List
	list.PushClip(Rect{X: 0, Y: 0, Width: 100, Height: 100})
	list.PushClip(Rect{X: 50, Y: 50, Width: 100, Height: 100})
	list.Rectangle(Rect{X: 60, Y: 60, Width: 10, Height: 10}, 0, Hex("#ffffff"), Transparent, 0)
	list.PopClip()
	list.PopClip()
	list.Rectangle(Rect{X: 0, Y: 0, Width: 10, Height: 10}, 0, Hex("#ffffff"), Transparent, 0)

	if clip := list.Commands[0].Clip; clip != (Rect{X: 50, Y: 50, Width: 50, Height: 50}) {
		t.Fatalf("the inner clip should be inside the outer one, got %v", clip)
	}
	if list.Commands[1].HasClip {
		t.Fatal("after popping every clip nothing is clipped")
	}
}

func TestClippedAwayShapesAreDropped(t *testing.T) {
	var list List
	list.PushClip(Rect{X: 0, Y: 0, Width: 10, Height: 10})
	list.Rectangle(Rect{X: 20, Y: 20, Width: 5, Height: 5}, 0, Hex("#ffffff"), Transparent, 0)
	if !list.IsEmpty() {
		t.Fatal("a rectangle outside the clip draws nothing")
	}
}

func TestInvisibleShapesAreSkipped(t *testing.T) {
	var list List
	list.Rectangle(Rect{Width: 10, Height: 10}, 4, Transparent, Transparent, 1)
	list.Text("", 0, 10, 13, Regular, Hex("#000000"))
	list.Icon("anchor", 0, 0, 16, Transparent)
	if !list.IsEmpty() {
		t.Fatalf("nothing visible was added, got %d commands", len(list.Commands))
	}
}

func TestLayersAreStamped(t *testing.T) {
	var list List
	list.SetLayer(2)
	list.Icon("anchor", 0, 0, 16, Hex("#000000"))
	if list.Commands[0].Layer != 2 {
		t.Fatal("commands take the current layer")
	}
	list.Reset()
	if !list.IsEmpty() || list.layer != 0 {
		t.Fatal("reset empties the list and returns to layer 0")
	}
}

func TestTextsOutOfTheClipAreLeftOut(t *testing.T) {
	var list List
	list.PushClip(Rect{X: 0, Y: 100, Width: 200, Height: 100})
	list.Text("above", 10, 90, 12, Regular, Hex("#000000")) // its line ends at 94
	list.Text("inside", 10, 150, 12, Regular, Hex("#000000"))
	list.Text("half in", 10, 104, 12, Regular, Hex("#000000")) // its top is clipped, the rest shows
	list.Text("below", 10, 220, 12, Regular, Hex("#000000"))
	list.Text("right", 210, 150, 12, Regular, Hex("#000000"))
	list.Text("from the left", -50, 150, 12, Regular, Hex("#000000")) // may run into the clip
	var kept []string
	for _, command := range list.Commands {
		kept = append(kept, command.Text)
	}
	if strings.Join(kept, ",") != "inside,half in,from the left" {
		t.Fatalf("kept %v", kept)
	}
}

func TestFingerprintChangesWithWhatIsDrawn(t *testing.T) {
	seed := maphash.MakeSeed()
	build := func(text string, fill Color) *List {
		var list List
		list.Rectangle(Rect{X: 1, Y: 2, Width: 30, Height: 40}, 4, fill, Color{}, 0)
		list.Text(text, 5, 20, 12, Medium, Hex("#000000"))
		return &list
	}
	same := build("Faro", Hex("#ffffff")).Fingerprint(seed)
	if build("Faro", Hex("#ffffff")).Fingerprint(seed) != same {
		t.Fatal("the same picture, the same fingerprint")
	}
	if build("Faros", Hex("#ffffff")).Fingerprint(seed) == same || build("Faro", Hex("#fffffe")).Fingerprint(seed) == same {
		t.Fatal("a different text or colour changes it")
	}
	var empty List
	if empty.Fingerprint(seed) == same {
		t.Fatal("an empty list differs")
	}
}

func TestLightsAreAddedAsCommands(t *testing.T) {
	var list List
	picture := NewBlankImage(4, 4)
	list.PointLight(100, 50, 10, 40, Color{1, 0.9, 0.6, 1.5})
	list.SpriteLight(picture, Rect{X: 0, Y: 0, Width: 20, Height: 10}, Color{1, 1, 1, 1})
	list.Lighting(Color{1, 1, 1, 0.72})
	if len(list.Commands) != 3 {
		t.Fatalf("three light commands made %d", len(list.Commands))
	}
	point := list.Commands[0]
	if point.Kind != PointLightKind || point.Area != (Rect{X: 60, Y: 10, Width: 80, Height: 80}) || point.Radius != 10 || point.Fill[3] != 1.5 {
		t.Fatalf("the point light is %+v", point)
	}
	if list.Commands[1].Kind != SpriteLightKind || list.Commands[1].Picture != picture {
		t.Fatalf("the sprite light is %+v", list.Commands[1])
	}
	if lighting := list.Commands[2]; lighting.Kind != LightingKind || lighting.Fill[3] != 0.72 {
		t.Fatalf("the lighting is %+v", lighting)
	}
}

func TestLightsThatCannotShineAreLeftOut(t *testing.T) {
	var list List
	list.PointLight(0, 0, 0, 0, Color{1, 1, 1, 1})
	list.PointLight(0, 0, 0, 10, Color{1, 1, 1, 0})
	list.SpriteLight(nil, Rect{Width: 10, Height: 10}, Color{1, 1, 1, 1})
	if len(list.Commands) != 0 {
		t.Fatalf("lights with no size, no intensity or no picture made %d commands", len(list.Commands))
	}
}

func TestALightingUnderAClipLightsOnlyTheClip(t *testing.T) {
	var list List
	clip := Rect{X: 100, Y: 100, Width: 10, Height: 10}
	list.PushClip(clip)
	list.Lighting(Color{1, 1, 1, 1})
	list.PopClip()
	if len(list.Commands) != 1 || !list.Commands[0].HasClip || list.Commands[0].Clip != clip {
		t.Fatalf("a lighting under a clip became %+v", list.Commands)
	}
}
