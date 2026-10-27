//go:build js && wasm

package overlay

import (
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
)

// Where there is no canvas to draw on (Node runs these tests), a page has no emoji: its text is
// still drawn, with a question mark for each, and nothing fails.
func TestWithoutACanvasAnEmojiIsDrawnAsAQuestionMark(test *testing.T) {
	if _, found := newBrowserEmoji(); found {
		test.Skip("this runtime has a canvas: the browser draws the emoji")
	}
	measurer, err := NewMeasurer(Fonts{Folders: []string{}})
	if err != nil {
		test.Fatal(err)
	}
	width := func(text string) float32 { return measurer.MeasureText(text, 16, draw2d.Regular, 1).Width }
	if emoji, question := width("😀"), width("?"); emoji != question {
		test.Fatalf("the emoji measures %v, a question mark %v", emoji, question)
	}
}
