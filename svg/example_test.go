package svg_test

import (
	"fmt"

	"github.com/akrck02/go-renderer/svg"
)

// An icon read and drawn at 32 pixels: a filled circle, its centre painted, its corner not.
func ExampleDrawing_Picture() {
	icon := []byte(`<svg viewBox="0 0 16 16"><circle cx="8" cy="8" r="6" fill="#3366cc"/></svg>`)
	drawing, err := svg.Read(icon)
	if err != nil {
		fmt.Println(err)
		return
	}
	picture := drawing.Picture(32, 32)
	fmt.Println(picture.Pixels.NRGBAAt(16, 16), picture.Pixels.NRGBAAt(1, 1).A)
	// Output: {51 102 204 255} 0
}
