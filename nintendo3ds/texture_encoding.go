package nintendo3ds

import (
	"encoding/binary"
	"image"
	"math/bits"

	"github.com/akrck02/go-renderer/scene"
)

// The PICA200's textures: sides that are powers of two from 8 to 1024 pixels, in tiles of 8 × 8
// read from the bottom of the picture (see tileForPICA), in one of the formats scene.TextureFormat
// names. A picture of other sides is scaled once, when it is uploaded, to the nearest that fit.

// smallestTexture is the PICA200's smallest texture side, one tile.
const smallestTexture = 8

// textureSide is the side a texture of this many pixels is kept at: the nearest power of two, from
// smallestTexture to largestTexture.
func textureSide(side int) int {
	if side <= smallestTexture {
		return smallestTexture
	}
	if side >= largestTexture {
		return largestTexture
	}
	below := 1 << (bits.Len(uint(side)) - 1)
	if side-below < 2*below-side {
		return below
	}
	return 2 * below
}

// fittedPixels is a picture at the sides the PICA200 takes, scaled to the nearest pixel when its
// own sides do not fit; the picture itself when they do.
func fittedPixels(picture *image.NRGBA) *image.NRGBA {
	bounds := picture.Bounds()
	width, height := textureSide(bounds.Dx()), textureSide(bounds.Dy())
	if width == bounds.Dx() && height == bounds.Dy() {
		return picture
	}
	scaled := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			scaled.SetNRGBA(x, y, picture.NRGBAAt(bounds.Min.X+x*bounds.Dx()/width, bounds.Min.Y+y*bounds.Dy()/height))
		}
	}
	return scaled
}

// CheckTexture tells whether the 3DS can keep a texture as it asks: a picture with see-through
// pixels in a format without opacity is TextureLosesTransparency (drawing keeps it in RGBA8).
// Sides that are not powers of two from 8 to 1024 are no error: the picture is scaled to fit.
func CheckTexture(texture *scene.Texture) error {
	if texture.Format != scene.RGBA8 && hasTransparency(texture.Picture.Pixels) {
		return TextureLosesTransparency
	}
	return nil
}

// formatToUse is the format a texture is kept in: the one it asks for, or RGBA8 when that would
// lose its transparency.
func formatToUse(texture *scene.Texture) scene.TextureFormat {
	if CheckTexture(texture) != nil {
		return scene.RGBA8
	}
	return texture.Format
}

func hasTransparency(pixels *image.NRGBA) bool {
	bounds := pixels.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		row := pixels.Pix[pixels.PixOffset(bounds.Min.X, y):pixels.PixOffset(bounds.Max.X, y)]
		for alpha := 3; alpha < len(row); alpha += 4 {
			if row[alpha] != 255 {
				return true
			}
		}
	}
	return false
}

// bytesPerPixel of each format, as eighths of a byte: ETC1 keeps half a byte a pixel.
var eighthsOfABytePerPixel = [...]int{scene.RGBA8: 32, scene.RGB565: 16, scene.ETC1: 4}

// encodeTexture lays a picture (whose sides fit, see fittedPixels) out as the PICA200 reads it in
// a format, into into's memory when it is large enough.
func encodeTexture(into []byte, picture *image.NRGBA, format scene.TextureFormat) []byte {
	width, height := picture.Rect.Dx(), picture.Rect.Dy()
	into = grownTo(into, width*height*eighthsOfABytePerPixel[format]/8)
	switch format {
	case scene.ETC1:
		encodeETC1(into, picture)
	case scene.RGB565:
		for y := range height {
			for x := range width {
				pixel := picture.NRGBAAt(picture.Rect.Min.X+x, picture.Rect.Min.Y+y)
				packed := uint16(pixel.R>>3)<<11 | uint16(pixel.G>>2)<<5 | uint16(pixel.B>>3)
				binary.LittleEndian.PutUint16(into[2*tiledIndex(x, y, width, height):], packed)
			}
		}
	default:
		for y := range height {
			for x := range width {
				pixel := picture.NRGBAAt(picture.Rect.Min.X+x, picture.Rect.Min.Y+y)
				to := 4 * tiledIndex(x, y, width, height)
				into[to], into[to+1], into[to+2], into[to+3] = pixel.A, pixel.B, pixel.G, pixel.R
			}
		}
	}
	return into
}

// tiledIndex is where the PICA200 reads pixel (x, y), counted from the picture's top-left, among a
// texture's: its tile, the tiles row by row from the bottom, then its place in the tile.
func tiledIndex(x, y, width, height int) int {
	fromBottom := height - 1 - y
	tile := (fromBottom/8)*(width/8) + x/8
	return tile*64 + mortonInTile(x, fromBottom)
}
