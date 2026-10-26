package scene

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/akrck02/go-renderer/draw2d"
)

// builtInTexture is how one built-in texture is made: its side in pixels and its brightness at
// each point (u and v from 0 to 1), which repeats past 1 without a seam.
type builtInTexture struct {
	name       string
	side       int
	brightness func(u, v float64) float64
}

// The sides of the built-in textures, in pixels: larger for those with more detail in a repeat.
const (
	detailedSide = 128
	simpleSide   = 64
)

// builtInTextures lists the built-in textures in the order BuiltInTextureNames gives them.
var builtInTextures = []builtInTexture{
	{"cobbles", detailedSide, cobbles},
	{"masonry", detailedSide, masonry},
	{"plaster", detailedSide, plaster},
	{"shingles", detailedSide, shingles},
	{"planks", detailedSide, planks},
	{"grass", detailedSide, grass},
	{"panels", detailedSide, panels},
	{"grid", simpleSide, grid},
	{"stripes", simpleSide, stripes},
	{"carbon", simpleSide, carbon},
	{"hexagons", detailedSide, hexagons},
	{"worn", detailedSide, worn},
	{"camouflage", detailedSide, camouflage},
	{"windows", detailedSide, windows},
}

// madeTextures keeps the built-in textures made so far, by name.
var madeTextures = struct {
	sync.Mutex
	byName map[string]*Texture
}{byName: map[string]*Texture{}}

// BuiltInTextureNames lists the textures BuiltInTexture makes.
func BuiltInTextureNames() []string {
	names := make([]string, len(builtInTextures))
	for index, texture := range builtInTextures {
		names[index] = texture.name
	}
	return names
}

// BuiltInTexture returns a texture made in Go, by name: surfaces (cobbles, masonry, plaster,
// shingles, planks, grass) and tiling details (panels, grid, stripes, carbon, hexagons, worn,
// camouflage, windows). Each is a grey square, a power of two a side, that repeats without a seam:
// a material tints it with its BaseColor and sizes it with its TextureScale. It is made the first
// time it is asked for and kept, so every material that asks shares one texture; found is false
// for any other name.
func BuiltInTexture(name string) (texture *Texture, found bool) {
	madeTextures.Lock()
	defer madeTextures.Unlock()
	if texture, found = madeTextures.byName[name]; found {
		return texture, true
	}
	for _, recipe := range builtInTextures {
		if recipe.name == name {
			pixels := recipe.paint()
			texture = &Texture{Picture: draw2d.NewImage(pixels), Gain: gainToAverageOne(pixels)}
			madeTextures.byName[name] = texture
			return texture, true
		}
	}
	return nil, false
}

// paint draws the texture's brightness at the middle of each of its pixels.
func (recipe builtInTexture) paint() *image.NRGBA {
	pixels := image.NewNRGBA(image.Rect(0, 0, recipe.side, recipe.side))
	for y := range recipe.side {
		for x := range recipe.side {
			u, v := (float64(x)+0.5)/float64(recipe.side), (float64(y)+0.5)/float64(recipe.side)
			grey := uint8(255*clampToUnit(recipe.brightness(u, v)) + 0.5)
			pixels.SetNRGBA(x, y, color.NRGBA{grey, grey, grey, 255})
		}
	}
	return pixels
}

// gainToAverageOne is the gain that makes a grey picture average 1 in linear light, as the GPU
// reads it (its pixels are sRGB).
func gainToAverageOne(pixels *image.NRGBA) float32 {
	total := 0.0
	for index := 0; index < len(pixels.Pix); index += 4 {
		total += linearOf(pixels.Pix[index])
	}
	return float32(float64(len(pixels.Pix)/4) / total)
}

// linearOf is an sRGB channel's value in linear light, from 0 to 1.
func linearOf(channel uint8) float64 {
	value := float64(channel) / 255
	if value <= 0.04045 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}
