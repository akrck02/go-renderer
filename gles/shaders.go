package gles

// The shaders are GLSL ES 3.00, which WebGL2 and OpenGL ES 3 read alike. They translate the desktop
// ones in graphics (overlay_shaders.go, scene_shaders.go): the overlay exactly, the scene in its
// simplest form (sun and sky light, fog, translucent water; no shadows, patterns, waves or wind).

// quadVertexShader places the overlay's rectangles, given in screen pixels from the top-left
// corner, and passes on where they read the atlas (in atlas pixels) and their colour.
const quadVertexShader = `#version 300 es
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

// quadFragmentShader paints the colour where the atlas (a grey coverage mask: glyphs, icons,
// rounded corners) covers.
const quadFragmentShader = `#version 300 es
precision mediump float;
in vec2 atlasCoordinate;
in vec4 color;
uniform sampler2D atlas;
out vec4 fragmentColor;

void main() {
    float coverage = texture(atlas, atlasCoordinate).r;
    fragmentColor = vec4(color.rgb, color.a * coverage);
}
`

// sceneVertexShader places a mesh in the world, raised by the vertical exaggeration, and passes on
// its world position, normal and colour (the vertex colour times the instance's tint).
const sceneVertexShader = `#version 300 es
layout(location = 0) in vec3 vertexPosition;
layout(location = 1) in vec3 vertexNormal;
layout(location = 2) in vec4 vertexColor;
uniform mat4 viewProjection;
uniform mat4 model;
uniform vec4 tint;
uniform float verticalScale;
out vec3 worldPosition;
out vec3 worldNormal;
out vec4 surfaceColor;

void main() {
    vec4 world = model * vec4(vertexPosition, 1.0);
    world.y *= verticalScale;
    worldPosition = world.xyz;
    // the inverse transpose keeps normals perpendicular under scales that differ per axis
    mat3 normalMatrix = transpose(inverse(mat3(model)));
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
const sceneFragmentShader = `#version 300 es
precision highp float;
in vec3 worldPosition;
in vec3 worldNormal;
in vec4 surfaceColor;
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
out vec4 fragmentColor;

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
    fragmentColor = vec4(pow(max(color.rgb, vec3(0.0)), vec3(1.0 / 2.2)), color.a);
}
`
