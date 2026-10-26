package gles

// The shaders are GLSL ES 1.00, which OpenGL ES 2 and 3 and WebGL2 all read. They translate the
// desktop ones in opengl (overlay_shaders.go, scene_shaders.go): the overlay exactly, the scene
// in its simplest form (sun and sky light, textures, fog, translucent water; no shadows, waves or
// wind). Attributes get their indices from the program (attributeNames), since GLSL ES 1.00 has
// no layout qualifiers.

// The attributes both programs read, by index.
var (
	quadAttributes     = []string{"screenPosition", "atlasPosition", "vertexColor", "picturePosition", "backdropFlag"}
	backdropAttributes = []string{"corner"}
	sceneAttributes    = []string{"vertexPosition", "vertexNormal", "vertexColor", "vertexUV"}
)

// quadVertexShader places the overlay's rectangles, given in screen pixels from the top-left
// corner, and passes on where they read the atlas and the pictures (in their pixels; a picture
// position below zero reads no picture) and their colour.
const quadVertexShader = `#version 100
attribute vec2 screenPosition;
attribute vec2 atlasPosition;
attribute vec4 vertexColor;
attribute vec2 picturePosition;
attribute float backdropFlag;
uniform vec2 screenSize;
uniform vec2 atlasSize;
uniform vec2 picturesSize;
varying vec2 atlasCoordinate;
varying vec2 pictureCoordinate;
varying float sharp;
varying vec2 pictureTexel;
varying vec4 color;
varying float backdrop;

void main() {
    vec2 normalized = screenPosition / screenSize;
    gl_Position = vec4(normalized.x * 2.0 - 1.0, 1.0 - normalized.y * 2.0, 0.0, 1.0);
    atlasCoordinate = atlasPosition / atlasSize;
    vec2 picture = picturePosition;
    sharp = picture.x >= 65536.0 ? 1.0 : 0.0; // shading.SharpPictureOffset: read at the nearest pixel
    picture.x -= sharp * 65536.0;
    pictureCoordinate = picture.x < 0.0 ? vec2(-1.0) : picture / picturesSize;
    pictureTexel = 1.0 / picturesSize; // a uniform of the fragment shader would get another precision: the program would not link
    color = vertexColor;
    backdrop = backdropFlag;
}
`

// quadFragmentShader paints the colour where the atlas (a grey coverage mask: glyphs, icons,
// rounded corners) covers; an image's quads take their colours from the pictures, tinted by it,
// and a backdrop's from the blurred snapshot of the frame, where the quad is on screen.
const quadFragmentShader = `#version 100
precision mediump float;
varying vec2 atlasCoordinate;
varying vec2 pictureCoordinate;
varying float sharp;
varying vec2 pictureTexel;
varying vec4 color;
varying float backdrop;
uniform sampler2D atlas;
uniform sampler2D pictures;
uniform sampler2D blurred;
uniform vec2 frameSize; // the screen's size: screenSize is highp to the vertex stage, so not shared

void main() {
    float coverage = texture2D(atlas, atlasCoordinate).r;
    vec4 painted = color;
    if (pictureCoordinate.x >= 0.0) {
        vec2 read = pictureCoordinate;
        if (sharp > 0.5) {
            read = (floor(read / pictureTexel) + 0.5) * pictureTexel; // the middle of the nearest pixel
        }
        painted *= texture2D(pictures, read);
    }
    if (backdrop > 0.5) {
        painted = vec4(texture2D(blurred, gl_FragCoord.xy / frameSize).rgb, 1.0);
    }
    gl_FragColor = vec4(painted.rgb, painted.a * coverage);
}
`

// sceneVertexShader places a mesh in the world, raised by the vertical exaggeration, and passes on
// its world position, normal and colour (the vertex colour times the instance's tint). The normal
// matrix (the inverse transpose of the model's rotation and scale, which keeps normals
// perpendicular under scales that differ per axis) comes worked out, as GLSL ES 1.00 has no
// inverse.
const sceneVertexShader = `#version 100
attribute vec3 vertexPosition;
attribute vec3 vertexNormal;
attribute vec4 vertexColor;
attribute vec2 vertexUV;
uniform mat4 viewProjection;
uniform mat4 model;
uniform mat3 normalMatrix;
uniform vec4 tint;
uniform float verticalScale;
uniform vec4 texturePlacement; // scale along u and v, then offset along u and v
varying vec3 worldPosition;
varying vec3 worldNormal;
varying vec4 surfaceColor;
varying vec2 surfaceUV;

void main() {
    vec4 world = model * vec4(vertexPosition, 1.0);
    world.y *= verticalScale;
    worldPosition = world.xyz;
    vec3 normal = normalMatrix * vertexNormal;
    normal.y /= verticalScale;
    worldNormal = normal;
    surfaceColor = vertexColor * tint;
    surfaceUV = vertexUV * texturePlacement.xy + texturePlacement.zw;
    gl_Position = viewProjection * world;
}
`

