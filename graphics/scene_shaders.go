package graphics

// SceneVertexShader transforms meshes and GPU instances. Attributes: 0 position, 1 normal,
// 2 color, 3-6 instance matrix columns, 7 instance color. The vertical scale exaggerates
// heights (and corrects normals) without touching the geometry. Meshes with sway (plants modelled
// with height 1 and the base at the origin) bend with the wind, more at the top.
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
uniform float time;
uniform float sway;           // how much the wind bends this mesh
uniform vec2 windDirection;   // horizontal (x, z)
uniform float windStrength;
uniform int kind;                    // material kind; 2 = water
uniform int levelVariationEnabled;
uniform sampler2D levelVariation;    // r: in phase, g: in quadrature (world units)
uniform vec4 levelVariationArea;     // minimum x, minimum z, maximum x, maximum z
uniform float levelVariationAngle;
uniform float depthBias;             // share of the distance the surface is pulled towards the camera
uniform vec3 cameraPosition;
out vec3 worldPosition;
out vec3 worldNormal;
out vec4 surfaceColor;
// windBend moves a vertex along the wind, growing with the square of its height in the model, with
// gusts whose phase depends on where the model stands.
vec2 windBend(mat4 modelMatrix) {
    if (sway == 0.0 || windStrength == 0.0) return vec2(0.0);
    vec3 base = modelMatrix[3].xyz;
    float modelHeight = length(modelMatrix[1].xyz);
    float phase = base.x * 37.0 + base.z * 23.0;
    float gust = 0.65 + 0.35 * sin(time * (1.3 + windStrength) + phase) + 0.15 * sin(time * 3.7 + phase * 1.7);
    float height = max(vertexPosition.y, 0.0);
    return windDirection * (sway * windStrength * gust * height * height * modelHeight);
}

// levelOffset is how far a varying water surface (tides) is above its level at a point.
float levelOffset(vec2 position) {
    vec2 coordinates = (position - levelVariationArea.xy) / (levelVariationArea.zw - levelVariationArea.xy);
    vec2 harmonic = texture(levelVariation, coordinates).rg;
    return harmonic.r * cos(levelVariationAngle) + harmonic.g * sin(levelVariationAngle);
}

