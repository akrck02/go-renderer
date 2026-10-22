//go:build nintendo3ds

// Eight 2D lights on the Nintendo 3DS: five coloured panels on the top screen in a dim blue
// light, lit by a warm lamp, a cold one, a glow shining through a checked picture and a row of
// five small lamps, with a label drawn after the lighting, unlit. The bottom screen shows the
// same panels unlit, to compare, but for its left half, darkened inside a clip.
//
//	tinygo build -target=nintendo3ds -o lights3ds.3dsx ./cmd/lights3ds
package main

import (
	"image"
	"image/color"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
)

// checkSide is the checked picture's side, in pixels, and checkCell a check's.
const (
	checkSide = 16
	checkCell = 4
)

func main() {
	screen, err := ctr.Open()
	if err != nil {
		println(err.Error())
		return
	}
	bottom, err := ctr.OpenBottomScreen()
	if err != nil {
		println(err.Error())
		return
	}
	renderer, err := ctr.NewFlatRenderer(bottom)
	if err != nil {
		println(err.Error())
		return
	}
	lit, unlit := &draw2d.List{}, &draw2d.List{}
	addPanels(lit)
	addPanels(unlit)
	unlit.PushClip(draw2d.Rectangle{Width: ctr.BottomWidth / 2, Height: ctr.BottomHeight})
	unlit.Lighting(draw2d.Color{1, 1, 1, 0.5})
	unlit.PopClip()
	addLights(lit)
	lit.Rectangle(draw2d.Rectangle{X: 8, Y: 200, Width: 120, Height: 32}, 6, draw2d.MustHex("#ffffff"), draw2d.Transparent, 0)
	lit.Text("Unlit, after", 16, 221, 12, draw2d.Medium, draw2d.MustHex("#202020"))
	grey := draw2d.MustHex("#808080")
	for ctr.Running() {
		ctr.BeginFrame()
		renderer.PaintOn(screen, lit, grey)
		renderer.Paint(unlit, grey)
		ctr.EndFrame()
	}
}

// addPanels adds five panels of different colours in a row.
func addPanels(list *draw2d.List) {
	colours := []string{"#e85d5d", "#5de87a", "#5d8de8", "#e8d35d", "#ffffff"}
	for index, colour := range colours {
		area := draw2d.Rectangle{X: 20 + float32(index)*72, Y: 100, Width: 60, Height: 60}
		list.Rectangle(area, 8, draw2d.MustHex(colour), draw2d.Transparent, 0)
	}
}

// addLights adds the eight lights and lights the panels with them, in a dim blue.
func addLights(list *draw2d.List) {
	list.PointLight(90, 70, 15, 90, draw2d.MustHex("#ffd27a"))
	list.PointLight(320, 180, 10, 70, draw2d.Color{0.5, 0.7, 1, 1.5})
	list.SpriteLight(checks(), draw2d.Rectangle{X: 300, Y: 20, Width: 64, Height: 64}, draw2d.Color{1, 0.4, 0.8, 1})
	for index := range 5 {
		list.PointLight(40+float32(index)*80, 200, 10, 60, draw2d.Color{1, 1, 1, 0.6})
	}
	list.Lighting(draw2d.Color{0.35, 0.35, 0.5, 1})
}

// checks is a picture of white checks on clear ones, for a light to shine through.
func checks() *draw2d.Image {
	pixels := image.NewNRGBA(image.Rect(0, 0, checkSide, checkSide))
	for x := range checkSide {
		for y := range checkSide {
			if (x/checkCell+y/checkCell)%2 == 0 {
				pixels.Set(x, y, color.NRGBA{255, 255, 255, 255})
			}
		}
	}
	return draw2d.NewImage(pixels)
}