// sceneFragmentShader lights a surface by kind: 0 lit, 1 unlit, 2 water, 3 waterfall, 4 glow (the
// values of scene.MaterialKind). Colours are linear and turned to the screen's gamma at the end,
// like the desktop renderer.
const sceneFragmentShader = `#version 100
precision highp float;
varying vec3 worldPosition;
varying vec3 worldNormal;
varying vec4 surfaceColor;
varying vec2 surfaceUV;
uniform vec4 baseColor;
uniform int textured;              // 1: the material's texture multiplies its colour
uniform sampler2D surfaceTexture;
uniform int kind;
uniform vec3 cameraPosition;
uniform vec3 sunDirection;
uniform vec3 sunColor;
uniform vec3 skyColor;
uniform vec3 groundColor;
uniform float ambient;
uniform float brightness;
uniform vec3 fogColor;
uniform vec2 fogRange;
uniform int lightCount;            // the point lights this draw takes, up to four
uniform vec4 lightPositions[4];    // xyz: where, w: the outer radius
uniform vec4 lightColors[4];       // rgb: linear colour times intensity, w: the inner radius

// lampLight is what the draw's point lights bring, as opengl's sceneFragmentShader's.
vec3 lampLight(vec3 normal) {
    vec3 total = vec3(0.0);
    for (int index = 0; index < 4; index++) {
        if (index >= lightCount) break;
        vec3 towardsLight = lightPositions[index].xyz - worldPosition;
        float distance = length(towardsLight);
        float fade = 1.0 - smoothstep(lightColors[index].w, lightPositions[index].w, distance);
        total += lightColors[index].rgb * fade * max(dot(normal, towardsLight / max(distance, 1e-4)), 0.0);
    }
    return total;
}

vec3 shadeLit(vec3 color) {
    vec3 normal = normalize(worldNormal);
    if (!gl_FrontFacing) normal = -normal;
    float sunlight = max(dot(normal, normalize(sunDirection)), 0.0);
    vec3 skylight = mix(groundColor, skyColor, normal.y * 0.5 + 0.5) * ambient;
    return color * (skylight + sunColor * sunlight + lampLight(normal));
}

// textureColor reads the surface's texture as linear light: a picture's colours are sRGB, as a
// PNG's are, and OpenGL ES 2 has no texture format that turns them by itself.
vec4 textureColor(vec2 uv) {
    vec4 texel = texture2D(surfaceTexture, uv);
    return vec4(pow(texel.rgb, vec3(2.2)), texel.a);
}

// shadeWater gives the surface a sky reflection that grows at grazing angles, over its own colour.
vec4 shadeWater(vec4 color) {
    vec3 towardsCamera = normalize(cameraPosition - worldPosition);
    float grazing = pow(1.0 - max(towardsCamera.y, 0.0), 4.0);
    vec3 lit = color.rgb * (skyColor * ambient + sunColor * max(normalize(sunDirection).y, 0.0));
    return vec4(mix(lit, skyColor * brightness, grazing * 0.6), max(color.a, 0.75));
}

void main() {
    vec4 color = surfaceColor * baseColor;
    if (textured == 1) color *= textureColor(surfaceUV);
    if (kind == 0) {
        color.rgb = shadeLit(color.rgb);
    } else if (kind == 4) {
        color.rgb = max(shadeLit(color.rgb), color.rgb * (1.0 - brightness) * 1.4);
    } else if (kind == 2) {
        color = shadeWater(color);
    } else {
        color.rgb *= brightness;
    }
    if (fogRange.y > fogRange.x) {
        float fogAmount = smoothstep(fogRange.x, fogRange.y, distance(cameraPosition, worldPosition));
        color.rgb = mix(color.rgb, fogColor, fogAmount);
    }
    gl_FragColor = vec4(pow(max(color.rgb, vec3(0.0)), vec3(1.0 / 2.2)), color.a);
}
`