void main() {
    mat4 modelMatrix = model;
    vec4 color = vertexColor;
    if (instanced == 1) {
        modelMatrix = model * mat4(instanceColumnOne, instanceColumnTwo, instanceColumnThree, instanceColumnFour);
        color *= instanceColor;
    }
    vec4 world = modelMatrix * vec4(vertexPosition, 1.0);
    world.xz += windBend(modelMatrix);
    if (kind == 2 && levelVariationEnabled == 1) world.y += levelOffset(world.xz);
    world.y *= verticalScale;
    // the inverse transpose keeps normals perpendicular under scales that differ per axis
    vec3 normal = transpose(inverse(mat3(modelMatrix))) * vertexNormal;
    normal.y /= verticalScale;
    worldPosition = world.xyz;
    worldNormal = normal;
    surfaceColor = color;
    // thin surfaces on the terrain (rivers) are drawn a little closer, so its simpler far versions do not hide them
    if (depthBias > 0.0) world.xyz += (cameraPosition - world.xyz) * depthBias;
    gl_Position = projection * view * world;
}
` + "\x00"

// SceneFragmentShader shades by material kind: 0 lit, 1 unlit, 2 water, 3 waterfall, 4 glow; lit and
// glowing materials can carry a procedural surface pattern (see scene.Pattern).
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
uniform float brightness;          // light level for water and unlit materials (1 by day, low at night)
uniform sampler2DShadow shadowMap;
uniform mat4 lightViewProjection;
uniform int shadowsEnabled;
uniform float shadowTexelSize;     // in shadow map coordinates
uniform float shadowNormalOffset;  // in world units, pushes the lookup out of the surface
uniform sampler2D seabedMap;       // depth of the ground seen from above
uniform mat4 seabedViewProjection;
uniform vec2 seabedHeightRange;    // heights at depth 0 and depth 1 of the seabed map
uniform int seabedEnabled;
uniform float verticalScale;
uniform vec4 waterShallowColor;
uniform float waterColorDepth;
uniform float shoreFadeDepth;      // the sea fades out over this depth at the shore, with a line of foam
uniform int seaSurface;            // 1 for the environment's sea; rivers and lakes keep their own colour
uniform int pattern;               // procedural surface pattern (scene.Pattern), 0 = none
uniform float patternScale;        // world units per pattern cell
out vec4 fragmentColor;

float pseudoRandom(float seed) { return fract(sin(seed * 91.7) * 437.5); }

// shadowRegionFade is 1 inside the shadow region and fades to 0 near its border.
float shadowRegionFade(vec2 coordinates) {
    vec2 distanceToBorder = min(coordinates, 1.0 - coordinates);
    return smoothstep(0.0, 0.08, min(distanceToBorder.x, distanceToBorder.y));
}

// sunVisibility returns how much of the sun reaches the fragment (1 lit, 0 in shadow), filtered 3x3.
float sunVisibility(vec3 normal) {
    if (shadowsEnabled == 0) return 1.0;
    vec4 lightPosition = lightViewProjection * vec4(worldPosition + normal * shadowNormalOffset, 1.0);
    vec3 coordinates = lightPosition.xyz / lightPosition.w * 0.5 + 0.5;
    if (coordinates.z > 1.0 || any(lessThan(coordinates.xy, vec2(0.0))) || any(greaterThan(coordinates.xy, vec2(1.0)))) return 1.0;
    float visible = 0.0;
    for (int offsetX = -1; offsetX <= 1; offsetX++) {
        for (int offsetY = -1; offsetY <= 1; offsetY++) {
            vec2 samplePoint = coordinates.xy + vec2(float(offsetX), float(offsetY)) * shadowTexelSize;
            visible += texture(shadowMap, vec3(samplePoint, coordinates.z - 0.0005));
        }
    }
    return mix(1.0, visible / 9.0, shadowRegionFade(coordinates.xy));
}

float hashOf(vec2 cell) { return fract(sin(dot(cell, vec2(127.1, 311.7))) * 43758.5453); }
vec2 hashOf2(vec2 cell) { return fract(sin(vec2(dot(cell, vec2(127.1, 311.7)), dot(cell, vec2(269.5, 183.3)))) * 43758.5453); }

float valueNoise(vec2 point) {
    vec2 cell = floor(point), inside = fract(point);
    vec2 eased = inside * inside * (3.0 - 2.0 * inside);
    return mix(mix(hashOf(cell), hashOf(cell + vec2(1.0, 0.0)), eased.x),
               mix(hashOf(cell + vec2(0.0, 1.0)), hashOf(cell + vec2(1.0, 1.0)), eased.x), eased.y);
}

// surfaceCoordinates are pattern coordinates on the surface: x and z on flat ground; along the
// wall and up on walls; along the slope's contour and down the slope on roofs.
vec2 surfaceCoordinates(vec3 normal) {
    vec3 position = worldPosition / patternScale;
    if (abs(normal.y) > 0.92) return position.xz;
    vec2 across = normalize(vec2(-normal.z, normal.x) + vec2(1e-5, 0.0));
    return vec2(dot(position.xz, across), position.y);
}

float cobbles(vec2 point) {
    vec2 cell = floor(point);
    float nearest = 8.0, second = 8.0;
    vec2 nearestCell = cell;
    for (int column = -1; column <= 1; column++) {
        for (int row = -1; row <= 1; row++) {
            vec2 neighbour = cell + vec2(float(column), float(row));
            float gap = length(neighbour + 0.15 + 0.7 * hashOf2(neighbour) - point);
            if (gap < nearest) { second = nearest; nearest = gap; nearestCell = neighbour; }
            else if (gap < second) { second = gap; }
        }
    }
    float joint = smoothstep(0.03, 0.12, second - nearest);
    return mix(0.55, 0.82 + 0.3 * hashOf(nearestCell), joint) * (1.0 - 0.12 * nearest);
}

float masonry(vec2 point) {
    float course = floor(point.y);
    float along = point.x * 0.5 + 0.5 * mod(course, 2.0);
    vec2 block = vec2(floor(along), course);
    float joint = smoothstep(0.0, 0.06, fract(point.y)) * smoothstep(1.0, 0.94, fract(point.y))
                * smoothstep(0.0, 0.03, fract(along)) * smoothstep(1.0, 0.97, fract(along));
    return mix(0.62, 0.84 + 0.26 * hashOf(block) + 0.06 * valueNoise(point * 4.0), joint);
}

float shingles(vec2 point) {
    float row = floor(point.y);
    float along = point.x * 1.4 + 0.5 * mod(row, 2.0);
    float tone = 0.8 + 0.3 * hashOf(vec2(floor(along), row));
    float overlap = 0.72 + 0.28 * fract(point.y);
    float gap = smoothstep(0.0, 0.05, fract(along)) * smoothstep(1.0, 0.95, fract(along));
    return tone * overlap * mix(0.7, 1.0, gap);
}

float planks(vec2 point) {
    float board = floor(point.y);
    float gap = smoothstep(0.0, 0.05, fract(point.y)) * smoothstep(1.0, 0.95, fract(point.y));
    float grain = 0.94 + 0.08 * valueNoise(vec2(point.x * 0.6, point.y * 9.0));
    return mix(0.6, (0.82 + 0.26 * hashOf(vec2(board, floor(point.x * 0.25 + hashOf(vec2(board, 3.0)))))) * grain, gap);
}

// patternFactor multiplies a surface's colour by the material's pattern, fading it out where a
// cell covers less than about two pixels.
float patternFactor(vec3 normal) {
    if (pattern == 0 || patternScale <= 0.0) return 1.0;
    float cellsPerPixel = length(fwidth(worldPosition)) / patternScale;
    float strength = 1.0 - smoothstep(0.25, 0.8, cellsPerPixel);
    if (strength <= 0.0) return 1.0;
    vec2 point = surfaceCoordinates(normal);
    float factor = 1.0;
    if (pattern == 1) factor = cobbles(point);
    else if (pattern == 2) factor = masonry(point);
    else if (pattern == 3) factor = 0.9 + 0.1 * valueNoise(point * 0.8) + 0.06 * valueNoise(point * 7.0);
    else if (pattern == 4) factor = shingles(point);
    else if (pattern == 5) factor = planks(point);
    else if (pattern == 6) factor = 0.82 + 0.2 * valueNoise(point * 0.7) + 0.1 * valueNoise(point * 5.0);
    return mix(1.0, factor, strength);
}

vec3 shadeLit(vec3 color) {
    vec3 normal = normalize(worldNormal);
    if (!gl_FrontFacing) normal = -normal;
    float sunlight = max(dot(normal, normalize(sunDirection)), 0.0) * sunVisibility(normal);
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

// seabedHeight returns the height of the ground under the fragment (unscaled world units).
float seabedHeight() {
    vec4 projected = seabedViewProjection * vec4(worldPosition.x, 0.0, worldPosition.z, 1.0);
    vec2 coordinates = projected.xy / projected.w * 0.5 + 0.5;
    float depth = texture(seabedMap, coordinates).r;
    return mix(seabedHeightRange.x, seabedHeightRange.y, depth);
}

// waterBodyColor goes from the shallow to the deep color as the water under the fragment deepens.
// waterDepthHere is the depth of the sea under the fragment (the surface may vary with the tides).
float waterDepthHere() {
    return max(worldPosition.y / verticalScale - seabedHeight(), 0.0);
}

vec4 waterBodyColor(vec4 deepColor) {
    if (seabedEnabled == 0 || seaSurface == 0) return deepColor;
    float waterDepth = waterDepthHere();
    float relativeDepth = waterDepth / max(waterColorDepth, 1e-6);
    vec4 color = mix(waterShallowColor, deepColor, 1.0 - exp(-relativeDepth));
    // deep water hides the seabed completely, including where the geometry ends
    color.a = mix(color.a, 1.0, smoothstep(3.0, 6.0, relativeDepth));
    return color;
}

// softenShore fades the sea into the beach instead of cutting it where it meets the ground, with a
// thin moving line of foam just before the water ends.
vec4 softenShore(vec4 color) {
    float depth = waterDepthHere();
    float fade = max(shoreFadeDepth, 1e-6);
    float surf = 0.5 + 0.5 * sin(time * 1.4 + (worldPosition.x + worldPosition.z) * 3.0 / fade * 0.01);
    float foam = (1.0 - smoothstep(fade * 0.3, fade * 1.3, depth)) * smoothstep(0.0, fade * 0.3, depth) * (0.55 + 0.45 * surf);
    color.rgb = mix(color.rgb, vec3(0.95) * brightness, foam * 0.6);
    color.a *= smoothstep(0.0, fade, depth);
    return color;
}

vec4 shadeWater(vec4 color, vec3 towardsCamera) {
    color = waterBodyColor(color);
    color.rgb *= brightness;
    vec3 normal = waterNormal();
    float fresnel = pow(1.0 - max(dot(normal, towardsCamera), 0.0), 4.0) * 0.85 + 0.03;
    vec3 shaded = mix(color.rgb, mix(horizonColor, skyColor, 0.4), fresnel);
    vec3 reflected = reflect(-normalize(sunDirection), normal);
    shaded += sunColor * pow(max(dot(reflected, towardsCamera), 0.0), 180.0) * 1.6;
    vec4 result = vec4(shaded, mix(color.a, 1.0, fresnel));
    if (seaSurface == 1 && seabedEnabled == 1) result = softenShore(result);
    return result;
}


float waterfallOpacity(float baseOpacity) {
    float lane = floor((worldPosition.x + worldPosition.z) * 40.0);
    float streak = fract(worldPosition.y * 3.0 + time * (0.9 + pseudoRandom(lane) * 0.6) + pseudoRandom(lane));
    return baseOpacity * (0.45 + 0.55 * smoothstep(0.0, 0.5, streak));
}

void main() {
    vec4 color = surfaceColor * baseColor;
    vec3 towardsCamera = normalize(cameraPosition - worldPosition);
    if (kind == 0 || kind == 4) {
        vec3 normal = normalize(worldNormal);
        color.rgb *= patternFactor(gl_FrontFacing ? normal : -normal);
    }
    if (kind == 0) {
        color.rgb = shadeLit(color.rgb);
    } else if (kind == 4) {
        // glows: lit by day, its own colour in the dark
        color.rgb = max(shadeLit(color.rgb), color.rgb * (1.0 - brightness) * 1.4);
    } else if (kind == 2) {
        color = shadeWater(color, towardsCamera);
    } else if (kind == 3) {
        color.a = waterfallOpacity(color.a);
        color.rgb *= brightness;
    } else {
        color.rgb *= brightness;
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

// SkyFragmentShader paints the sky for any time of day: a day gradient with the sun, a twilight
// glow, a night with procedural stars and a moon with phases, and a cloud layer over both. A day
// or night image (equirectangular or cube) can replace the procedural sky of that time.
const SkyFragmentShader = `
#version 410
in vec2 screenPosition;
uniform mat4 inverseViewProjection;
uniform vec3 cameraPosition;
uniform float time;
uniform vec3 sunDirection;
uniform vec3 sunColor;
uniform float sunVisible;
uniform vec3 skyColor;          // zenith, already blended for the time of day
uniform vec3 horizonColor;
uniform vec3 twilightColor;
uniform float dayAmount;
uniform float twilightAmount;
uniform mat4 starRotation;      // world direction -> celestial direction
uniform float starDensity;
uniform float starBrightness;
uniform float starTwinkle;
uniform float starDaytimeVisibility;
uniform int moonEnabled;
uniform vec3 moonDirection;
uniform float moonRadius;       // angular radius in radians
uniform vec3 moonColor;
uniform float moonPhase;
uniform float cloudCoverage;
uniform vec3 cloudColor;
uniform float cloudSpeed;
uniform float cloudScale;
uniform vec2 cloudOffset;       // drift moved by a simulation
uniform vec2 windDirection;
uniform int dayImageKind;       // 0 none, 1 equirectangular, 2 cube
uniform int nightImageKind;
uniform sampler2D dayPanorama;
uniform sampler2D nightPanorama;
uniform samplerCube dayCube;
uniform samplerCube nightCube;
out vec4 fragmentColor;

