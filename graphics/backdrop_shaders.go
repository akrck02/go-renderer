package graphics

import "math"

// BackdropTaps is how many texels each pass of the backdrop's blur reads on each side of a texel:
// with the picture shrunk first (see the backends), a few taps blur as widely as many would.
const BackdropTaps = 4

// BackdropVertexShader covers the target with one triangle made from the vertex number alone, so a
// blur pass needs no vertex buffer.
const BackdropVertexShader = `
#version 410
out vec2 coordinate;

void main() {
    vec2 corner = vec2((gl_VertexID << 1) & 2, gl_VertexID & 2);
    coordinate = corner;
    gl_Position = vec4(corner * 2.0 - 1.0, 0.0, 1.0);
}
`

// BackdropBlurShader is one pass of a separable Gaussian blur: along step, a texel is the weighted
// sum of itself and BackdropTaps texels on each side.
const BackdropBlurShader = `
#version 410
in vec2 coordinate;
uniform sampler2D source;
uniform vec2 step;
uniform float weights[5];
out vec4 fragmentColor;

void main() {
    vec4 sum = texture(source, coordinate) * weights[0];
    for (int tap = 1; tap <= 4; tap++) {
        sum += texture(source, coordinate + step * float(tap)) * weights[tap];
        sum += texture(source, coordinate - step * float(tap)) * weights[tap];
    }
    fragmentColor = sum;
}
`

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
