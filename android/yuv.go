package android

import "image"

// YUVPlanes is a decoded video picture: a full-size luma plane and two chroma planes of half its
// width and height, each with how far apart its rows are, and its chroma pixels (1 when a plane
// holds only its own, 2 when the two are interleaved). Crop is the part that shows.
type YUVPlanes struct {
	Luma, Blue, Red          []byte
	LumaRow, BlueRow, RedRow int
	BluePixel, RedPixel      int
	Crop                     image.Rectangle
}

// ToRGBA turns the picture into straight RGBA, as many pixels as fit into, from the top-left of
// the crop. Films are in video's limited range; the colours of pictures 720 lines tall and over are
// the high-definition standard's (BT.709), those of smaller ones the older standard's (BT.601).
func (planes YUVPlanes) ToRGBA(into *image.NRGBA) {
	width := min(planes.Crop.Dx(), into.Rect.Dx())
	height := min(planes.Crop.Dy(), into.Rect.Dy())
	weights := standardDefinition
	if planes.Crop.Dy() >= 720 {
		weights = highDefinition
	}
	for row := range height {
		y := planes.Crop.Min.Y + row
		lumaRow := planes.Luma[y*planes.LumaRow:]
		blueRow := planes.Blue[(y/2)*planes.BlueRow:]
		redRow := planes.Red[(y/2)*planes.RedRow:]
		pixels := into.Pix[into.PixOffset(into.Rect.Min.X, into.Rect.Min.Y+row):]
		for column := range width {
			x := planes.Crop.Min.X + column
			chroma := x / 2
			red, green, blue := weights.colour(lumaRow[x], blueRow[chroma*planes.BluePixel], redRow[chroma*planes.RedPixel])
			pixels[column*4], pixels[column*4+1], pixels[column*4+2], pixels[column*4+3] = red, green, blue, 255
		}
	}
}

// colourWeights turn a limited-range luma and chroma into colours, in 1/256ths.
type colourWeights struct{ luma, redFromRed, greenFromBlue, greenFromRed, blueFromBlue int }

var (
	standardDefinition = colourWeights{luma: 298, redFromRed: 409, greenFromBlue: 100, greenFromRed: 208, blueFromBlue: 516}
	highDefinition     = colourWeights{luma: 298, redFromRed: 459, greenFromBlue: 55, greenFromRed: 136, blueFromBlue: 541}
)

func (weights colourWeights) colour(luma, blue, red byte) (byte, byte, byte) {
	lightness := weights.luma * (int(luma) - 16)
	towardsBlue, towardsRed := int(blue)-128, int(red)-128
	return clampByte(lightness + weights.redFromRed*towardsRed),
		clampByte(lightness - weights.greenFromBlue*towardsBlue - weights.greenFromRed*towardsRed),
		clampByte(lightness + weights.blueFromBlue*towardsBlue)
}

func clampByte(value int) byte {
	return byte(min(max((value+128)>>8, 0), 255))
}
