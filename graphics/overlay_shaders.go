package graphics

// OverlayVertexShader places the overlay's rectangles, given in screen pixels from the top-left
// corner, and passes on where they read the atlas and the pictures (in their pixels; a picture
// position below zero reads no picture) and their colour.
// Attributes: 0 screen position, 1 atlas position, 2 colour, 3 picture position, 4 backdrop (1 for
// a quad that shows the blurred snapshot, 0 otherwise). A picture to read at the nearest pixel has
// SharpPictureOffset added to its position's x.
const OverlayVertexShader = `
#version 410
layout(location = 0) in vec2 screenPosition;
layout(location = 1) in vec2 atlasPosition;
layout(location = 2) in vec4 vertexColor;
layout(location = 3) in vec2 picturePosition;
layout(location = 4) in float backdropFlag;
uniform vec2 screenSize;
uniform vec2 atlasSize;
uniform vec2 picturesSize;
out vec2 atlasCoordinate;
out vec2 pictureCoordinate;
flat out float sharp;
out vec4 color;
out float backdrop;

void main() {
    vec2 normalized = screenPosition / screenSize;
    gl_Position = vec4(normalized.x * 2.0 - 1.0, 1.0 - normalized.y * 2.0, 0.0, 1.0);
    atlasCoordinate = atlasPosition / atlasSize;
    vec2 picture = picturePosition;
    sharp = picture.x >= 65536.0 ? 1.0 : 0.0;
    picture.x -= sharp * 65536.0;
    pictureCoordinate = picture.x < 0.0 ? vec2(-1.0) : picture / picturesSize;
    color = vertexColor;
    backdrop = backdropFlag;
}
`

// SharpPictureOffset marks a quad's picture position as one to read at the nearest pixel: a
// picture atlas is never this wide, and the vertex layout stays as it is.
const SharpPictureOffset = 65536

// OverlayFragmentShader paints the colour where the atlas (a grey coverage mask: glyphs, icons,
// rounded corners) covers. Colours are already the ones shown on screen, so they are not
// gamma-corrected like the lit scene.
// An image's quads take their colours from the pictures, tinted by the colour; a backdrop's take
// them from the blurred snapshot of the frame, where the quad is on screen.
const OverlayFragmentShader = `
#version 410
in vec2 atlasCoordinate;
in vec2 pictureCoordinate;
flat in float sharp;
in vec4 color;
in float backdrop;
uniform sampler2D atlas;
uniform sampler2D pictures;
uniform vec2 picturesSize;
uniform sampler2D blurred;
uniform vec2 screenSize;
out vec4 fragmentColor;

void main() {
    float coverage = texture(atlas, atlasCoordinate).r;
    vec4 painted = color;
    if (pictureCoordinate.x >= 0.0) {
        vec2 read = pictureCoordinate;
        if (sharp > 0.5) {
            read = (floor(read * picturesSize) + 0.5) / picturesSize; // the middle of the nearest pixel
        }
        painted *= texture(pictures, read);
    }
    if (backdrop > 0.5) {
        painted = vec4(texture(blurred, gl_FragCoord.xy / screenSize).rgb, 1.0);
    }
    fragmentColor = vec4(painted.rgb, painted.a * coverage);
}
`
