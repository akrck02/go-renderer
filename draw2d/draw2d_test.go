package draw2d

import (
	"hash/maphash"
	"strings"
	"testing"
)

func TestMustHexReadsColoursWithAndWithoutAlpha(test *testing.T) {
	color := MustHex("#ff8000")
	if color != (Color{1, 128.0 / 255, 0, 1}) {
		test.Fatalf("got %v", color)
	}
	if MustHex("#00000080")[3] != 128.0/255 {
		test.Fatal("the alpha digits should be read")
	}
	if MustHex("#fff") != (Color{1, 1, 1, 1}) {
		test.Fatal("the short form should be read")
	}
	if _, err := ParseHex("orange"); err == nil {
		test.Fatal("a name is not a hex colour")
	}
}

func TestTwoRectanglesIntersectInTheAreaTheyShare(test *testing.T) {
	first := Rectangle{X: 0, Y: 0, Width: 10, Height: 10}
	second := Rectangle{X: 5, Y: 6, Width: 10, Height: 10}
	if shared := first.Intersect(second); shared != (Rectangle{X: 5, Y: 6, Width: 5, Height: 4}) {
		test.Fatalf("got %v", shared)
	}
	if !first.Intersect(Rectangle{X: 20, Y: 20, Width: 1, Height: 1}).IsEmpty() {
		test.Fatal("rectangles apart share nothing")
	}
}

func TestAClipInsideAClipKeepsOnlyWhatBothShow(test *testing.T) {
	var list List
	list.PushClip(Rectangle{X: 0, Y: 0, Width: 100, Height: 100})
	list.PushClip(Rectangle{X: 50, Y: 50, Width: 100, Height: 100})
	list.Rectangle(Rectangle{X: 60, Y: 60, Width: 10, Height: 10}, 0, MustHex("#ffffff"), Transparent, 0)
	list.PopClip()
	list.PopClip()
	list.Rectangle(Rectangle{X: 0, Y: 0, Width: 10, Height: 10}, 0, MustHex("#ffffff"), Transparent, 0)

	if clip := list.Commands[0].Clip; clip != (Rectangle{X: 50, Y: 50, Width: 50, Height: 50}) {
		test.Fatalf("the inner clip should be inside the outer one, got %v", clip)
	}
	if list.Commands[1].HasClip {
		test.Fatal("after popping every clip nothing is clipped")
	}
}

func TestClippedAwayShapesAreDropped(test *testing.T) {
	var list List
	list.PushClip(Rectangle{X: 0, Y: 0, Width: 10, Height: 10})
	list.Rectangle(Rectangle{X: 20, Y: 20, Width: 5, Height: 5}, 0, MustHex("#ffffff"), Transparent, 0)
	if !list.IsEmpty() {
		test.Fatal("a rectangle outside the clip draws nothing")
	}
}

func TestInvisibleShapesAreSkipped(test *testing.T) {
	var list List
	list.Rectangle(Rectangle{Width: 10, Height: 10}, 4, Transparent, Transparent, 1)
	list.Text("", 0, 10, 13, Regular, MustHex("#000000"))
	list.Icon("anchor", 0, 0, 16, Transparent)
	if !list.IsEmpty() {
		test.Fatalf("nothing visible was added, got %d commands", len(list.Commands))
	}
}

func TestEachCommandIsStampedWithItsLayer(test *testing.T) {
	var list List
	list.SetLayer(2)
	list.Icon("anchor", 0, 0, 16, MustHex("#000000"))
	if list.Commands[0].Layer != 2 {
		test.Fatal("commands take the current layer")
	}
	list.Reset()
	if !list.IsEmpty() || list.layer != 0 {
		test.Fatal("reset empties the list and returns to layer 0")
	}
}

