package gles

// The shaders are GLSL ES 1.00, which OpenGL ES 2 and 3 and WebGL2 all read. They translate the
// desktop ones in graphics (overlay_shaders.go, scene_shaders.go): the overlay exactly, the scene
// in its simplest form (sun and sky light, fog, translucent water; no shadows, patterns, waves or
// wind). Attributes get their indices from the program (attributeNames), since GLSL ES 1.00 has
// no layout qualifiers.

// The attributes both programs read, by index.
var (
	quadAttributes     = []string{"screenPosition", "atlasPosition", "vertexColor", "picturePosition", "backdropFlag"}
	backdropAttributes = []string{"corner"}
	sceneAttributes    = []string{"vertexPosition", "vertexNormal", "vertexColor"}
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
    sharp = picture.x >= 65536.0 ? 1.0 : 0.0; // graphics.SharpPictureOffset: read at the nearest pixel
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
uniform mat4 viewProjection;
uniform mat4 model;
uniform mat3 normalMatrix;
uniform vec4 tint;
uniform float verticalScale;
varying vec3 worldPosition;
varying vec3 worldNormal;
varying vec4 surfaceColor;

void main() {
    vec4 world = model * vec4(vertexPosition, 1.0);
    world.y *= verticalScale;
    worldPosition = world.xyz;
    vec3 normal = normalMatrix * vertexNormal;
    normal.y /= verticalScale;
    worldNormal = normal;
    surfaceColor = vertexColor * tint;
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
uniform vec4 baseColor;
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

vec3 shadeLit(vec3 color) {
    vec3 normal = normalize(worldNormal);
    if (!gl_FrontFacing) normal = -normal;
    float sunlight = max(dot(normal, normalize(sunDirection)), 0.0);
    vec3 skylight = mix(groundColor, skyColor, normal.y * 0.5 + 0.5) * ambient;
    return color * (skylight + sunColor * sunlight);
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
// sum of itself and four texels on each side (graphics.BackdropTaps). With the centre's weight one
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
