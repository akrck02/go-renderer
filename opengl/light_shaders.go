package opengl

// 2D lights are drawn in two steps (see overlay.Lighting). First the lights are added into a
// picture of light, cleared to the ambient colour: lightVertexShader and lightFragmentShader draw
// each light as a square, added to what is there. Then lightMultiplyShader lays that picture over
// the frame with a blend that multiplies by twice the light, so light is kept at half: 0.5 leaves
// a colour as it is, 0 makes it black, 1 makes it twice as bright.

// lightVertexShader places a light's square, given in screen pixels from the top-left corner.
// Attributes: 0 screen position, 1 corner (from -1 to 1 across the square, to measure the
// distance from its centre), 2 colour (alpha: intensity), 3 picture position (below zero for a
// point light), 4 inner (the share of the half side the light is full within).
const lightVertexShader = `
#version 410
layout(location = 0) in vec2 screenPosition;
layout(location = 1) in vec2 cornerPosition;
layout(location = 2) in vec4 lightColor;
layout(location = 3) in vec2 picturePosition;
layout(location = 4) in float innerShare;
uniform vec2 screenSize;
uniform vec2 picturesSize;
out vec2 offset;
out vec4 color;
out vec2 pictureCoordinate;
flat out float inner;

void main() {
    vec2 normalized = screenPosition / screenSize;
    gl_Position = vec4(normalized.x * 2.0 - 1.0, 1.0 - normalized.y * 2.0, 0.0, 1.0);
    offset = cornerPosition;
    color = lightColor;
    pictureCoordinate = picturePosition.x < 0.0 ? vec2(-1.0) : picturePosition / picturesSize;
    inner = innerShare;
}
`

// lightFragmentShader is how much light a light adds, at half: its colour times its intensity,
// times its picture for a sprite light, or for a point light full inside inner and fading
// smoothly to nothing at the square's inscribed circle.
const lightFragmentShader = `
#version 410
in vec2 offset;
in vec4 color;
in vec2 pictureCoordinate;
flat in float inner;
uniform sampler2D pictures;
out vec4 fragmentColor;

void main() {
    vec3 shape;
    if (pictureCoordinate.x >= 0.0) {
        vec4 picture = texture(pictures, pictureCoordinate);
        shape = picture.rgb * picture.a;
    } else {
        float distance = length(offset);
        shape = vec3(1.0 - smoothstep(min(inner, 0.999), 1.0, distance));
    }
    fragmentColor = vec4(color.rgb * color.a * shape * 0.5, 1.0);
}
`

// lightAreaVertexShader covers the area a lighting lights (in screen pixels from the top-left
// corner: left, top, right, bottom) with two triangles made from the vertex number alone, and
// passes on where the picture of light is read there.
const lightAreaVertexShader = `
#version 410
uniform vec4 area;
uniform vec2 screenSize;
out vec2 coordinate;
const vec2 corners[6] = vec2[](vec2(0.0, 0.0), vec2(0.0, 1.0), vec2(1.0, 1.0), vec2(0.0, 0.0), vec2(1.0, 1.0), vec2(1.0, 0.0));

void main() {
    vec2 normalized = mix(area.xy, area.zw, corners[gl_VertexID]) / screenSize;
    coordinate = vec2(normalized.x, 1.0 - normalized.y);
    gl_Position = vec4(normalized.x * 2.0 - 1.0, 1.0 - normalized.y * 2.0, 0.0, 1.0);
}
`

// lightMultiplyShader reads the picture of light over the area lit (drawn with
// lightAreaVertexShader). The blend multiplies the frame by it twice over. Light is kept at half
// in bytes, so a dark ambient leaves few steps for a fade: a noise of half a step, different at
// every pixel, breaks the bands they would draw.
const lightMultiplyShader = `
#version 410
in vec2 coordinate;
uniform sampler2D light;
out vec4 fragmentColor;

void main() {
    float noise = fract(sin(dot(gl_FragCoord.xy, vec2(12.9898, 78.233))) * 43758.5453) - 0.5;
    fragmentColor = vec4(texture(light, coordinate).rgb + noise / 255.0, 1.0);
}
`
