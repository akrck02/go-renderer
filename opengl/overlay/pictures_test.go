package overlay

import (
	"image"
	"image/color"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/scene"
)

// stripes is a picture of width × height, red on its left half and blue on its right.
func stripes(width, height int) *draw2d.Image {
	picture := draw2d.NewBlankImage(width, height)
	for y := range height {
		for x := range width {
			shade := color.NRGBA{R: 255, A: 255}
			if x >= width/2 {
				shade = color.NRGBA{B: 255, A: 255}
			}
			picture.Pixels.SetNRGBA(x, y, shade)
		}
	}
	return picture
}

func TestAnImageCoversItsAreaWithTheMiddleOfAWiderPicture(test *testing.T) {
	var list draw2d.List
	picture := stripes(400, 100) // four times as wide as tall
	list.Image(picture, draw2d.Rectangle{X: 10, Y: 20, Width: 100, Height: 100}, 0, white)
	quads := paintList(test, &list, 1)
	if len(quads) == 0 {
		test.Fatal("an image drew nothing")
	}
	if bounds := covered(quads); bounds != [4]float32{10, 20, 110, 120} {
		test.Fatalf("the image covers %v; want its area", bounds)
	}
	left, right := float32(1e9), float32(-1e9)
	for _, quad := range quads {
		if !quad.Pictured {
			test.Fatal("a quad of an image reads no picture")
		}
		left, right = min(left, quad.Picture[0]), max(right, quad.Picture[2])
	}
	// a square area shows the middle 100 of the picture's 400 columns
	if width := right - left; width < 99 || width > 101 {
		test.Fatalf("the image reads %v columns of its picture; want the middle 100", width)
	}
}

func TestAClippedImageReadsOnlyWhatShows(test *testing.T) {
	var list draw2d.List
	list.PushClip(draw2d.Rectangle{X: 0, Y: 0, Width: 60, Height: 600})
	list.Image(stripes(100, 100), draw2d.Rectangle{X: 10, Y: 10, Width: 100, Height: 100}, 0, white)
	quads := paintList(test, &list, 1)
	right := float32(-1e9)
	for _, quad := range quads {
		if quad.Screen[2] > 60 {
			test.Fatalf("a quad reaches %v past the clip at 60", quad.Screen[2])
		}
		right = max(right, quad.Picture[2])
	}
	// the picture starts one pixel in (its border); half of its 100 columns show
	if right > 1+50+1 {
		test.Fatalf("the clipped image reads its picture up to %v; want only the first half", right)
	}
}

func TestAChangedPictureIsCopiedAgainInPlace(test *testing.T) {
	painter := newTestPainter(test)
	picture := stripes(8, 8)
	var list draw2d.List
	list.Image(picture, draw2d.Rectangle{Width: 80, Height: 80}, 0, white)
	paint := func() { painter.Paint(&scene.Overlay{Drawing: &list}, nil, Frame{Width: 800, Height: 600, Scale: 1}) }
	paint()
	pictures := painter.Pictures()
	if whole, _ := pictures.Pending(); !whole {
		test.Fatal("a new atlas does not ask to be uploaded whole")
	}
	pictures.Uploaded()
	paint()
	if whole, areas := pictures.Pending(); whole || len(areas) != 0 {
		test.Fatal("the same picture drawn again asks for an upload")
	}
	picture.Pixels.SetNRGBA(0, 0, color.NRGBA{G: 255, A: 255})
	picture.Changed()
	paint()
	whole, areas := pictures.Pending()
	if whole || len(areas) != 1 || areas[0].Dx() != 10 {
		test.Fatalf("whole %v, areas %v after the picture changed; want its own 10×10 area with the border", whole, areas)
	}
	bounds, _ := pictures.bounds(picture)
	if got := pictures.Image.NRGBAAt(bounds.Min.X, bounds.Min.Y); got.G != 255 {
		test.Fatalf("the atlas holds %v where the picture changed; want the new green", got)
	}
}

func TestAFullAtlasKeepsOnlyThisFramesPictures(test *testing.T) {
	pictures := NewPictures()
	large := func() *draw2d.Image { return draw2d.NewImage(image.NewNRGBA(image.Rect(0, 0, 2000, 2000))) }
	first := []*draw2d.Image{large(), large(), large(), large()}
	pictures.prepare(first)
	if pictures.Size() != maximumPicturesSize {
		test.Fatalf("atlas %d wide for four large pictures; want it grown to %d", pictures.Size(), maximumPicturesSize)
	}
	pictures.Uploaded()
	next := []*draw2d.Image{large()}
	pictures.prepare(next)
	if _, found := pictures.bounds(next[0]); !found {
		test.Fatal("a picture that did not fit was not drawn after the atlas emptied")
	}
	if _, found := pictures.bounds(first[0]); found {
		test.Fatal("an earlier frame's picture is still there after the atlas emptied")
	}
	if whole, _ := pictures.Pending(); !whole {
		test.Fatal("an emptied atlas does not ask to be uploaded whole")
	}
}

func TestALargePictureIsScaledDown(test *testing.T) {
	width, height := keptSize(draw2d.NewBlankImage(4000, 1000))
	if width != maximumPictureSide || height != 512 {
		test.Fatalf("a 4000×1000 picture is kept at %d×%d; want %d×512", width, height, maximumPictureSide)
	}
}

func TestASharpPictureMarksItsQuadsSharp(test *testing.T) {
	var list draw2d.List
	pixelArt, photograph := stripes(16, 16), stripes(16, 16)
	pixelArt.Sharp = true
	list.Image(pixelArt, draw2d.Rectangle{X: 0, Y: 0, Width: 64, Height: 64}, 0, white)
	list.Image(photograph, draw2d.Rectangle{X: 100, Y: 0, Width: 64, Height: 64}, 0, white)
	sharp, blended := 0, 0
	for _, quad := range paintList(test, &list, 1) {
		switch {
		case !quad.Pictured:
		case quad.Sharp:
			sharp++
		default:
			blended++
		}
	}
	if sharp == 0 || blended == 0 {
		test.Fatalf("%d sharp and %d blended pictured quads", sharp, blended)
	}
}
