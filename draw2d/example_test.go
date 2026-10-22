package draw2d_test

import (
	"fmt"
	"image/color"

	"github.com/akrck02/go-renderer/draw2d"
)

// ExampleList fills a list with a card: its shadow, the card itself, an icon and a line of text.
func ExampleList() {
	var drawing draw2d.List
	card := draw2d.Rect{X: 24, Y: 24, Width: 240, Height: 64}
	drawing.Shadow(card, 12, 12, 4, draw2d.Hex("#14202f33"))
	drawing.Rectangle(card, 12, draw2d.Hex("#ffffff"), draw2d.Hex("#dde3ea"), 1)
	drawing.Icon("lightbulb", 44, 48, 16, draw2d.Hex("#e0a020"))
	drawing.Text("The lighthouse is lit", 70, 61, 14, draw2d.Medium, draw2d.Hex("#14202f"))
	fmt.Println(len(drawing.Commands), "commands")
	// Output: 4 commands
}

// ExampleList_PushClip cuts a long text to a box, and puts a tooltip on a layer above it.
func ExampleList_PushClip() {
	var drawing draw2d.List
	box := draw2d.Rect{X: 24, Y: 120, Width: 120, Height: 32}
	drawing.Rectangle(box, 6, draw2d.Hex("#f4f6f8"), draw2d.Transparent, 0)
	drawing.PushClip(box.Inset(8))
	drawing.Text("A line too long for its box", 32, 141, 13, draw2d.Regular, draw2d.Hex("#14202f"))
	drawing.PopClip()

	drawing.SetLayer(1) // drawn over layer 0, whatever comes after
	tip := draw2d.Rect{X: 100, Y: 100, Width: 90, Height: 26}
	drawing.Rectangle(tip, 4, draw2d.Hex("#14202f"), draw2d.Transparent, 0)
	drawing.Text("Lighthouse", 108, 117, 12, draw2d.Regular, draw2d.Hex("#ffffff"))
	fmt.Println(drawing.Commands[1].HasClip, drawing.Commands[3].Layer)
	// Output: true 1
}

// ExampleList_Image shows a picture the program made, kept from frame to frame.
func ExampleList_Image() {
	const side = 8
	tiles := draw2d.NewBlankImage(side, side)
	tiles.Sharp = true // pixel art: no blending between pixels
	for y := range side {
		for x := range side {
			if (x+y)%2 == 0 {
				tiles.Pixels.Set(x, y, color.NRGBA{R: 40, G: 120, B: 200, A: 255})
			}
		}
	}

	var drawing draw2d.List
	white := draw2d.Color{1, 1, 1, 1} // a white tint shows the picture as it is
	drawing.Image(tiles, draw2d.Rect{X: 24, Y: 24, Width: 64, Height: 64}, 8, white)
	fmt.Println(tiles.Size())
	// Output: 8 8
}

// ExamplePath draws the outline of a lighthouse and a sun as strokes.
func ExamplePath() {
	var tower draw2d.Path
	tower.MoveTo(0, 10) // the path's own units: here, one per metre
	tower.LineTo(1, 0)
	tower.LineTo(3, 0)
	tower.LineTo(4, 10)
	tower.Close()

	var sun draw2d.Path
	sun.Circle(0, 0, 1)

	var drawing draw2d.List
	drawing.Stroke(&tower, 40, 30, 12, 3, draw2d.Hex("#14202f")) // 12 layout pixels a metre
	drawing.Stroke(&sun, 120, 40, 10, 2, draw2d.Hex("#e0a020"))
	fmt.Println(tower.Bounds())
	// Output: {0 0 4 10}
}