const float PI = 3.14159265;
const float STAR_CELLS = 220.0;  // star cells per cube face side

float hash(vec3 seed) { return fract(sin(dot(seed, vec3(12.9898, 78.233, 37.719))) * 43758.5453); }
float hash2(vec2 seed) { return fract(sin(dot(seed, vec2(127.1, 311.7))) * 43758.5453); }

float valueNoise(vec2 position) {
    vec2 cell = floor(position);
    vec2 inside = fract(position);
    vec2 blend = inside * inside * (3.0 - 2.0 * inside);
    float bottom = mix(hash2(cell), hash2(cell + vec2(1.0, 0.0)), blend.x);
    float top = mix(hash2(cell + vec2(0.0, 1.0)), hash2(cell + vec2(1.0, 1.0)), blend.x);
    return mix(bottom, top, blend.y);
}

float fractalNoise(vec2 position) {
    float total = 0.0;
    float amplitude = 0.5;
    for (int octave = 0; octave < 5; octave++) {
        total += valueNoise(position) * amplitude;
        position = position * 2.03 + vec2(17.0, 9.0);
        amplitude *= 0.5;
    }
    return total;
}

vec3 viewDirection() {
    vec4 farPoint = inverseViewProjection * vec4(screenPosition, 1.0, 1.0);
    return normalize(farPoint.xyz / farPoint.w - cameraPosition);
}

