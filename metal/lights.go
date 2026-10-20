package metal

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
)

// The flat renderer's 2D lights (see overlay.Lighting), laid out for the Objective-C side
// (flat_darwin.h), byte for byte.

// light is one light as the shader reads it: where it is on screen, its colour (alpha: its
// intensity), the picture it shines through (-1 for a point light) and how far it is full. 64 bytes.
type light struct {
	screen  [4]float32
	color   [4]float32
	picture [4]float32
	inner   float32
	unused  [3]float32
}

// lighting is one lighting: before the instance at, the lights from first to last are added into
// a picture of light filled with the ambient (already at half, as light is kept), which then
// multiplies the area (left, top, right, bottom on screen). 48 bytes.
type lighting struct {
	at, first, last, unused int32
	ambient                 [4]float32
	area                    [4]float32
}

// packLights turns the painter's lights into the Objective-C side's, reusing into's memory.
func packLights(lights []overlay.LightQuad, into []light) []light {
	packed := into[:0]
	for _, each := range lights {
		picture := noPicture
		if each.Pictured {
			picture = each.Picture
		}
		packed = append(packed, light{screen: each.Screen, color: each.Color, picture: picture, inner: each.Inner})
	}
	return packed
}

// packLightings turns the painter's lightings into the Objective-C side's, reusing into's memory.
func packLightings(lightings []overlay.Lighting, into []lighting) []lighting {
	packed := into[:0]
	for _, each := range lightings {
		ambient := graphics.AmbientLight(each.Ambient)
		packed = append(packed, lighting{at: int32(each.At), first: int32(each.First), last: int32(each.Last),
			ambient: [4]float32{ambient[0], ambient[1], ambient[2], 1}, area: each.Area})
	}
	return packed
}