func TestTextsOutOfTheClipAreLeftOut(test *testing.T) {
	var list List
	list.PushClip(Rectangle{X: 0, Y: 100, Width: 200, Height: 100})
	list.Text("above", 10, 90, 12, Regular, MustHex("#000000")) // its line ends at 94
	list.Text("inside", 10, 150, 12, Regular, MustHex("#000000"))
	list.Text("half in", 10, 104, 12, Regular, MustHex("#000000")) // its top is clipped, the rest shows
	list.Text("below", 10, 220, 12, Regular, MustHex("#000000"))
	list.Text("right", 210, 150, 12, Regular, MustHex("#000000"))
	list.Text("from the left", -50, 150, 12, Regular, MustHex("#000000")) // may run into the clip
	var kept []string
	for _, command := range list.Commands {
		kept = append(kept, command.Text)
	}
	if strings.Join(kept, ",") != "inside,half in,from the left" {
		test.Fatalf("kept %v", kept)
	}
}

func TestFingerprintChangesWithWhatIsDrawn(test *testing.T) {
	seed := maphash.MakeSeed()
	build := func(text string, fill Color) *List {
		var list List
		list.Rectangle(Rectangle{X: 1, Y: 2, Width: 30, Height: 40}, 4, fill, Color{}, 0)
		list.Text(text, 5, 20, 12, Medium, MustHex("#000000"))
		return &list
	}
	same := build("Faro", MustHex("#ffffff")).Fingerprint(seed)
	if build("Faro", MustHex("#ffffff")).Fingerprint(seed) != same {
		test.Fatal("the same picture, the same fingerprint")
	}
	if build("Faros", MustHex("#ffffff")).Fingerprint(seed) == same || build("Faro", MustHex("#fffffe")).Fingerprint(seed) == same {
		test.Fatal("a different text or colour changes it")
	}
	var empty List
	if empty.Fingerprint(seed) == same {
		test.Fatal("an empty list differs")
	}
}

func TestLightsAreAddedAsCommands(test *testing.T) {
	var list List
	picture := NewBlankImage(4, 4)
	list.PointLight(100, 50, 10, 40, Color{1, 0.9, 0.6, 1.5})
	list.SpriteLight(picture, Rectangle{X: 0, Y: 0, Width: 20, Height: 10}, Color{1, 1, 1, 1})
	list.Lighting(Color{1, 1, 1, 0.72})
	if len(list.Commands) != 3 {
		test.Fatalf("three light commands made %d", len(list.Commands))
	}
	point := list.Commands[0]
	if point.Kind != PointLightKind || point.Area != (Rectangle{X: 60, Y: 10, Width: 80, Height: 80}) || point.Radius != 10 || point.Fill[3] != 1.5 {
		test.Fatalf("the point light is %+v", point)
	}
	if list.Commands[1].Kind != SpriteLightKind || list.Commands[1].Picture != picture {
		test.Fatalf("the sprite light is %+v", list.Commands[1])
	}
	if lighting := list.Commands[2]; lighting.Kind != LightingKind || lighting.Fill[3] != 0.72 {
		test.Fatalf("the lighting is %+v", lighting)
	}
}

func TestLightsThatCannotShineAreLeftOut(test *testing.T) {
	var list List
	list.PointLight(0, 0, 0, 0, Color{1, 1, 1, 1})
	list.PointLight(0, 0, 0, 10, Color{1, 1, 1, 0})
	list.SpriteLight(nil, Rectangle{Width: 10, Height: 10}, Color{1, 1, 1, 1})
	if len(list.Commands) != 0 {
		test.Fatalf("lights with no size, no intensity or no picture made %d commands", len(list.Commands))
	}
}

func TestALightingUnderAClipLightsOnlyTheClip(test *testing.T) {
	var list List
	clip := Rectangle{X: 100, Y: 100, Width: 10, Height: 10}
	list.PushClip(clip)
	list.Lighting(Color{1, 1, 1, 1})
	list.PopClip()
	if len(list.Commands) != 1 || !list.Commands[0].HasClip || list.Commands[0].Clip != clip {
		test.Fatalf("a lighting under a clip became %+v", list.Commands)
	}
}