// samplePanorama reads an equirectangular image. The horizontal coordinate jumps where the
// panorama wraps around, so its screen derivative is taken from a copy shifted by half a turn.
vec3 samplePanorama(sampler2D panorama, vec3 direction) {
    vec2 coordinates = vec2(atan(direction.x, -direction.z) / (2.0 * PI) + 0.5, acos(clamp(direction.y, -1.0, 1.0)) / PI);
    float shifted = fract(coordinates.x + 0.5);
    vec2 alongX = vec2(dFdx(coordinates.x), dFdx(shifted));
    vec2 alongY = vec2(dFdy(coordinates.x), dFdy(shifted));
    float gradientX = abs(alongX.x) < abs(alongX.y) ? alongX.x : alongX.y;
    float gradientY = abs(alongY.x) < abs(alongY.y) ? alongY.x : alongY.y;
    return textureGrad(panorama, coordinates, vec2(gradientX, dFdx(coordinates.y)), vec2(gradientY, dFdy(coordinates.y))).rgb;
}

vec3 dayGradient(vec3 direction) {
    float elevation = max(direction.y, 0.0);
    vec3 color = mix(horizonColor, skyColor, pow(elevation, 0.55));
    // the twilight glow is strongest on the side of the sun
    float sunSide = max(dot(normalize(direction.xz + 1e-5), normalize(sunDirection.xz + 1e-5)), 0.0);
    color += twilightColor * twilightAmount * pow(1.0 - elevation, 6.0) * (0.25 + 0.75 * sunSide) * 0.6;
    return color;
}

