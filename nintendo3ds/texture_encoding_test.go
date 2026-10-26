package nintendo3ds

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/scene"
)

// decodeETC1Block decodes a block as the ETC1 specification says (OES_compressed_ETC1_RGB8_texture),
// pixel (x, y) at [y*4+x]: what the PICA200 shows for it.
func decodeETC1Block(block uint64) [16][3]int {
	bits := func(from, count uint) int { return int(block>>from) & (1<<count - 1) }
	var bases [2][3]int
	if bits(33, 1) == 1 {
		for channel := range 3 {
			first := bits(uint(59-8*channel), 5)
			delta := bits(uint(56-8*channel), 3)
			if delta >= 4 {
				delta -= 8
			}
			bases[0][channel], bases[1][channel] = first<<3|first>>2, (first+delta)<<3|(first+delta)>>2
		}
	} else {
		for channel := range 3 {
			bases[0][channel], bases[1][channel] = bits(uint(60-8*channel), 4)*17, bits(uint(56-8*channel), 4)*17
		}
	}
	tables := [2]int{bits(37, 3), bits(34, 3)}
	var pixels [16][3]int
	for y := range 4 {
		for x := range 4 {
			half := x / 2
			if bits(32, 1) == 1 {
				half = y / 2
			}
			position := uint(x*4 + y)
			index := bits(16+position, 1)<<1 | bits(position, 1)
			modifier := etc1Modifiers[tables[half]][index%2]
			if index >= 2 {
				modifier = -modifier
			}
			for channel := range 3 {
				pixels[y*4+x][channel] = min(max(bases[half][channel]+modifier, 0), 255)
			}
		}
	}
	return pixels
}

// TestTheDecoderReadsTwoBlocksWorkedOutByHand checks the test's own decoder on two blocks whose
// pixels were worked out by hand from the specification, so the encoder's tests can trust it.
func TestTheDecoderReadsTwoBlocksWorkedOutByHand(test *testing.T) {
	// individual colours 8,8,8 (136) and 0,0,0; tables 0 (±2, ±8) and 7 (±47, ±183); left and
	// right halves; every pixel the small positive step
	individual := uint64(8)<<60 | uint64(8)<<52 | uint64(8)<<44 | uint64(7)<<34
	pixels := decodeETC1Block(individual)
	if pixels[0] != [3]int{138, 138, 138} || pixels[3] != [3]int{47, 47, 47} {
		test.Errorf("individual block: %v and %v, wanted 138 grey on the left, 47 on the right", pixels[0], pixels[3])
	}
	// differential 16,16,16 (132) and deltas +3, 0, -4 (156, 132, 99); tables 1 (±5) and 2 (±9,
	// ±29); top and bottom halves; the last pixel the large negative step
	differential := uint64(16)<<59 | uint64(3)<<56 | uint64(16)<<51 | uint64(16)<<43 | uint64(4)<<40 |
		uint64(1)<<37 | uint64(2)<<34 | 1<<33 | 1<<32 | 1<<31 | 1<<15
	pixels = decodeETC1Block(differential)
	if pixels[0] != [3]int{137, 137, 137} || pixels[8] != [3]int{165, 141, 108} || pixels[15] != [3]int{127, 103, 70} {
		test.Errorf("differential block: %v, %v, %v; wanted 137 grey, (165, 141, 108), (127, 103, 70)", pixels[0], pixels[8], pixels[15])
	}
}

func TestEncodedBlocksDecodeCloseToTheirPixels(test *testing.T) {
	cases := map[string]struct {
		left, right [3]uint8
		tolerance   int
	}{
		"mid grey":        {[3]uint8{128, 128, 128}, [3]uint8{128, 128, 128}, 4},
		"a dark green":    {[3]uint8{30, 90, 40}, [3]uint8{30, 90, 40}, 4},
		"white":           {[3]uint8{255, 255, 255}, [3]uint8{255, 255, 255}, 2},
		"red beside blue": {[3]uint8{220, 20, 20}, [3]uint8{20, 20, 220}, 12},
	}
	for name, sample := range cases {
		var block [16][3]uint8
		for index := range block {
			block[index] = sample.left
			if index%4 >= 2 {
				block[index] = sample.right
			}
		}
		decoded := decodeETC1Block(encodeETC1Block(&block))
		for index := range block {
			for channel := range 3 {
				if difference := decoded[index][channel] - int(block[index][channel]); difference > sample.tolerance || difference < -sample.tolerance {
					test.Errorf("%s: pixel %d decodes as %v from %v", name, index, decoded[index], block[index])
					break
				}
			}
		}
	}
}

