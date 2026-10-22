// Package shading holds what the backends' shaders agree on, whatever their language (GLSL in
// opengl and gles, MSL in metal, the PICA200's in ctr): how a quad asks for a sharp picture, how
// ambient light is kept, and the backdrop blur's weights. The shader sources themselves live in
// the backend that compiles them.
package shading

import "math"

// SharpPictureOffset marks a quad's picture position as one to read at the nearest pixel: a
// picture atlas is never this wide, and the vertex layout stays as it is.
const SharpPictureOffset = 65536

// AmbientLight is the colour a picture of light is cleared to for an ambient colour whose alpha is
// its intensity: at half, as all light is kept.
func AmbientLight(ambient [4]float32) [3]float32 {
	const half = 0.5
	return [3]float32{ambient[0] * ambient[3] * half, ambient[1] * ambient[3] * half, ambient[2] * ambient[3] * half}
}

// BackdropTaps is how many texels each pass of the backdrop's blur reads on each side of a texel:
// with the picture shrunk first (see the backends), a few taps blur as widely as many would.
const BackdropTaps = 4

// BackdropWeights are the blur's weights for a standard deviation in texels: the centre's, then
// each tap's on one side, adding up to one over the whole kernel.
func BackdropWeights(deviation float32) [BackdropTaps + 1]float32 {
	var weights [BackdropTaps + 1]float32
	if deviation <= 0 {
		weights[0] = 1
		return weights
	}
	total := float32(0)
	for tap := range weights {
		distance := float64(tap)
		weights[tap] = float32(math.Exp(-distance * distance / (2 * float64(deviation*deviation))))
		total += weights[tap]
		if tap > 0 {
			total += weights[tap] // the same tap on the other side
		}
	}
	for tap := range weights {
		weights[tap] /= total
	}
	return weights
}

// BackdropHalvings is how many times a snapshot is halved before it is blurred, for a standard
// deviation in screen pixels: until the deviation is a couple of texels, which the taps cover.
// Halving is cheap and smooth; each step quarters the blur's work.
func BackdropHalvings(deviation float32) int {
	const texelsTheTapsCover = 2
	const mostHalvings = 5
	halvings := 0
	for deviation > texelsTheTapsCover && halvings < mostHalvings {
		deviation /= 2
		halvings++
	}
	return halvings
}
