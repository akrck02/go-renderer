package metal

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
)

// instance is one quad as the Metal shader reads it: 52 bytes, against the 240 of the OpenGL
// path's six vertices. Its layout matches flatInstance in flat_darwin.h and OverlayInstance in the
// scene's shaders.
type instance struct {
	screen  [4]float32 // left, top, right, bottom on screen, in pixels
	atlas   [4]float32 // left, top, right, bottom in the atlas, in pixels
	picture [4]float32 // left, top, right, bottom in the pictures, in pixels; -1 for none
	color   [4]uint8   // straight RGBA
}

// noPicture is the picture place of a quad that shows none; backdropPicture marks a backdrop's
// quad, which shows the blurred snapshot, without making the instance any bigger.
var (
	noPicture       = [4]float32{-1, -1, -1, -1}
	backdropPicture = [4]float32{-2, -2, -2, -2}
)

// packInstances turns the painter's quads into instances, reusing into's memory. Backdrops are
// packed only for a renderer that blurs them; the others leave them out, and their panels show
// over what is behind, unblurred.
func packInstances(quads []overlay.Quad, into []instance, withBackdrops bool) []instance {
	packed := into[:0]
	for _, quad := range quads {
		picture := noPicture
		if quad.Pictured {
			picture = quad.Picture
		}
		if quad.Backdrop {
			if !withBackdrops {
				continue
			}
			picture = backdropPicture
		}
		packed = append(packed, instance{
			screen:  quad.Screen,
			atlas:   quad.Atlas,
			picture: picture,
			color:   [4]uint8{colourByte(quad.Color[0]), colourByte(quad.Color[1]), colourByte(quad.Color[2]), colourByte(quad.Color[3])},
		})
	}
	return packed
}

// colourByte turns a colour channel from 0–1 into 0–255, rounding to the nearest.
func colourByte(channel float32) uint8 {
	return uint8(math.Round(float64(min(max(channel, 0), 1)) * 255))
}

// snapshot is one of the overlay's snapshots as the Objective-C side reads it (flatSnapshot in
// flat_darwin.h): before which instance it is taken, how many times it is halved, and the blur's
// weights at that size, worked out here so Go keeps the blur's arithmetic in one place.
type snapshot struct {
	at       int32
	halvings int32
	weights  [graphics.BackdropTaps + 1]float32
}

// packSnapshots turns the painter's snapshots into the Objective-C side's, reusing into's memory.
func packSnapshots(snapshots []overlay.Snapshot, into []snapshot) []snapshot {
	packed := into[:0]
	for _, taken := range snapshots {
		halvings := graphics.BackdropHalvings(taken.Blur)
		packed = append(packed, snapshot{
			at:       int32(taken.At),
			halvings: int32(halvings),
			weights:  graphics.BackdropWeights(taken.Blur / float32(int(1)<<halvings)),
		})
	}
	return packed
}
