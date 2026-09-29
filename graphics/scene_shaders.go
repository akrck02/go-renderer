package graphics

// SceneVertexShader transforms meshes and GPU instances. Attributes: 0 position, 1 normal,
// 2 color, 3-6 instance matrix columns, 7 instance color. The vertical scale exaggerates
// heights (and corrects normals) without touching the geometry.
const SceneVertexShader = `
#version 410
layout(location = 0) in vec3 vertexPosition;
layout(location = 1) in vec3 vertexNormal;
layout(location = 2) in vec4 vertexColor;
layout(location = 3) in vec4 instanceColumnOne;
layout(location = 4) in vec4 instanceColumnTwo;
layout(location = 5) in vec4 instanceColumnThree;
layout(location = 6) in vec4 instanceColumnFour;
layout(location = 7) in vec4 instanceColor;
uniform mat4 projection;
uniform mat4 view;
uniform mat4 model;
uniform int instanced;
uniform float verticalScale;
out vec3 worldPosition;
out vec3 worldNormal;
out vec4 surfaceColor;
void main() {
    mat4 modelMatrix = model;
    vec4 color = vertexColor;
    if (instanced == 1) {
        modelMatrix = model * mat4(instanceColumnOne, instanceColumnTwo, instanceColumnThree, instanceColumnFour);
        color *= instanceColor;
    }
    vec4 world = modelMatrix * vec4(vertexPosition, 1.0);
    world.y *= verticalScale;
    vec3 normal = mat3(modelMatrix) * vertexNormal;
    normal.y /= verticalScale;
    worldPosition = world.xyz;
    worldNormal = normal;
    surfaceColor = color;
    gl_Position = projection * view * world;
}
` + "\x00"

// SceneFragmentShader shades by material kind: 0 lit, 1 unlit, 2 water, 3 waterfall.
const SceneFragmentShader = `
#version 410
in vec3 worldPosition;
in vec3 worldNormal;
in vec4 surfaceColor;
uniform vec4 baseColor;
uniform int kind;
uniform vec3 sunDirection;
uniform vec3 sunColor;
uniform vec3 skyColor;
uniform vec3 horizonColor;
uniform vec3 groundColor;
uniform float ambient;
uniform vec3 fogColor;
uniform vec2 fogRange;
uniform vec3 cameraPosition;
uniform float time;
uniform float waveLength;
out vec4 fragmentColor;

float pseudoRandom(float seed) { return fract(sin(seed * 91.7) * 437.5); }

vec3 shadeLit(vec3 color) {
    vec3 normal = normalize(worldNormal);
    if (!gl_FrontFacing) normal = -normal;
    float sunlight = max(dot(normal, normalize(sunDirection)), 0.0);
    vec3 skylight = mix(groundColor, skyColor, normal.y * 0.5 + 0.5) * ambient;
    return color * (skylight + sunColor * sunlight);
}

vec3 waterNormal() {
    vec2 position = worldPosition.xz;
    vec2 slope = vec2(0.0);
    vec2 directions[4] = vec2[4](normalize(vec2(1.0, 0.4)), normalize(vec2(-0.6, 1.0)), normalize(vec2(0.3, -1.0)), normalize(vec2(-1.0, -0.2)));
    float frequencies[4] = float[4](1.0, 1.9, 3.4, 5.9);
    float waveNumber = 6.2831853 / max(waveLength, 1e-6);
    for (int wave = 0; wave < 4; wave++) {
        float frequency = waveNumber * frequencies[wave];
        slope += directions[wave] * cos(dot(position, directions[wave]) * frequency + time * (1.2 + float(wave) * 0.5)) * (0.5 / frequencies[wave]);
    }
    // waves fade with distance so that far water does not alias into regular stripes
    float fade = 1.0 / (1.0 + distance(cameraPosition, worldPosition) / (waveLength * 60.0));
    return normalize(vec3(-slope.x * 0.12 * fade, 1.0, -slope.y * 0.12 * fade));
}

vec4 shadeWater(vec4 color, vec3 towardsCamera) {
    vec3 normal = waterNormal();
    float fresnel = pow(1.0 - max(dot(normal, towardsCamera), 0.0), 4.0) * 0.85 + 0.03;
    vec3 shaded = mix(color.rgb, mix(horizonColor, skyColor, 0.4), fresnel);
    vec3 reflected = reflect(-normalize(sunDirection), normal);
    shaded += sunColor * pow(max(dot(reflected, towardsCamera), 0.0), 180.0) * 1.6;
    return vec4(shaded, mix(color.a, 1.0, fresnel));
}

float waterfallOpacity(float baseOpacity) {
    float lane = floor((worldPosition.x + worldPosition.z) * 40.0);
    float streak = fract(worldPosition.y * 3.0 + time * (0.9 + pseudoRandom(lane) * 0.6) + pseudoRandom(lane));
    return baseOpacity * (0.45 + 0.55 * smoothstep(0.0, 0.5, streak));
}

void main() {
    vec4 color = surfaceColor * baseColor;
    vec3 towardsCamera = normalize(cameraPosition - worldPosition);
    if (kind == 0) {
        color.rgb = shadeLit(color.rgb);
    } else if (kind == 2) {
        color = shadeWater(color, towardsCamera);
    } else if (kind == 3) {
        color.a = waterfallOpacity(color.a);
    }
    if (fogRange.y > fogRange.x) {
        float fogAmount = smoothstep(fogRange.x, fogRange.y, distance(cameraPosition, worldPosition));
        color.rgb = mix(color.rgb, fogColor, fogAmount);
        color.a = mix(color.a, 1.0, fogAmount * step(1.5, float(kind)));
    }
    fragmentColor = vec4(pow(max(color.rgb, vec3(0.0)), vec3(1.0 / 2.2)), color.a);
}
` + "\x00"

// SkyVertexShader draws a full-screen triangle at the far plane.
const SkyVertexShader = `
#version 410
out vec2 screenPosition;
void main() {
    vec2 corner = vec2(float((gl_VertexID << 1) & 2), float(gl_VertexID & 2)) * 2.0 - 1.0;
    screenPosition = corner;
    gl_Position = vec4(corner, 1.0, 1.0);
}
` + "\x00"

// SkyFragmentShader paints a zenith-horizon gradient with a sun disc and halo.
const SkyFragmentShader = `
#version 410
in vec2 screenPosition;
uniform mat4 inverseViewProjection;
uniform vec3 cameraPosition;
uniform vec3 sunDirection;
uniform vec3 sunColor;
uniform vec3 skyColor;
uniform vec3 horizonColor;
out vec4 fragmentColor;
void main() {
    vec4 farPoint = inverseViewProjection * vec4(screenPosition, 1.0, 1.0);
    vec3 direction = normalize(farPoint.xyz / farPoint.w - cameraPosition);
    float elevation = max(direction.y, 0.0);
    vec3 color = mix(horizonColor, skyColor, pow(elevation, 0.55));
    float towardsSun = max(dot(direction, normalize(sunDirection)), 0.0);
    color += sunColor * (pow(towardsSun, 900.0) * 3.0 + pow(towardsSun, 12.0) * 0.25);
    if (direction.y < 0.0) color = mix(horizonColor, horizonColor * 0.8, min(-direction.y * 4.0, 1.0));
    fragmentColor = vec4(pow(color, vec3(1.0 / 2.2)), 1.0);
}
` + "\x00"