// cubeCell finds the star cell of a celestial direction: the cube face and the cell on it.
vec3 cubeCell(vec3 celestial, out vec2 insideCell) {
    vec3 magnitude = abs(celestial);
    vec2 face;
    float faceNumber;
    if (magnitude.x >= magnitude.y && magnitude.x >= magnitude.z) { face = celestial.yz / magnitude.x; faceNumber = sign(celestial.x); }
    else if (magnitude.y >= magnitude.z) { face = celestial.xz / magnitude.y; faceNumber = 2.0 * sign(celestial.y); }
    else { face = celestial.xy / magnitude.z; faceNumber = 3.0 * sign(celestial.z); }
    vec2 grid = (face * 0.5 + 0.5) * STAR_CELLS;
    insideCell = fract(grid);
    return vec3(floor(grid), faceNumber);
}

vec3 starField(vec3 direction) {
    vec3 celestial = mat3(starRotation) * direction;
    // derivatives first: they are undefined inside the branches below
    float cellInPixels = (2.0 / STAR_CELLS) / max(length(fwidth(celestial)), 1e-6);
    vec2 insideCell;
    vec3 cell = cubeCell(celestial, insideCell);
    if (hash(cell) > starDensity) return vec3(0.0);
    vec2 center = vec2(hash(cell + 1.3), hash(cell + 2.7)) * 0.6 + 0.2;
    float distanceInPixels = length(insideCell - center) * cellInPixels;
    float magnitude = pow(hash(cell + 4.1), 10.0) * 3.0 + 0.12;
    float twinkle = 1.0 + starTwinkle * sin(time * (2.0 + hash(cell + 5.9) * 3.0) + hash(cell) * 40.0);
    float temperature = hash(cell + 7.3);
    vec3 tint = mix(vec3(1.0, 0.78, 0.6), vec3(0.72, 0.82, 1.0), temperature);
    float point = exp(-distanceInPixels * distanceInPixels * 1.6);
    float extinction = smoothstep(-0.02, 0.18, direction.y);
    return tint * point * magnitude * twinkle * starBrightness * extinction;
}

