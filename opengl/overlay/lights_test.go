package overlay

import (
	"image"
	"image/color"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/scene"
)

// paintWithLights lays out a list at a scale and gives its quads, lights and lightings.
func paintWithLights(test *testing.T, list *draw2d.List, scale float32) ([]Quad, []LightQuad, []Lighting) {
	test.Helper()
	painter, err := NewPainter()
	if err != nil {
		test.Fatal(err)
	}
	quads := painter.Paint(&scene.Overlay{Drawing: list}, nil, Frame{Width: 800, Height: 600, Scale: scale})
	return quads, painter.Lights(), painter.Lightings()
}

func TestAPointLightIsASquareRoundItsCentre(test *testing.T) {
	var list draw2d.List
	list.PointLight(100, 50, 10, 40, draw2d.Color{1, 0.9, 0.6, 1.5})
	list.Lighting(draw2d.Color{1, 1, 1, 0.72})
	_, lights, lightings := paintWithLights(test, &list, 2)
	if len(lights) != 1 || lights[0].Screen != [4]float32{120, 20, 280, 180} || lights[0].Inner != 0.25 || lights[0].Pictured {
		test.Fatalf("the light, at twice the scale, is %+v", lights)
	}
	if lights[0].Color != [4]float32{1, 0.9, 0.6, 1.5} {
		test.Fatalf("the light's colour is %v", lights[0].Color)
	}
	if len(lightings) != 1 || lightings[0] != (Lighting{At: 0, First: 0, Last: 1, Ambient: [4]float32{1, 1, 1, 0.72}, Area: [4]float32{0, 0, 800, 600}}) {
		test.Fatalf("the lighting is %+v", lightings)
	}
}

func TestALightingLightsWhatWasDrawnBeforeItWithTheLightsSinceTheLast(test *testing.T) {
	var list draw2d.List
	list.Rectangle(draw2d.Rect{Width: 800, Height: 600}, 0, white, draw2d.Transparent, 0)
	list.PointLight(100, 100, 0, 50, white)
	list.PointLight(300, 100, 0, 50, white)
	list.Lighting(draw2d.Color{0, 0, 0, 1})
	list.SetLayer(1)
	list.PointLight(500, 100, 0, 50, white)
	list.Lighting(draw2d.Color{0, 0, 0, 1})
	list.Rectangle(draw2d.Rect{X: 10, Y: 10, Width: 50, Height: 20}, 0, white, draw2d.Transparent, 0)
	quads, lights, lightings := paintWithLights(test, &list, 1)
	if len(lights) != 3 || len(lightings) != 2 {
		test.Fatalf("three lights and two lightings became %d and %d", len(lights), len(lightings))
	}
	first, second := lightings[0], lightings[1]
	if first.First != 0 || first.Last != 2 || second.First != 2 || second.Last != 3 {
		test.Fatalf("the lightings take the lights %v and %v", first, second)
	}
	if first.At == 0 || second.At != first.At || second.At >= len(quads) {
		test.Fatalf("the lightings come after the background (%d), and before the rectangle above them: %d of %d", first.At, second.At, len(quads))
	}
}

func TestASpriteLightShinesThroughItsPicture(test *testing.T) {
	pixels := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for x := range 4 {
		pixels.Set(x, 1, color.NRGBA{255, 255, 255, 255})
	}
	picture := draw2d.NewImage(pixels)
	var list draw2d.List
	list.SpriteLight(picture, draw2d.Rect{X: 10, Y: 10, Width: 40, Height: 40}, draw2d.Color{1, 1, 0.8, 1})
	list.Lighting(draw2d.Color{1, 1, 1, 0.5})
	_, lights, _ := paintWithLights(test, &list, 1)
	if len(lights) != 1 || !lights[0].Pictured || lights[0].Picture[2]-lights[0].Picture[0] != 4 {
		test.Fatalf("the sprite light is %+v", lights)
	}
}

func TestNothingLitCostsNothing(test *testing.T) {
	var list draw2d.List
	list.Rectangle(draw2d.Rect{Width: 100, Height: 100}, 0, white, draw2d.Transparent, 0)
	_, lights, lightings := paintWithLights(test, &list, 1)
	if len(lights) != 0 || len(lightings) != 0 {
		test.Fatalf("an unlit list has %d lights and %d lightings", len(lights), len(lightings))
	}
}

func TestALightingLightsItsClipOrTheWholeFrame(test *testing.T) {
	var list draw2d.List
	list.Lighting(draw2d.Color{1, 1, 1, 1})
	list.PushClip(draw2d.Rect{X: 100, Y: 50, Width: 200, Height: 100})
	list.Lighting(draw2d.Color{1, 1, 1, 1})
	list.PopClip()
	_, _, lightings := paintWithLights(test, &list, 2)
	if whole := lightings[0].Area; whole != [4]float32{0, 0, 800, 600} {
		test.Errorf("a lighting with no clip lights %v, not the whole frame", whole)
	}
	if clipped := lightings[1].Area; clipped != [4]float32{200, 100, 600, 300} {
		test.Errorf("a lighting under a clip lights %v, not the clip in screen pixels", clipped)
	}
}
