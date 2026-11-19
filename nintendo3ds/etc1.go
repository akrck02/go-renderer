package nintendo3ds

import (
	"encoding/binary"
	"image"
)

// ETC1 keeps a 4 × 4 block of pixels in 64 bits: the block cut in two halves (left and right, or
// top and bottom), each half one base colour, and each pixel that colour moved up or down by one of
// four steps from a table of eight. The bits are the specification's (Khronos'
// OES_compressed_ETC1_RGB8_texture); the PICA200 reads each block as a little-endian word, and the
// blocks in 8 × 8 tiles of four, bottom-left first as the picture's other formats are tiled.
//
// This encoder tries each way of cutting the block, with the base colours as 4 bits each or as 5
// bits and a small difference, and keeps what is closest: slower than the best tools, which also
// search around the averages, but a texture is encoded once, when it is uploaded.

// etc1Modifiers are the specification's eight tables: the small and the large step, which a pixel
// adds or takes away.
var etc1Modifiers = [8][2]int{{2, 8}, {5, 17}, {9, 29}, {13, 42}, {18, 60}, {24, 80}, {33, 106}, {47, 183}}

// encodeETC1 writes a picture's blocks into into, eight bytes each, as the PICA200 reads them.
func encodeETC1(into []byte, picture *image.NRGBA) {
	width, height := picture.Rect.Dx(), picture.Rect.Dy()
	var block [16][3]uint8
	written := 0
	for tileTop := 0; tileTop < height; tileTop += 8 {
		for tileLeft := 0; tileLeft < width; tileLeft += 8 {
			for _, corner := range [4][2]int{{0, 0}, {4, 0}, {0, 4}, {4, 4}} {
				readBlock(&block, picture, tileLeft+corner[0], tileTop+corner[1])
				binary.LittleEndian.PutUint64(into[written:], encodeETC1Block(&block))
				written += 8
			}
		}
	}
}

// readBlock reads the 4 × 4 block at (left, top) of the picture turned upside down, which is how
// the PICA200 counts rows; pixel (x, y) of the block goes to [y*4+x].
func readBlock(block *[16][3]uint8, picture *image.NRGBA, left, top int) {
	height := picture.Rect.Dy()
	for y := range 4 {
		for x := range 4 {
			pixel := picture.NRGBAAt(picture.Rect.Min.X+left+x, picture.Rect.Min.Y+height-1-(top+y))
			block[y*4+x] = [3]uint8{pixel.R, pixel.G, pixel.B}
		}
	}
}

// etc1Half is how one half of a block is encoded: its base colour as stored and as expanded to 8
// bits, its table, each of its pixels' step, and how far it is from the pixels.
type etc1Half struct {
	stored, expanded [3]int
	table            int
	steps            [16]int // by pixel, for the pixels in the half
	distance         int     // squared, summed over the half's pixels
}

// encodeETC1Block encodes a block of pixels, (x, y) at [y*4+x], as the specification's 64 bits.
func encodeETC1Block(block *[16][3]uint8) uint64 {
	best, bestError := uint64(0), -1
	for _, flipped := range []bool{false, true} {
		averages := halfAverages(block, flipped)
		candidates := []bool{false}
		if fitsDifference(averages) {
			candidates = append(candidates, true)
		}
		for _, differential := range candidates {
			word, distance := encodeWithBases(block, flipped, differential, averages)
			if bestError < 0 || distance < bestError {
				best, bestError = word, distance
			}
		}
	}
	return best
}

// inFirstHalf tells whether pixel (x, y) is in a block's first half: the left one, or the top one
// when the block is flipped.
func inFirstHalf(x, y int, flipped bool) bool {
	if flipped {
		return y < 2
	}
	return x < 2
}

// halfAverages are the mean colours of a block's two halves.
func halfAverages(block *[16][3]uint8, flipped bool) [2][3]float64 {
	var sums [2][3]float64
	for index, pixel := range block {
		half := 1
		if inFirstHalf(index%4, index/4, flipped) {
			half = 0
		}
		for channel := range 3 {
			sums[half][channel] += float64(pixel[channel]) / 8
		}
	}
	return sums
}

