package draw2d

import "testing"

// alphaAt is a stroke picture's coverage at a pixel.
func alphaAt(picture *Image, x, y int) uint8 {
	return picture.Pixels.NRGBAAt(x, y).A
}

func TestAStrokeCoversItsLineAndNotAroundIt(test *testing.T) {
	var path Path
	path.MoveTo(0, 0)
	path.LineTo(10, 0)
	var list List
	list.Stroke(&path, 100, 50, 1, 2, Color{0, 0, 0, 1})
	if len(list.Commands) != 1 || list.Commands[0].Kind != ImageKind {
		test.Fatalf("a stroke draws %v", list.Commands)
	}
	command := list.Commands[0]
	if command.Area != (Rectangle{X: 98, Y: 48, Width: 14, Height: 4}) {
		test.Fatalf("the stroke covers %v", command.Area)
	}
	picture := command.Picture
	middleX, lineY := 2*7, 2*2 // the line's middle, in the picture's pixels (two per layout pixel)
	if alphaAt(picture, middleX, lineY) != 255 {
		test.Fatal("the stroke does not cover its line")
	}
	if alphaAt(picture, middleX, 0) != 0 {
		test.Fatal("the stroke covers more than its width")
	}
}

func TestAStrokeIsDrawnAgainOnlyWhenItsSizeChanges(test *testing.T) {
	var path Path
	path.Circle(10, 10, 10)
	var list List
	list.Stroke(&path, 0, 0, 1, 1.5, Color{0, 0, 0, 1})
	first := list.Commands[0].Picture
	list.Reset()
	list.Stroke(&path, 40, 40, 1, 1.5, Color{1, 0, 0, 1}) // moved and tinted: the same picture
	if list.Commands[0].Picture != first {
		test.Fatal("a stroke moved draws its picture again")
	}
	list.Reset()
	list.Stroke(&path, 0, 0, 2, 1.5, Color{0, 0, 0, 1})
	if list.Commands[0].Picture == first {
		test.Fatal("a stroke scaled keeps the picture of its old size")
	}
}

func TestACurveEndsWhereItIsAimed(test *testing.T) {
	var path Path
	path.MoveTo(0, 0)
	path.CurveTo(0, 10, 10, 10, 10, 0)
	if end := path.last(); end != (Point{10, 0}) {
		test.Fatalf("the curve ends at %v", end)
	}
	if bounds := path.Bounds(); bounds.Height <= 5 || bounds.Height >= 10 {
		test.Fatalf("the curve bulges %v down, not between its ends and its handles", bounds.Height)
	}
}

func TestARoundedRectangleKeepsToItsSides(test *testing.T) {
	var path Path
	path.RoundedRectangle(10, 20, 100, 40, 8)
	if bounds := path.Bounds(); bounds != (Rectangle{X: 10, Y: 20, Width: 100, Height: 40}) {
		test.Fatalf("the rectangle's line spans %v", bounds)
	}
}

func TestDrawingAStrokeKeptAllocatesNothing(test *testing.T) {
	var path Path
	path.Circle(10, 10, 10)
	var list List
	list.Stroke(&path, 0, 0, 1, 1.5, Color{0, 0, 0, 1})
	allocations := testing.AllocsPerRun(100, func() {
		list.Reset()
		list.Stroke(&path, 0, 0, 1, 1.5, Color{0, 0, 0, 1})
	})
	if allocations != 0 {
		test.Fatalf("drawing a stroke kept allocates %v times", allocations)
	}
}
