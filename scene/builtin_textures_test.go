package scene

import (
	"math"
	"slices"
	"testing"
)

// The names every program can ask BuiltInTexture for.
var builtInNames = []string{"cobbles", "masonry", "plaster", "shingles", "planks", "grass",
	"panels", "grid", "stripes", "carbon", "hexagons", "worn", "camouflage", "windows"}

func TestEveryBuiltInTextureIsASquarePowerOfTwo(test *testing.T) {
	if !slices.Equal(BuiltInTextureNames(), builtInNames) {
		test.Errorf("the built-in textures are %v, wanted %v", BuiltInTextureNames(), builtInNames)
	}
	for _, name := range builtInNames {
		texture, found := BuiltInTexture(name)
		if !found {
			test.Fatalf("%s: not found", name)
		}
		width, height := texture.Picture.Size()
		if width != height || width < 8 || width&(width-1) != 0 {
			test.Errorf("%s: %d×%d, wanted a square power of two", name, width, height)
		}
	}
}

func TestEveryBuiltInTextureIsMadeOnce(test *testing.T) {
	first, _ := BuiltInTexture("cobbles")
	second, _ := BuiltInTexture("cobbles")
	if first != second {
		test.Error("asking twice made two textures")
	}
	if _, found := BuiltInTexture("marble"); found {
		test.Error("an unknown name gave a texture")
	}
}

// TestEveryBuiltInTextureTilesWithoutASeam checks that each texture's brightness one repeat to
// the right, or one repeat down, is the brightness where it started: its last column runs into its
// first, and its last row into its first, as they would inside a larger picture.
func TestEveryBuiltInTextureTilesWithoutASeam(test *testing.T) {
	const samples, tolerance = 97, 1e-9
	for _, recipe := range builtInTextures {
		for step := range samples * samples {
			u, v := float64(step%samples)/samples, float64(step/samples)/samples
			here := recipe.brightness(u, v)
			right, below := recipe.brightness(u+1, v), recipe.brightness(u, v+1)
			if math.Abs(right-here) > tolerance || math.Abs(below-here) > tolerance {
				test.Errorf("%s at (%.3f, %.3f): %.3f, but %.3f a repeat to the right and %.3f a repeat down",
					recipe.name, u, v, here, right, below)
				break
			}
		}
	}
}
