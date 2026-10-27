package android

import (
	"image"
	"math"
)

// The header EmojiPicture.java puts before an emoji's pixels: the picture's width and height, then
// where it sits, as float bits: left of the pen, top above the baseline, and the pen's move.
const (
	emojiWidth = iota
	emojiHeight
	emojiLeft
	emojiTop
	emojiAdvance
	emojiHeader
)

// emojiFromJava reads EmojiPicture.draw's answer: the picture, not premultiplied, as Java's
// getPixels gives it, and where it sits; false when the answer is not a whole picture.
func emojiFromJava(answer []int32) (picture *image.NRGBA, left, top, advance float32, found bool) {
	if len(answer) < emojiHeader {
		return nil, 0, 0, 0, false
	}
	width, height := int(answer[emojiWidth]), int(answer[emojiHeight])
	pixels := answer[emojiHeader:]
	if width <= 0 || height <= 0 || len(pixels) != width*height {
		return nil, 0, 0, 0, false
	}
	picture = image.NewNRGBA(image.Rect(0, 0, width, height))
	for index, colour := range pixels {
		argb := uint32(colour)
		picture.Pix[index*4] = uint8(argb >> 16)
		picture.Pix[index*4+1] = uint8(argb >> 8)
		picture.Pix[index*4+2] = uint8(argb)
		picture.Pix[index*4+3] = uint8(argb >> 24)
	}
	return picture, floatFrom(answer[emojiLeft]), floatFrom(answer[emojiTop]), floatFrom(answer[emojiAdvance]), true
}

// floatFrom is a float Java sent as its bits (Float.floatToIntBits).
func floatFrom(bits int32) float32 {
	return math.Float32frombits(uint32(bits))
}