// pictureOf is a picture of a size painted by a function of each pixel.
func pictureOf(width, height int, paint func(x, y int) color.NRGBA) *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			picture.SetNRGBA(x, y, paint(x, y))
		}
	}
	return picture
}

var (
	red  = color.NRGBA{255, 0, 0, 255}
	blue = color.NRGBA{0, 0, 255, 255}
)

func TestETC1BlocksAreTiledFromTheBottomAndStoredLittleEndian(test *testing.T) {
	picture := pictureOf(8, 8, func(x, y int) color.NRGBA {
		if x < 4 && y >= 4 {
			return red // the bottom-left block: the first one the PICA200 reads
		}
		return blue
	})
	encoded := encodeTexture(nil, picture, scene.ETC1)
	if len(encoded) != 4*8 {
		test.Fatalf("an 8 × 8 picture is %d bytes in ETC1, wanted four blocks of 8", len(encoded))
	}
	first, second := decodeETC1Block(binary.LittleEndian.Uint64(encoded)), decodeETC1Block(binary.LittleEndian.Uint64(encoded[8:]))
	if first[0][0] < 200 || first[0][2] > 50 || second[0][2] < 200 {
		test.Errorf("the first block is %v, wanted red; the second %v, wanted blue", first[0], second[0])
	}
}

func TestRGB565AndRGBA8PixelsAreTiledAsThePICA200ReadsThem(test *testing.T) {
	picture := pictureOf(8, 8, func(x, y int) color.NRGBA {
		if x == 0 && y == 7 {
			return red // bottom-left: the first pixel
		}
		return color.NRGBA{0, 255, 0, 128}
	})
	if encoded := encodeTexture(nil, picture, scene.RGB565); len(encoded) != 128 || encoded[0] != 0x00 || encoded[1] != 0xf8 || encoded[2] != 0xe0 || encoded[3] != 0x07 {
		test.Errorf("RGB565 starts % x, wanted red (00 f8) then green (e0 07)", encoded[:4])
	}
	if encoded := encodeTexture(nil, picture, scene.RGBA8); len(encoded) != 256 || encoded[0] != 255 || encoded[3] != 255 || encoded[4] != 128 || encoded[6] != 255 {
		test.Errorf("RGBA8 starts % x, wanted alpha, blue, green, red", encoded[:8])
	}
}

func TestATexturesSidesArePowersOfTwoFromEightTo1024(test *testing.T) {
	cases := map[int]int{1: 8, 8: 8, 30: 32, 100: 128, 64: 64, 90: 64, 1024: 1024, 3000: 1024}
	for side, wanted := range cases {
		if got := textureSide(side); got != wanted {
			test.Errorf("a side of %d becomes %d, wanted %d", side, got, wanted)
		}
	}
	scaled := fittedPixels(pictureOf(100, 30, func(x, y int) color.NRGBA { return red }))
	if bounds := scaled.Bounds(); bounds.Dx() != 128 || bounds.Dy() != 32 || scaled.NRGBAAt(127, 31) != red {
		test.Errorf("a picture of 100 × 30 became %v, its corner %v", bounds, scaled.NRGBAAt(127, 31))
	}
}

func TestATextureWithTransparencyCannotLoseIt(test *testing.T) {
	seeThrough := draw2d.NewImage(pictureOf(8, 8, func(x, y int) color.NRGBA { return color.NRGBA{255, 255, 255, 100} }))
	opaque := draw2d.NewImage(pictureOf(8, 8, func(x, y int) color.NRGBA { return red }))
	cases := []struct {
		texture scene.Texture
		wanted  error
	}{
		{scene.Texture{Picture: seeThrough, Format: scene.ETC1}, TextureLosesTransparency},
		{scene.Texture{Picture: seeThrough, Format: scene.RGB565}, TextureLosesTransparency},
		{scene.Texture{Picture: seeThrough}, nil},
		{scene.Texture{Picture: opaque, Format: scene.ETC1}, nil},
	}
	for _, sample := range cases {
		if err := CheckTexture(&sample.texture); !errors.Is(err, sample.wanted) {
			test.Errorf("format %v: %v, wanted %v", sample.texture.Format, err, sample.wanted)
		}
	}
	if format := formatToUse(&scene.Texture{Picture: seeThrough, Format: scene.ETC1}); format != scene.RGBA8 {
		test.Errorf("a transparent picture asked in ETC1 is kept in %v, wanted RGBA8", format)
	}
}
