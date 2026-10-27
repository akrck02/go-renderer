//go:build js && wasm

package overlay

import (
	"bytes"
	"image"
	"math"
	"strconv"
	"syscall/js"

	"github.com/akrck02/go-renderer/draw2d"
)

// A page cannot read the system's font files, but the browser draws with them: each emoji is
// drawn by the browser on a small canvas of its own, read back, and kept as a picture.

// emojiFamilies are the systems' colour emoji fonts, the one the browser has first.
const emojiFamilies = `"Apple Color Emoji", "Segoe UI Emoji", "Noto Color Emoji", sans-serif`

// The two colours an emoji is drawn in, to tell it from a letter: a colour emoji keeps its own
// colours whatever the fill, and a letter of a plain font takes the fill's.
const (
	firstFill  = "#ff0000"
	secondFill = "#0000ff"
)

// emojiPresentationText asks the browser for the emoji picture of a character, not its outline.
const emojiPresentationText = "️"

// emojiSourceFor is where the emoji drawn come from: the browser's own, through a 2D canvas, or,
// where there is none (Node), the colour emoji fonts in these folders, as on other systems.
func emojiSourceFor(folders []string) emojiSource {
	if canvas, found := newBrowserEmoji(); found {
		return canvas
	}
	return newFontFileEmoji(folders)
}

// browserEmoji draws emoji with the browser's fonts on a canvas that is not shown.
type browserEmoji struct {
	canvas  js.Value
	context js.Value // the canvas's 2D context
}

// newBrowserEmoji makes the canvas: an OffscreenCanvas, or a page's canvas element; false where
// there is neither.
func newBrowserEmoji() (*browserEmoji, bool) {
	global := js.Global()
	var canvas js.Value
	switch {
	case global.Get("OffscreenCanvas").Truthy():
		canvas = global.Get("OffscreenCanvas").New(1, 1)
	case global.Get("document").Truthy():
		canvas = global.Get("document").Call("createElement", "canvas")
	default:
		return nil, false
	}
	context := canvas.Call("getContext", "2d", map[string]any{"willReadFrequently": true})
	if !context.Truthy() {
		return nil, false
	}
	return &browserEmoji{canvas: canvas, context: context}, true
}

// emoji draws a character as an emoji at a size in pixels. A character the browser draws in the
// fill's colour is a letter, not a colour emoji: it is left to the text's fonts.
func (browser *browserEmoji) emoji(character rune, size int) (*emojiGlyph, bool) {
	text := string(character) + emojiPresentationText
	browser.setFont(size)
	measure := browser.context.Call("measureText", text)
	advance := float32(measure.Get("width").Float())
	left := measure.Get("actualBoundingBoxLeft").Float()
	top := measure.Get("actualBoundingBoxAscent").Float()
	width := int(math.Ceil(left + measure.Get("actualBoundingBoxRight").Float()))
	height := int(math.Ceil(top + measure.Get("actualBoundingBoxDescent").Float()))
	if advance <= 0 || width <= 0 || height <= 0 {
		return nil, false
	}
	first := browser.draw(text, size, width, height, left, top, firstFill)
	second := browser.draw(text, size, width, height, left, top, secondFill)
	if empty(first) || !samePixels(first, second) {
		return nil, false
	}
	return &emojiGlyph{picture: draw2d.NewImage(first), left: float32(-left), top: float32(top),
		advance: advance, pixelWidth: width, pixelHeight: height}, true
}

// setFont sets the emoji fonts at a size in pixels; resizing the canvas forgets it.
func (browser *browserEmoji) setFont(size int) {
	browser.context.Set("font", strconv.Itoa(size)+"px "+emojiFamilies)
}

// draw draws a text in a colour on a canvas of a size, its pen at left and its baseline at top, and
// reads the pixels back.
func (browser *browserEmoji) draw(text string, size, width, height int, left, top float64, fill string) *image.NRGBA {
	browser.canvas.Set("width", width)
	browser.canvas.Set("height", height)
	browser.setFont(size)
	browser.context.Set("textBaseline", "alphabetic")
	browser.context.Set("fillStyle", fill)
	browser.context.Call("fillText", text, left, top)
	data := browser.context.Call("getImageData", 0, 0, width, height).Get("data")
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	js.CopyBytesToGo(picture.Pix, js.Global().Get("Uint8Array").New(data.Get("buffer")))
	return picture
}

// empty tells whether nothing was drawn: the browser has no picture of the character.
func empty(picture *image.NRGBA) bool {
	for index := 3; index < len(picture.Pix); index += 4 {
		if picture.Pix[index] != 0 {
			return false
		}
	}
	return true
}

// samePixels tells whether two pictures of the same size are the same.
func samePixels(first, second *image.NRGBA) bool {
	return bytes.Equal(first.Pix, second.Pix)
}