// moonDisc returns the moon color (rgb) and coverage (a) with its phase.
vec4 moonDisc(vec3 direction) {
    if (moonEnabled == 0) return vec4(0.0);
    float angle = acos(clamp(dot(direction, moonDirection), -1.0, 1.0));
    float edge = moonRadius * 0.04;
    float coverage = 1.0 - smoothstep(moonRadius - edge, moonRadius + edge, angle);
    vec3 right = normalize(cross(moonDirection, vec3(0.0, 1.0, 0.0)) + vec3(1e-5, 0.0, 0.0));
    vec3 up = cross(right, moonDirection);
    vec2 onDisc = vec2(dot(direction, right), dot(direction, up)) / max(moonRadius, 1e-5);
    vec3 surfaceNormal = vec3(onDisc, sqrt(max(1.0 - dot(onDisc, onDisc), 0.0)));
    float phaseAngle = 2.0 * PI * moonPhase;
    vec3 towardsLight = vec3(sin(phaseAngle), 0.0, -cos(phaseAngle));
    float lit = smoothstep(-0.03, 0.03, dot(surfaceNormal, towardsLight));
    float maria = 0.82 + 0.18 * valueNoise(onDisc * 3.0 + 5.0);
    vec3 color = moonColor * (lit * maria + 0.025);
    float illuminated = 0.5 - 0.5 * cos(phaseAngle);
    float halo = pow(max(dot(direction, moonDirection), 0.0), 2000.0) * 0.25 * illuminated * (1.0 - coverage);
    return vec4(color * coverage + moonColor * halo, max(coverage, halo));
}

vec4 cloudLayer(vec3 direction) {
    if (cloudCoverage <= 0.0 || direction.y <= 0.0) return vec4(0.0);
    vec2 drift = cloudOffset + windDirection * time * cloudSpeed;
    vec2 position = direction.xz / (direction.y + 0.12) * (1.6 / max(cloudScale, 1e-3)) + drift;
    // the noise stays mostly between 0.3 and 0.7, so the coverage moves the threshold inside that range
    float threshold = mix(0.72, 0.3, cloudCoverage);
    float density = smoothstep(threshold, threshold + 0.14, fractalNoise(position));
    density *= smoothstep(0.0, 0.18, direction.y);
    float towardsSun = max(dot(direction, normalize(sunDirection)), 0.0);
    vec3 litColor = cloudColor * (sunColor * (0.55 + 0.45 * pow(towardsSun, 8.0)) + horizonColor * 0.5);
    vec3 nightColor = horizonColor * 1.6 + skyColor;
    vec3 color = mix(nightColor, litColor, dayAmount) + twilightColor * twilightAmount * 0.35;
    return vec4(color, density * 0.92);
}

