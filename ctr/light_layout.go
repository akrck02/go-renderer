package ctr

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
)

// The PICA200 has no fragment shaders, so a point light cannot work out its own fade as the other
// backends' shaders do. It reads it from a texture instead: discs of light, full within an inner
// share of their radius and fading smoothly to nothing at its edge, one for each of a few inner
// shares, made once. A light takes the disc nearest its own.
const (
	discSide        = 64                        // pixels across one disc
	discsPerRow     = 4                         // discs side by side in the texture
	discLevels      = discsPerRow * discsPerRow // inner shares, from 0 to 1
	discTextureSide = discSide * discsPerRow
)

// The picture of light is drawn at half the screen's resolution, into a texture whose sides are
// powers of two, as the PICA200's must be: 256 × 128 holds half the top screen's 400 × 240, and
// a light pixel is two screen pixels a side.
const (
	lightTextureWidth  = 256
	lightTextureHeight = 128
	lightResolution    = 2 // screen pixels a side to one of the light's
)

// lightDiscs draws the discs, one byte of brightness a pixel, row by row from the top. The
// texel centres at a disc's edge are dark, so reading between discs never takes a neighbour's.
func lightDiscs() []byte {
	pixels := make([]byte, discTextureSide*discTextureSide)
	const half = discSide / 2
	const edge = half - 0.5 // from the centre to the centres of the texels at the edge
	for level := range discLevels {
		inner := min(innerShareOf(level), 0.999)
		originX, originY := (level%discsPerRow)*discSide, (level/discsPerRow)*discSide
		for y := range discSide {
			for x := range discSide {
				distance := math.Hypot(float64(x)+0.5-half, float64(y)+0.5-half) / edge
				brightness := 1 - smoothStep(float64(inner), 1, distance)
				pixels[(originY+y)*discTextureSide+originX+x] = byte(brightness*255 + 0.5)
			}
		}
	}
	return pixels
}

// innerShareOf is the inner share of a disc.
func innerShareOf(level int) float32 { return float32(level) / (discLevels - 1) }

// discFor is the disc nearest an inner share.
func discFor(inner float32) int {
	return int(min(max(inner, 0), 1)*(discLevels-1) + 0.5)
}

// smoothStep is GLSL's smoothstep: 0 below low, 1 above high, a smooth curve between; a step at
// high when the two meet.
func smoothStep(low, high, value float64) float64 {
	if high <= low {
		return float64(boolToNumber(value >= high))
	}
	share := min(max((value-low)/(high-low), 0), 1)
	return share * share * (3 - 2*share)
}

// discArea is where a disc is in the texture, from 0 to 1 from the top-left: left, top, right,
// bottom, inset half a texel so that its edge reads the edge texels' centres.
func discArea(level int) [4]float32 {
	const texel = 1.0 / discTextureSide
	left := float32(level%discsPerRow*discSide)*texel + texel/2
	top := float32(level/discsPerRow*discSide)*texel + texel/2
	side := float32(discSide-1) * texel
	return [4]float32{left, top, left + side, top + side}
}