// backdropVertexShader covers the target with two triangles, their corners from 0 to 1.
const backdropVertexShader = `#version 100
attribute vec2 corner;
varying vec2 coordinate;

void main() {
    coordinate = corner;
    gl_Position = vec4(corner * 2.0 - 1.0, 0.0, 1.0);
}
`

// backdropBlurShader is one pass of a separable Gaussian blur along step: a texel is the weighted
// sum of itself and four texels on each side (shading.BackdropTaps). With the centre's weight one
// and the others zero, read between texels, it averages four texels into one: a halving.
const backdropBlurShader = `#version 100
precision mediump float;
varying vec2 coordinate;
uniform sampler2D source;
uniform vec2 step;
uniform float weights[5];

void main() {
    vec4 sum = texture2D(source, coordinate) * weights[0];
    for (int tap = 1; tap <= 4; tap++) {
        sum += texture2D(source, coordinate + step * float(tap)) * weights[tap];
        sum += texture2D(source, coordinate - step * float(tap)) * weights[tap];
    }
    gl_FragColor = sum;
}
`

// lightAttributes are what the lights' program reads, by index.
var lightAttributes = []string{"screenPosition", "cornerPosition", "lightColor", "picturePosition", "innerShare"}

// lightVertexShader places a light's square, given in screen pixels from the top-left corner, as
// opengl's lightVertexShader does.
const lightVertexShader = `#version 100
attribute vec2 screenPosition;
attribute vec2 cornerPosition;
attribute vec4 lightColor;
attribute vec2 picturePosition;
attribute float innerShare;
uniform vec2 screenSize;
uniform vec2 picturesSize;
varying vec2 offset;
varying vec4 color;
varying vec2 pictureCoordinate;
varying float inner;

void main() {
    vec2 normalized = screenPosition / screenSize;
    gl_Position = vec4(normalized.x * 2.0 - 1.0, 1.0 - normalized.y * 2.0, 0.0, 1.0);
    offset = cornerPosition;
    color = lightColor;
    pictureCoordinate = picturePosition.x < 0.0 ? vec2(-1.0) : picturePosition / picturesSize;
    inner = innerShare;
}
`

// lightFragmentShader is how much light a light adds, at half, as opengl's lightFragmentShader.
const lightFragmentShader = `#version 100
precision mediump float;
varying vec2 offset;
varying vec4 color;
varying vec2 pictureCoordinate;
varying float inner;
uniform sampler2D pictures;

void main() {
    vec3 shape;
    if (pictureCoordinate.x >= 0.0) {
        vec4 picture = texture2D(pictures, pictureCoordinate);
        shape = picture.rgb * picture.a;
    } else {
        shape = vec3(1.0 - smoothstep(min(inner, 0.999), 1.0, length(offset)));
    }
    gl_FragColor = vec4(color.rgb * color.a * shape * 0.5, 1.0);
}
`

// lightAreaVertexShader stretches the corners (from 0 to 1) over the area a lighting lights, in
// screen pixels from the top-left corner (left, top, right, bottom), and passes on where the
// picture of light is read there.
const lightAreaVertexShader = `#version 100
attribute vec2 corner;
uniform vec4 area;
uniform vec2 screenSize;
varying vec2 coordinate;

void main() {
    vec2 normalized = mix(area.xy, area.zw, corner) / screenSize;
    coordinate = vec2(normalized.x, 1.0 - normalized.y);
    gl_Position = vec4(normalized.x * 2.0 - 1.0, 1.0 - normalized.y * 2.0, 0.0, 1.0);
}
`

// lightMultiplyShader reads the picture of light over the area lit (drawn with
// lightAreaVertexShader); the blend multiplies the frame by it twice over.
const lightMultiplyShader = `#version 100
precision mediump float;
varying vec2 coordinate;
uniform sampler2D light;

void main() {
    float noise = fract(sin(dot(gl_FragCoord.xy, vec2(12.9898, 78.233))) * 43758.5453) - 0.5; // breaks the bands of a dark fade
    gl_FragColor = vec4(texture2D(light, coordinate).rgb + noise / 255.0, 1.0);
}
`
