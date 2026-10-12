package metal

import (
	"math"

	"github.com/akrck02/go-renderer/opengl/overlay"
)

// instance is one quad as the Metal shader reads it: 36 bytes, against the 192 of the OpenGL
// path's six vertices. Its layout matches flatInstance in flat_darwin.h.
type instance struct {
	screen [4]float32 // left, top, right, bottom on screen, in pixels
	atlas  [4]float32 // left, top, right, bottom in the atlas, in pixels
	color  [4]uint8   // straight RGBA
}

// packInstances turns the painter's quads into instances, reusing into's memory.
func packInstances(quads []overlay.Quad, into []instance) []instance {
	packed := into[:0]
	for _, quad := range quads {
		packed = append(packed, instance{
			screen: quad.Screen,
			atlas:  quad.Atlas,
			color:  [4]uint8{colourByte(quad.Color[0]), colourByte(quad.Color[1]), colourByte(quad.Color[2]), colourByte(quad.Color[3])},
		})
	}
	return packed
}

// colourByte turns a colour channel from 0–1 into 0–255, rounding to the nearest.
func colourByte(channel float32) uint8 {
	return uint8(math.Round(float64(min(max(channel, 0), 1)) * 255))
}