// writeLightVertices writes the corners of the lights, as rectangles of the overlay's layout; after
// them the rectangle that fills the picture of light with an ambient; and after that, for each
// lighting, the rectangle that lays the picture of light over its area.
// A point light reads its disc as its first texture, a sprite light its picture as its second;
// runs group them as the quads' do. Colours are at half, as all light is kept (see
// overlay.Lighting).
func writeLightVertices(vertices []float32, runs []quadRun, lights []overlay.LightQuad, lightings []overlay.Lighting, picturesSize int) []quadRun {
	toPictures := 1 / float32(max(picturesSize, 1))
	_ = vertices[(len(lights)+1+len(lightings))*4*floatsPerQuadVertex-1] // one bounds check for them all
	for index := range lights {
		light := &lights[index]
		if len(runs) == 0 || runs[len(runs)-1].pictured != light.Pictured {
			runs = append(runs, quadRun{first: index, pictured: light.Pictured})
		}
		runs[len(runs)-1].count++
		var disc, picture [4]float32
		if light.Pictured {
			for side := range picture {
				picture[side] = light.Picture[side] * toPictures
			}
		} else {
			disc = discArea(discFor(light.Inner))
		}
		colour := halfLight(light.Color)
		writeRectangle(vertices[index*4*floatsPerQuadVertex:], light.Screen, disc, &colour, picture)
	}
	white := [4]float32{1, 1, 1, 1}
	canvas := [4]float32{0, 0, lightCanvasWidth, lightCanvasHeight}
	writeRectangle(vertices[len(lights)*4*floatsPerQuadVertex:], canvas, [4]float32{}, &white, [4]float32{})
	for index, lighting := range lightings {
		area := lighting.Area
		lit := [4]float32{area[0] / lightCanvasWidth, area[1] / lightCanvasHeight, area[2] / lightCanvasWidth, area[3] / lightCanvasHeight}
		writeRectangle(vertices[(len(lights)+1+index)*4*floatsPerQuadVertex:], area, lit, &white, [4]float32{})
	}
	return runs
}

// The screen pixels the picture of light covers.
const (
	lightCanvasWidth  = lightTextureWidth * lightResolution
	lightCanvasHeight = lightTextureHeight * lightResolution
)

// writeRectangle writes a rectangle's four corners as writeQuadVertices does.
func writeRectangle(corners []float32, screen, texture [4]float32, colour *[4]float32, picture [4]float32) {
	left, top, right, bottom := screen[0], screen[1], screen[2], screen[3]
	writeCorner(corners[0:floatsPerQuadVertex], left, top, texture[0], texture[1], colour, picture[0], picture[1])
	writeCorner(corners[floatsPerQuadVertex:2*floatsPerQuadVertex], left, bottom, texture[0], texture[3], colour, picture[0], picture[3])
	writeCorner(corners[2*floatsPerQuadVertex:3*floatsPerQuadVertex], right, bottom, texture[2], texture[3], colour, picture[2], picture[3])
	writeCorner(corners[3*floatsPerQuadVertex:4*floatsPerQuadVertex], right, top, texture[2], texture[1], colour, picture[2], picture[1])
}

// halfLight is a light's colour times its intensity, at half, opaque.
func halfLight(colour [4]float32) [4]float32 {
	ambient := graphics.AmbientLight(colour)
	return [4]float32{min(ambient[0], 1), min(ambient[1], 1), min(ambient[2], 1), 1}
}

// ambientColour is the colour the picture of light is filled with, packed as the PICA200's
// constant colours are: 0xAABBGGRR.
func ambientColour(ambient [4]float32) uint32 {
	light := halfLight(ambient)
	channel := func(value float32) uint32 { return uint32(value*255 + 0.5) }
	return 0xff<<24 | channel(light[2])<<16 | channel(light[1])<<8 | channel(light[0])
}

// Orthographic is citro3d's Mtx_Ortho for a right-handed view: an orthographic projection with
// the depth range the PICA200 wants, [-1, 0], for a picture that is not a screen (a texture drawn
// into), which is not turned as the screens are.
func Orthographic(left, right, bottom, top, near, far float32) graphics.Mat4 {
	var matrix graphics.Mat4
	set(&matrix, 0, 0, 2/(right-left))
	set(&matrix, 0, 3, (left+right)/(left-right))
	set(&matrix, 1, 1, 2/(top-bottom))
	set(&matrix, 1, 3, (bottom+top)/(bottom-top))
	set(&matrix, 2, 2, 1/(near-far))
	set(&matrix, 2, 3, 0.5*(near+far)/(near-far)-0.5)
	set(&matrix, 3, 3, 1)
	return matrix
}

// boolToNumber is 1 for true and 0 for false.
func boolToNumber(flag bool) int {
	if flag {
		return 1
	}
	return 0
}
