package graphics

// OverlayVertexShader places the overlay's rectangles, given in screen pixels from the top-left
// corner, and passes on where they read the atlas (in atlas pixels) and their colour.
// Attributes: 0 screen position, 1 atlas position, 2 colour.
const OverlayVertexShader = `
#version 410
layout(location = 0) in vec2 screenPosition;
layout(location = 1) in vec2 atlasPosition;
layout(location = 2) in vec4 vertexColor;
uniform vec2 screenSize;
uniform vec2 atlasSize;
out vec2 atlasCoordinate;
out vec4 color;

void main() {
    vec2 normalized = screenPosition / screenSize;
    gl_Position = vec4(normalized.x * 2.0 - 1.0, 1.0 - normalized.y * 2.0, 0.0, 1.0);
    atlasCoordinate = atlasPosition / atlasSize;
    color = vertexColor;
}
`

// OverlayFragmentShader paints the colour where the atlas (a grey coverage mask: glyphs, icons,
// rounded corners) covers. Colours are already the ones shown on screen, so they are not
// gamma-corrected like the lit scene.
const OverlayFragmentShader = `
#version 410
in vec2 atlasCoordinate;
in vec4 color;
uniform sampler2D atlas;
out vec4 fragmentColor;

void main() {
    float coverage = texture(atlas, atlasCoordinate).r;
    fragmentColor = vec4(color.rgb, color.a * coverage);
}
`
