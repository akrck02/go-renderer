package draw2d

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/gif" // the formats DecodeImage reads
	_ "image/jpeg"
	_ "image/png"
	"io"
	"sync/atomic"
)

// Image is a picture a list draws: a photo, a map tile, a video's frame. A program makes one for
// each picture it shows and keeps it between frames, so the backend uploads its pixels once. To
// show other pixels in it (the next frame of a video), the program writes them into Pixels and
// calls Changed.
type Image struct {
	// Pixels are the picture's colours, not premultiplied by their opacity.
	Pixels  *image.NRGBA
	id      uint64
	version uint64
}

var nextImage atomic.Uint64

// NewImage makes an Image of a decoded picture (image.Decode's PNG, JPEG or GIF), turning its
// colours into straight RGBA.
func NewImage(picture image.Image) *Image {
	pixels, isStraight := picture.(*image.NRGBA)
	if !isStraight || pixels.Rect.Min != (image.Point{}) {
		bounds := picture.Bounds()
		pixels = image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(pixels, pixels.Rect, picture, bounds.Min, draw.Src)
	}
	return &Image{Pixels: pixels, id: nextImage.Add(1)}
}

// DecodeImage reads a PNG, JPEG or GIF picture (a file the program embeds, a download) into an
// Image.
func DecodeImage(reader io.Reader) (*Image, error) {
	picture, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("draw2d: decoding a picture: %w", err)
	}
	return NewImage(picture), nil
}

// NewBlankImage makes an Image of a size, transparent, for pixels the program writes itself (a
// video's frames).
func NewBlankImage(width, height int) *Image {
	return &Image{Pixels: image.NewNRGBA(image.Rect(0, 0, width, height)), id: nextImage.Add(1)}
}

// Changed tells the backend that Pixels hold another picture of the same size, to upload again.
func (picture *Image) Changed() { picture.version++ }

// ID tells images apart for the life of the program.
func (picture *Image) ID() uint64 { return picture.id }

// Version counts the times the picture changed (Changed).
func (picture *Image) Version() uint64 { return picture.version }

// Size is the picture's width and height in pixels.
func (picture *Image) Size() (width, height int) {
	return picture.Pixels.Rect.Dx(), picture.Pixels.Rect.Dy()
}