// fitsDifference tells whether the halves' colours, in 5 bits, are close enough for the
// differential mode: the second within -4 to 3 of the first in every channel.
func fitsDifference(averages [2][3]float64) bool {
	for channel := range 3 {
		difference := quantized(averages[1][channel], 31) - quantized(averages[0][channel], 31)
		if difference < -4 || difference > 3 {
			return false
		}
	}
	return true
}

// quantized is a channel from 0 to 255 as the nearest of levels+1 steps.
func quantized(channel float64, levels int) int {
	return min(max(int(channel*float64(levels)/255+0.5), 0), levels)
}

// encodeWithBases encodes a block cut one way, its base colours in one mode, and says how far the
// result is from the pixels.
func encodeWithBases(block *[16][3]uint8, flipped, differential bool, averages [2][3]float64) (uint64, int) {
	var halves [2]etc1Half
	for half := range 2 {
		for channel := range 3 {
			if differential {
				stored := quantized(averages[half][channel], 31)
				halves[half].stored[channel], halves[half].expanded[channel] = stored, stored<<3|stored>>2
			} else {
				stored := quantized(averages[half][channel], 15)
				halves[half].stored[channel], halves[half].expanded[channel] = stored, stored*17
			}
		}
		chooseTable(&halves[half], block, flipped, half == 0)
	}
	return packETC1(halves, flipped, differential), halves[0].distance + halves[1].distance
}

// chooseTable finds the table, and each pixel's step, that bring a half's base colour closest to
// its pixels.
func chooseTable(half *etc1Half, block *[16][3]uint8, flipped, first bool) {
	half.distance = -1
	var steps [16]int
	for table, modifiers := range etc1Modifiers {
		total := 0
		for index, pixel := range block {
			if inFirstHalf(index%4, index/4, flipped) != first {
				continue
			}
			step, distance := nearestStep(half.expanded, modifiers, pixel)
			steps[index], total = step, total+distance
		}
		if half.distance < 0 || total < half.distance {
			half.distance, half.table, half.steps = total, table, steps
		}
	}
}

// nearestStep is the step (0: up a little, 1: up a lot, 2: down a little, 3: down a lot) that
// brings a base colour closest to a pixel, and the squared distance left.
func nearestStep(base [3]int, modifiers [2]int, pixel [3]uint8) (int, int) {
	bestStep, bestError := 0, -1
	for step := range 4 {
		modifier := modifiers[step%2]
		if step >= 2 {
			modifier = -modifier
		}
		distance := 0
		for channel := range 3 {
			difference := min(max(base[channel]+modifier, 0), 255) - int(pixel[channel])
			distance += difference * difference
		}
		if bestError < 0 || distance < bestError {
			bestStep, bestError = step, distance
		}
	}
	return bestStep, bestError
}

// packETC1 writes two encoded halves into the specification's 64 bits.
func packETC1(halves [2]etc1Half, flipped, differential bool) uint64 {
	var word uint64
	for channel := range 3 {
		shift := uint(56 - 8*channel)
		if differential {
			delta := uint64(halves[1].stored[channel]-halves[0].stored[channel]) & 7
			word |= uint64(halves[0].stored[channel])<<(shift+3) | delta<<shift
		} else {
			word |= uint64(halves[0].stored[channel])<<(shift+4) | uint64(halves[1].stored[channel])<<shift
		}
	}
	word |= uint64(halves[0].table)<<37 | uint64(halves[1].table)<<34
	if differential {
		word |= 1 << 33
	}
	if flipped {
		word |= 1 << 32
	}
	for index := range 16 {
		half := 1
		if inFirstHalf(index%4, index/4, flipped) {
			half = 0
		}
		step := uint64(halves[half].steps[index])
		position := uint(index%4*4 + index/4) // the specification counts pixels column by column
		word |= (step>>1)<<(16+position) | (step&1)<<position
	}
	return word
}
