//go:build android

package android

/*
#include <stdlib.h>
#include "activity.h"
*/
import "C"

import (
	"image"
	"unsafe"

	"github.com/akrck02/go-renderer/opengl/overlay"
)

// useSystemEmoji lets the overlay ask Android for the emoji no font file it reads has, when the
// package carries EmojiPicture.java.
func useSystemEmoji() {
	if C.emoji_available() == 0 {
		return
	}
	overlay.UseSystemEmoji(drawEmoji)
}

// drawEmoji asks Android to draw a character as a colour emoji at a size in pixels.
func drawEmoji(character rune, pixels int) (image.Image, float32, float32, float32, bool) {
	var length C.int
	answer := C.emoji_draw(C.int(character), C.int(pixels), &length)
	if answer == nil {
		return nil, 0, 0, 0, false
	}
	defer C.free(unsafe.Pointer(answer))
	picture, left, top, advance, found := emojiFromJava(unsafe.Slice((*int32)(unsafe.Pointer(answer)), int(length)))
	if !found {
		return nil, 0, 0, 0, false
	}
	return picture, left, top, advance, true
}
