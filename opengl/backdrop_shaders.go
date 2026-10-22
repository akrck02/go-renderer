package opengl

// backdropVertexShader covers the target with one triangle made from the vertex number alone, so a
// blur pass needs no vertex buffer.
const backdropVertexShader = `
#version 410
out vec2 coordinate;

void main() {
    vec2 corner = vec2((gl_VertexID << 1) & 2, gl_VertexID & 2);
    coordinate = corner;
    gl_Position = vec4(corner * 2.0 - 1.0, 0.0, 1.0);
}
`

// backdropBlurShader is one pass of a separable Gaussian blur: along step, a texel is the weighted
// sum of itself and BackdropTaps texels on each side.
const backdropBlurShader = `
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