vec3 daySky(vec3 direction) {
    if (dayImageKind == 1) return samplePanorama(dayPanorama, direction);
    if (dayImageKind == 2) return texture(dayCube, direction).rgb;
    vec3 color = dayGradient(direction);
    float towardsSun = max(dot(direction, normalize(sunDirection)), 0.0);
    color += sunColor * sunVisible * (pow(towardsSun, 900.0) * 3.0 + pow(towardsSun, 12.0) * 0.25);
    return color;
}

vec3 nightSky(vec3 direction) {
    vec3 celestial = mat3(starRotation) * direction;
    if (nightImageKind == 1) return samplePanorama(nightPanorama, celestial);
    if (nightImageKind == 2) return texture(nightCube, celestial).rgb;
    return dayGradient(direction);
}

void main() {
    vec3 direction = viewDirection();
    float nightAmount = 1.0 - dayAmount;
    vec3 color = mix(nightSky(direction), daySky(direction), dayAmount);
    if (nightImageKind == 0) {
        // stars appear only once the sky is dark enough
        float darkness = pow(nightAmount, 4.0);
        color += starField(direction) * clamp(darkness + starDaytimeVisibility * (1.0 - darkness), 0.0, 1.0);
    }
    vec4 moon = moonDisc(direction);
    color = mix(color, moon.rgb + color * 0.2, moon.a * smoothstep(-0.02, 0.02, direction.y) * mix(0.35, 1.0, nightAmount));
    vec4 clouds = cloudLayer(direction);
    color = mix(color, clouds.rgb, clouds.a);
    if (direction.y < 0.0) color = mix(horizonColor, horizonColor * 0.8, min(-direction.y * 4.0, 1.0));
    fragmentColor = vec4(pow(max(color, vec3(0.0)), vec3(1.0 / 2.2)), 1.0);
}
` + "\x00"

// ConstellationVertexShader places constellation stars and lines on the sky sphere: directions
// turn with the night sky and are drawn at the far plane around the camera.
const ConstellationVertexShader = `
#version 410
layout(location = 0) in vec3 celestialDirection;
uniform mat4 viewProjection;
uniform vec3 cameraPosition;
uniform mat4 starRotation;
uniform float pointSize;
void main() {
    vec3 world = transpose(mat3(starRotation)) * normalize(celestialDirection);
    vec4 position = viewProjection * vec4(cameraPosition + world * 1000.0, 1.0);
    position.z = position.w * 0.99999;
    gl_Position = position;
    gl_PointSize = pointSize;
}
` + "\x00"

// ConstellationFragmentShader colors constellation lines and round stars, fading with daylight.
const ConstellationFragmentShader = `
#version 410
uniform vec4 lineColor;
uniform float visibility;
uniform int drawingPoints;
out vec4 fragmentColor;
void main() {
    float alpha = lineColor.a * visibility;
    if (drawingPoints == 1) {
        float distanceFromCenter = length(gl_PointCoord - 0.5) * 2.0;
        alpha *= 1.0 - smoothstep(0.4, 1.0, distanceFromCenter);
    }
    fragmentColor = vec4(pow(lineColor.rgb, vec3(1.0 / 2.2)), alpha);
}
` + "\x00"

// DepthOnlyFragmentShader writes only depth; it pairs with SceneVertexShader for the shadow and
// seabed passes.
const DepthOnlyFragmentShader = `
#version 410
void main() {
}
` + "\x00"
