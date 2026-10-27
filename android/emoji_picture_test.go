package android

import (
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/akrck02/go-renderer/android/java"
)

// Java sends the measures as float bits and the pixels as ARGB, not premultiplied: Go must read
// them back as they were drawn, or a half-clear edge comes out darker.
func TestAnEmojiFromJavaIsReadAsItWasDrawn(test *testing.T) {
	halfClearRed := uint32(0x80ff0000) // as Java's int holds it
	answer := []int32{2, 1, int32(math.Float32bits(-1)), int32(math.Float32bits(14)), int32(math.Float32bits(18)), int32(halfClearRed), 0x0000ff00}
	picture, left, top, advance, found := emojiFromJava(answer)
	if !found || picture.Bounds().Dx() != 2 || picture.Bounds().Dy() != 1 {
		test.Fatal("a picture of two pixels by one")
	}
	if left != -1 || top != 14 || advance != 18 {
		test.Fatalf("left %v, top %v, advance %v", left, top, advance)
	}
	if picture.NRGBAAt(0, 0) != (color.NRGBA{R: 0xff, A: 0x80}) || picture.NRGBAAt(1, 0) != (color.NRGBA{G: 0xff}) {
		test.Fatalf("the pixels: %v %v", picture.NRGBAAt(0, 0), picture.NRGBAAt(1, 0))
	}
}

func TestAnAnswerThatIsNoWholePictureIsNoEmoji(test *testing.T) {
	for _, answer := range [][]int32{nil, {2, 2, 0, 0, 0, 1, 2, 3}, {0, 0, 0, 0, 0}} {
		if _, _, _, _, found := emojiFromJava(answer); found {
			test.Fatalf("%v read as a picture", answer)
		}
	}
}

// The header is numbered twice, here and in EmojiPicture.java, and the C side calls the method by
// its signature: they must agree.
func TestJavaAndGoAgreeOnTheEmojisAnswer(test *testing.T) {
	if !strings.Contains(java.EmojiPicture, "WIDTH = 0, HEIGHT = 1, LEFT = 2, TOP = 3, ADVANCE = 4, HEADER = 5;") ||
		emojiWidth != 0 || emojiHeight != 1 || emojiLeft != 2 || emojiTop != 3 || emojiAdvance != 4 || emojiHeader != 5 {
		test.Fatal("EmojiPicture.java and emoji_picture.go number the header differently")
	}
	if !strings.Contains(java.EmojiPicture, "public static int[] draw(int character, int pixels)") {
		test.Fatal(`emoji.c calls draw as "(II)[I"`)
	}
}
