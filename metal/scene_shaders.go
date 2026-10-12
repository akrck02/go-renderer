package metal

// sceneShaderSource is the 3D scene's shaders in Metal Shading Language: the translation of the
// OpenGL ones in graphics/scene_shaders.go (scene, sky, constellations) and
// graphics/overlay_shaders.go (the 2D overlay), compiled at run time so building needs no Xcode.
//
// The maths, constants and order of operations are the GLSL's, so a frame drawn through Metal
// matches one drawn through OpenGL within rounding. What differs is only what the two languages
// do differently:
//   - uniforms come in structs (FrameUniforms, DrawUniforms, SkyUniforms, ConstellationUniforms)
//     that the Go side mirrors byte for byte; static_asserts fail the compilation if a size drifts;
//   - vertices are read from buffers with the vertex and instance indices, without vertex
//     descriptors;
//   - the projections the renderer passes are already in Metal's clip space (depth from 0 to 1),
//     and Metal textures have row 0 at the top, so shadow and seabed lookups flip v and use the
//     depth as it is; the sky keeps OpenGL's inverse view-projection, as its maths is written for it;
//   - samplers are declared in the shader, so the renderer binds no sampler states;
//   - GLSL's mod, atan(y, x), inverse and textureGrad have no exact MSL spelling: see glslMod,
//     atan2, inverseTranspose and gradient2d below;
//   - the constellation shader's drawingPoints switch becomes two fragment functions, one for
//     lines and one for points, because a point coordinate only exists when drawing points.
//
// Compile it with safe maths (MTLMathModeSafe): fast maths changes sin for the large arguments
// the noise hashes use, and the patterns would no longer match OpenGL's.
const sceneShaderSource = `
#include <metal_stdlib>
using namespace metal;

// ---------------------------------------------------------------------------------------------
// Data shared with the Go side. Sizes are checked so a change on one side cannot go unnoticed.
// ---------------------------------------------------------------------------------------------

struct Vertex {
    packed_float3 position;
    packed_float3 normal;
    packed_float4 color;
};
static_assert(sizeof(Vertex) == 40, "Vertex must be 40 bytes, as the Go side writes it");

struct Instance {
    float4x4 transform;
    float4 color;
};
static_assert(sizeof(Instance) == 80, "Instance must be 80 bytes, as the Go side writes it");

struct FrameUniforms {
    float4x4 projection;            // Metal clip space
    float4x4 view;
    float4x4 lightViewProjection;   // Metal clip space
    float4x4 seabedViewProjection;  // Metal clip space
    float4 sunDirection;            // xyz used
    float4 sunColor;
    float4 skyColor;
    float4 horizonColor;
    float4 groundColor;
    float4 fogColor;
    float4 cameraPosition;
    float4 foliageTint;             // colour and amount (alpha) of the season's foliage
    float4 snowColor;
    float4 waterShallowColor;
    float4 levelVariationArea;      // minimum x, minimum z, maximum x, maximum z
    float2 windDirection;           // horizontal (x, z)
    float2 fogRange;
    float2 seabedHeightRange;       // heights at depth 0 and depth 1 of the seabed map
    float2 unusedPair;
    float verticalScale;
    float windStrength;
    float ambient;
    float brightness;               // light level for water and unlit materials (1 by day, low at night)
    float time;
    float waveLength;
    float shadowTexelSize;          // in shadow map coordinates
    float shadowNormalOffset;       // in world units, pushes the lookup out of the surface
    float waterColorDepth;
    float shoreFadeDepth;           // the sea fades out over this depth at the shore, with a line of foam
    float snowLevel;                // world units, before vertical scaling
    float snowBlend;
    float levelVariationAngle;
    int shadowsEnabled;
    int seabedEnabled;
    int snowEnabled;
};
static_assert(sizeof(FrameUniforms) == 528, "FrameUniforms must be 528 bytes, as the Go side writes it");

struct DrawUniforms {
    float4x4 model;
    float4 baseColor;
    float sway;                     // how much the wind bends this mesh
    float depthBias;                // share of the distance the surface is pulled towards the camera
    float patternScale;             // world units per pattern cell
    int instanced;
    int kind;                       // material kind: 0 lit, 1 unlit, 2 water, 3 waterfall, 4 glow
    int pattern;                    // procedural surface pattern (scene.Pattern), 0 = none
    int foliage;                    // 1: the material takes the season's foliage tint
    int levelVariationEnabled;
    int seaSurface;                 // 1 for the environment's sea; rivers and lakes keep their own colour
    int unused0;
    int unused1;
    int unused2;
};
static_assert(sizeof(DrawUniforms) == 128, "DrawUniforms must be 128 bytes, as the Go side writes it");

struct SkyUniforms {
    float4x4 inverseViewProjection; // OpenGL convention
    float4x4 starRotation;          // world direction -> celestial direction
    float4 cameraPosition;
    float4 sunDirection;
    float4 sunColor;
    float4 skyColor;                // zenith, already blended for the time of day
    float4 horizonColor;
    float4 twilightColor;
    float4 moonDirection;
    float4 moonColor;
    float4 cloudColor;
    float2 cloudOffset;             // drift moved by a simulation
    float2 windDirection;
    float time;
    float sunVisible;
    float dayAmount;
    float twilightAmount;
    float starDensity;
    float starBrightness;
    float starTwinkle;
    float starDaytimeVisibility;
    float moonRadius;               // angular radius in radians
    float moonPhase;
    float cloudCoverage;
    float cloudSpeed;
    float cloudScale;
    int moonEnabled;
    int dayImageKind;               // 0 none, 1 equirectangular, 2 cube
    int nightImageKind;
};
static_assert(sizeof(SkyUniforms) == 352, "SkyUniforms must be 352 bytes, as the Go side writes it");

struct ConstellationUniforms {
    float4x4 viewProjection;        // Metal clip space
    float4x4 starRotation;
    float4 cameraPosition;
    float4 lineColor;
    float visibility;
    float pointSize;
    int unused0;
    int unused1;
};
static_assert(sizeof(ConstellationUniforms) == 176, "ConstellationUniforms must be 176 bytes, as the Go side writes it");

struct OverlaySizes {
    float2 screen;
    float2 atlas;
};
static_assert(sizeof(OverlaySizes) == 16, "OverlaySizes must be 16 bytes, as the Go side writes it");

struct OverlayInstance {
    packed_float4 screen;
    packed_float4 atlas;
    uchar4 color;
};
static_assert(sizeof(OverlayInstance) == 36, "OverlayInstance must be 36 bytes, as the Go side writes it");

// ---------------------------------------------------------------------------------------------
// Samplers. OpenGL keeps them in the texture objects; here they live in the shader.
// ---------------------------------------------------------------------------------------------

// outside the shadow map everything is lit: the white border is the farthest depth
constexpr sampler shadowSampler(coord::normalized, filter::linear, address::clamp_to_border,
                                border_color::opaque_white, compare_func::less_equal);
constexpr sampler seabedSampler(filter::linear, address::clamp_to_edge);
constexpr sampler levelVariationSampler(filter::linear, address::clamp_to_edge);
// panoramas wrap around horizontally but not over the poles
constexpr sampler panoramaSampler(filter::linear, mip_filter::linear, s_address::repeat, t_address::clamp_to_edge);
constexpr sampler cubeSampler(filter::linear, address::clamp_to_edge);
constexpr sampler atlasSampler(filter::linear, address::clamp_to_edge);

// ---------------------------------------------------------------------------------------------
// What GLSL has built in and MSL spells differently.
// ---------------------------------------------------------------------------------------------

// glslMod is GLSL's mod, x - y * floor(x / y): MSL's fmod truncates instead, which differs for
// negative values (and courses and rows below zero must alternate the same way).
float glslMod(float x, float y) {
    return x - y * floor(x / y);
}

// smoothstepAnyOrder is smoothstep written out, because the patterns call it with the edges
// reversed (from 1 down to 0.94) to fade out at the end of a cell. Both languages leave that case
// undefined; OpenGL drivers compute this formula, so writing it here keeps Metal the same.
float smoothstepAnyOrder(float edgeStart, float edgeEnd, float value) {
    float amount = clamp((value - edgeStart) / (edgeEnd - edgeStart), 0.0, 1.0);
    return amount * amount * (3.0 - 2.0 * amount);
}

// upperLeft is GLSL's mat3(matrix): the rotation and scale part of a 4×4 matrix.
float3x3 upperLeft(float4x4 transform) {
    return float3x3(transform[0].xyz, transform[1].xyz, transform[2].xyz);
}

// inverseTranspose is GLSL's transpose(inverse(matrix)), which MSL lacks: the cofactors divided
// by the determinant. The columns of the inverse transpose are the cross products of the other
// two columns.
float3x3 inverseTranspose(float3x3 transform) {
    float3 first = cross(transform[1], transform[2]);
    float3 second = cross(transform[2], transform[0]);
    float3 third = cross(transform[0], transform[1]);
    float determinant = dot(transform[0], first);
    return float3x3(first / determinant, second / determinant, third / determinant);
}

// ---------------------------------------------------------------------------------------------
// Noise shared by the scene and the sky (the GLSL of both has the same hash and value noise).
// ---------------------------------------------------------------------------------------------

float hashOf(float2 cell) { return fract(sin(dot(cell, float2(127.1, 311.7))) * 43758.5453); }
float2 hashPairOf(float2 cell) { return fract(sin(float2(dot(cell, float2(127.1, 311.7)), dot(cell, float2(269.5, 183.3)))) * 43758.5453); }
float hashOf(float3 seed) { return fract(sin(dot(seed, float3(12.9898, 78.233, 37.719))) * 43758.5453); }

float valueNoise(float2 point) {
    float2 cell = floor(point), inside = fract(point);
    float2 eased = inside * inside * (3.0 - 2.0 * inside);
    return mix(mix(hashOf(cell), hashOf(cell + float2(1.0, 0.0)), eased.x),
               mix(hashOf(cell + float2(0.0, 1.0)), hashOf(cell + float2(1.0, 1.0)), eased.x), eased.y);
}

// ---------------------------------------------------------------------------------------------
// The scene: meshes and GPU instances. The vertical scale exaggerates heights (and corrects
// normals) without touching the geometry. Meshes with sway (plants modelled with height 1 and the
// base at the origin) bend with the wind, more at the top.
// ---------------------------------------------------------------------------------------------

struct SceneVaryings {
    float4 position [[position]];
    float3 worldPosition;
    float3 worldNormal;
    float4 surfaceColor;
};

// windBend moves a vertex along the wind, growing with the square of its height in the model, with
// gusts whose phase depends on where the model stands.
float2 windBend(float4x4 modelMatrix, float3 vertexPosition, constant FrameUniforms &frame, constant DrawUniforms &draw) {
    if (draw.sway == 0.0 || frame.windStrength == 0.0) return float2(0.0);
    float3 base = modelMatrix[3].xyz;
    float modelHeight = length(modelMatrix[1].xyz);
    float phase = base.x * 37.0 + base.z * 23.0;
    float gust = 0.65 + 0.35 * sin(frame.time * (1.3 + frame.windStrength) + phase) + 0.15 * sin(frame.time * 3.7 + phase * 1.7);
    float height = max(vertexPosition.y, 0.0);
    return frame.windDirection * (draw.sway * frame.windStrength * gust * height * height * modelHeight);
}

// levelOffset is how far a varying water surface (tides) is above its level at a point. The
// texture holds the harmonic in phase (r) and in quadrature (g), in world units. A vertex function
// has no derivatives to pick a mipmap level, so the base level is named, as GLSL does implicitly.
float levelOffset(float2 position, constant FrameUniforms &frame, texture2d<float> levelVariation) {
    float2 coordinates = (position - frame.levelVariationArea.xy) / (frame.levelVariationArea.zw - frame.levelVariationArea.xy);
    float2 harmonic = levelVariation.sample(levelVariationSampler, coordinates, level(0.0)).rg;
    return harmonic.r * cos(frame.levelVariationAngle) + harmonic.g * sin(frame.levelVariationAngle);
}

// sceneVertex also serves the depth-only passes (shadow map, seabed map), which have no fragment
// function, so it must not depend on anything only the fragment stage has.
vertex SceneVaryings sceneVertex(uint vertexIndex [[vertex_id]],
                                 uint instanceIndex [[instance_id]],
                                 const device Vertex *vertices [[buffer(0)]],
                                 const device Instance *instances [[buffer(1)]],
                                 constant FrameUniforms &frame [[buffer(2)]],
                                 constant DrawUniforms &draw [[buffer(3)]],
                                 texture2d<float> levelVariation [[texture(0)]]) {
    Vertex input = vertices[vertexIndex];
    float3 vertexPosition = float3(input.position);
    float3 vertexNormal = float3(input.normal);
    float4x4 modelMatrix = draw.model;
    float4 color = float4(input.color);
    if (draw.instanced == 1) {
        Instance instance = instances[instanceIndex];
        modelMatrix = draw.model * instance.transform;
        color *= instance.color;
    }
    float4 world = modelMatrix * float4(vertexPosition, 1.0);
    world.xz += windBend(modelMatrix, vertexPosition, frame, draw);
    if (draw.kind == 2 && draw.levelVariationEnabled == 1) world.y += levelOffset(world.xz, frame, levelVariation);
    world.y *= frame.verticalScale;
    // the inverse transpose keeps normals perpendicular under scales that differ per axis
    float3 normal = inverseTranspose(upperLeft(modelMatrix)) * vertexNormal;
    normal.y /= frame.verticalScale;
    SceneVaryings output;
    output.worldPosition = world.xyz;
    output.worldNormal = normal;
    output.surfaceColor = color;
    // thin surfaces on the terrain (rivers) are drawn a little closer, so its simpler far versions do not hide them
    if (draw.depthBias > 0.0) world.xyz += (frame.cameraPosition.xyz - world.xyz) * draw.depthBias;
    output.position = frame.projection * frame.view * world;
    return output;
}

float pseudoRandom(float seed) { return fract(sin(seed * 91.7) * 437.5); }

// shadowRegionFade is 1 inside the shadow region and fades to 0 near its border.
float shadowRegionFade(float2 coordinates) {
    float2 distanceToBorder = min(coordinates, 1.0 - coordinates);
    return smoothstep(0.0, 0.08, min(distanceToBorder.x, distanceToBorder.y));
}

// sunVisibility returns how much of the sun reaches the fragment (1 lit, 0 in shadow), filtered 3x3.
// The light's projection is already in Metal's clip space, so the depth needs no remapping; v is
// flipped because the shadow map's row 0 is its top.
float sunVisibility(float3 normal, float3 worldPosition, constant FrameUniforms &frame, depth2d<float> shadowMap) {
    if (frame.shadowsEnabled == 0) return 1.0;
    float4 lightPosition = frame.lightViewProjection * float4(worldPosition + normal * frame.shadowNormalOffset, 1.0);
    float3 deviceCoordinates = lightPosition.xyz / lightPosition.w;
    float3 coordinates = float3(deviceCoordinates.x * 0.5 + 0.5, 0.5 - deviceCoordinates.y * 0.5, deviceCoordinates.z);
    if (coordinates.z > 1.0 || any(coordinates.xy < float2(0.0)) || any(coordinates.xy > float2(1.0))) return 1.0;
    float visible = 0.0;
    for (int offsetX = -1; offsetX <= 1; offsetX++) {
        for (int offsetY = -1; offsetY <= 1; offsetY++) {
            float2 samplePoint = coordinates.xy + float2(float(offsetX), float(offsetY)) * frame.shadowTexelSize;
            visible += shadowMap.sample_compare(shadowSampler, samplePoint, coordinates.z - 0.0005);
        }
    }
    return mix(1.0, visible / 9.0, shadowRegionFade(coordinates.xy));
}

// surfaceCoordinates are pattern coordinates on the surface: x and z on flat ground; along the
// wall and up on walls; along the slope's contour and down the slope on roofs.
float2 surfaceCoordinates(float3 normal, float3 worldPosition, constant DrawUniforms &draw) {
    float3 position = worldPosition / draw.patternScale;
    if (abs(normal.y) > 0.92) return position.xz;
    float2 across = normalize(float2(-normal.z, normal.x) + float2(1e-5, 0.0));
    return float2(dot(position.xz, across), position.y);
}

float cobbles(float2 point) {
    float2 cell = floor(point);
    float nearest = 8.0, second = 8.0;
    float2 nearestCell = cell;
    for (int column = -1; column <= 1; column++) {
        for (int row = -1; row <= 1; row++) {
            float2 neighbour = cell + float2(float(column), float(row));
            float gap = length(neighbour + 0.15 + 0.7 * hashPairOf(neighbour) - point);
            if (gap < nearest) { second = nearest; nearest = gap; nearestCell = neighbour; }
            else if (gap < second) { second = gap; }
        }
    }
    float joint = smoothstep(0.03, 0.12, second - nearest);
    return mix(0.55, 0.82 + 0.3 * hashOf(nearestCell), joint) * (1.0 - 0.12 * nearest);
}

float masonry(float2 point) {
    float course = floor(point.y);
    float along = point.x * 0.5 + 0.5 * glslMod(course, 2.0);
    float2 block = float2(floor(along), course);
    float joint = smoothstep(0.0, 0.06, fract(point.y)) * smoothstepAnyOrder(1.0, 0.94, fract(point.y))
                * smoothstep(0.0, 0.03, fract(along)) * smoothstepAnyOrder(1.0, 0.97, fract(along));
    return mix(0.62, 0.84 + 0.26 * hashOf(block) + 0.06 * valueNoise(point * 4.0), joint);
}

float shingles(float2 point) {
    float row = floor(point.y);
    float along = point.x * 1.4 + 0.5 * glslMod(row, 2.0);
    float tone = 0.8 + 0.3 * hashOf(float2(floor(along), row));
    float overlap = 0.72 + 0.28 * fract(point.y);
    float gap = smoothstep(0.0, 0.05, fract(along)) * smoothstepAnyOrder(1.0, 0.95, fract(along));
    return tone * overlap * mix(0.7, 1.0, gap);
}

float planks(float2 point) {
    float board = floor(point.y);
    float gap = smoothstep(0.0, 0.05, fract(point.y)) * smoothstepAnyOrder(1.0, 0.95, fract(point.y));
    float grain = 0.94 + 0.08 * valueNoise(float2(point.x * 0.6, point.y * 9.0));
    return mix(0.6, (0.82 + 0.26 * hashOf(float2(board, floor(point.x * 0.25 + hashOf(float2(board, 3.0)))))) * grain, gap);
}

// patternHeight is the relief of a pattern (0 in the joints, up to 1 on a stone, a tile or a
// board), which bends the light: rounded cobbles, sunken joints, overlapping tiles, board gaps.
float patternHeight(float2 point, int pattern) {
    if (pattern == 1) {
        float2 cell = floor(point);
        float nearest = 8.0, second = 8.0;
        for (int column = -1; column <= 1; column++) {
            for (int row = -1; row <= 1; row++) {
                float2 neighbour = cell + float2(float(column), float(row));
                float gap = length(neighbour + 0.15 + 0.7 * hashPairOf(neighbour) - point);
                if (gap < nearest) { second = nearest; nearest = gap; }
                else if (gap < second) { second = gap; }
            }
        }
        float edge = smoothstep(0.0, 0.35, second - nearest);
        return sqrt(edge) * (1.0 - 0.35 * nearest * nearest);
    }
    if (pattern == 2) {
        float course = floor(point.y);
        float along = point.x * 0.5 + 0.5 * glslMod(course, 2.0);
        return smoothstep(0.0, 0.08, fract(point.y)) * smoothstepAnyOrder(1.0, 0.92, fract(point.y))
             * smoothstep(0.0, 0.04, fract(along)) * smoothstepAnyOrder(1.0, 0.96, fract(along))
             * (0.85 + 0.15 * valueNoise(point * 3.0));
    }
    if (pattern == 4) {
        float along = point.x * 1.4 + 0.5 * glslMod(floor(point.y), 2.0);
        return fract(point.y) * smoothstep(0.0, 0.06, fract(along)) * smoothstepAnyOrder(1.0, 0.94, fract(along));
    }
    if (pattern == 5) return smoothstep(0.0, 0.06, fract(point.y)) * smoothstepAnyOrder(1.0, 0.94, fract(point.y));
    if (pattern == 3 || pattern == 6) return valueNoise(point * 3.0);
    return 0.0;
}

// bumpStrength is how far each pattern's relief bends the normal.
float bumpStrength(int pattern) {
    if (pattern == 1) return 0.55;
    if (pattern == 2) return 0.35;
    if (pattern == 4) return 0.45;
    if (pattern == 5) return 0.25;
    return 0.08;
}

// bumpedNormal bends a surface's normal by its pattern's relief, fading out with distance like the
// pattern itself.
float3 bumpedNormal(float3 normal, float3 worldPosition, constant DrawUniforms &draw) {
    if (draw.pattern == 0 || draw.patternScale <= 0.0) return normal;
    float cellsPerPixel = length(fwidth(worldPosition)) / draw.patternScale;
    float strength = (1.0 - smoothstep(0.15, 0.6, cellsPerPixel)) * bumpStrength(draw.pattern);
    if (strength <= 0.0) return normal;
    float2 point = surfaceCoordinates(normal, worldPosition, draw);
    // GLSL names this step; here that name would hide the step function
    const float stepSize = 0.04;
    float here = patternHeight(point, draw.pattern);
    float slopeU = (patternHeight(point + float2(stepSize, 0.0), draw.pattern) - here) / stepSize;
    float slopeV = (patternHeight(point + float2(0.0, stepSize), draw.pattern) - here) / stepSize;
    float3 tangentU, tangentV;
    if (abs(normal.y) > 0.92) {
        tangentU = float3(1.0, 0.0, 0.0);
        tangentV = float3(0.0, 0.0, 1.0);
    } else {
        float2 across = normalize(float2(-normal.z, normal.x) + float2(1e-5, 0.0));
        tangentU = float3(across.x, 0.0, across.y);
        tangentV = float3(0.0, 1.0, 0.0);
    }
    return normalize(normal - strength * 0.1 * (slopeU * tangentU + slopeV * tangentV));
}

// patternFactor multiplies a surface's colour by the material's pattern, fading it out where a
// cell covers less than about two pixels.
float patternFactor(float3 normal, float3 worldPosition, constant DrawUniforms &draw) {
    if (draw.pattern == 0 || draw.patternScale <= 0.0) return 1.0;
    float cellsPerPixel = length(fwidth(worldPosition)) / draw.patternScale;
    float strength = 1.0 - smoothstep(0.25, 0.8, cellsPerPixel);
    if (strength <= 0.0) return 1.0;
    float2 point = surfaceCoordinates(normal, worldPosition, draw);
    float factor = 1.0;
    if (draw.pattern == 1) factor = cobbles(point);
    else if (draw.pattern == 2) factor = masonry(point);
    else if (draw.pattern == 3) factor = 0.9 + 0.1 * valueNoise(point * 0.8) + 0.06 * valueNoise(point * 7.0);
    else if (draw.pattern == 4) factor = shingles(point);
    else if (draw.pattern == 5) factor = planks(point);
    else if (draw.pattern == 6) factor = 0.82 + 0.2 * valueNoise(point * 0.7) + 0.1 * valueNoise(point * 5.0);
    return mix(1.0, factor, strength);
}

float3 shadeLit(float3 color, SceneVaryings input, bool frontFacing,
                constant FrameUniforms &frame, constant DrawUniforms &draw, depth2d<float> shadowMap) {
    float3 normal = normalize(input.worldNormal);
    if (!frontFacing) normal = -normal;
    float3 surface = bumpedNormal(normal, input.worldPosition, draw);
    float3 towardsSun = normalize(frame.sunDirection.xyz);
    float shadowed = sunVisibility(normal, input.worldPosition, frame, shadowMap);
    float sunlight = max(dot(surface, towardsSun), 0.0) * shadowed;
    // the relief also hides light from the sky: joints and gaps are darker
    float cavity = mix(1.0, 0.8 + 0.2 * max(dot(surface, normal), 0.0), step(0.5, float(draw.pattern)));
    float3 skylight = mix(frame.groundColor.xyz, frame.skyColor.xyz, surface.y * 0.5 + 0.5) * frame.ambient * cavity;
    // a soft sheen on stone, tiles and wood under the sun
    float3 towardsCamera = normalize(frame.cameraPosition.xyz - input.worldPosition);
    float sheen = draw.pattern > 0 ? pow(max(dot(normalize(towardsSun + towardsCamera), surface), 0.0), 24.0) * 0.12 * shadowed : 0.0;
    return color * (skylight + frame.sunColor.xyz * sunlight) + frame.sunColor.xyz * sheen;
}

float3 waterNormal(float3 worldPosition, constant FrameUniforms &frame) {
    float2 position = worldPosition.xz;
    float2 slope = float2(0.0);
    const float2 directions[4] = {normalize(float2(1.0, 0.4)), normalize(float2(-0.6, 1.0)), normalize(float2(0.3, -1.0)), normalize(float2(-1.0, -0.2))};
    const float frequencies[4] = {1.0, 1.9, 3.4, 5.9};
    float waveNumber = 6.2831853 / max(frame.waveLength, 1e-6);
    for (int wave = 0; wave < 4; wave++) {
        float frequency = waveNumber * frequencies[wave];
        slope += directions[wave] * cos(dot(position, directions[wave]) * frequency + frame.time * (1.2 + float(wave) * 0.5)) * (0.5 / frequencies[wave]);
    }
    // waves fade with distance so that far water does not alias into regular stripes
    float fade = 1.0 / (1.0 + distance(frame.cameraPosition.xyz, worldPosition) / (frame.waveLength * 60.0));
    return normalize(float3(-slope.x * 0.12 * fade, 1.0, -slope.y * 0.12 * fade));
}

// seabedHeight returns the height of the ground under the fragment (unscaled world units). As with
// the shadow map, v is flipped because the seabed map's row 0 is its top.
float seabedHeight(float3 worldPosition, constant FrameUniforms &frame, depth2d<float> seabedMap) {
    float4 projected = frame.seabedViewProjection * float4(worldPosition.x, 0.0, worldPosition.z, 1.0);
    float2 deviceCoordinates = projected.xy / projected.w;
    float2 coordinates = float2(deviceCoordinates.x * 0.5 + 0.5, 0.5 - deviceCoordinates.y * 0.5);
    float depth = seabedMap.sample(seabedSampler, coordinates);
    return mix(frame.seabedHeightRange.x, frame.seabedHeightRange.y, depth);
}

// waterDepthHere is the depth of the sea under the fragment (the surface may vary with the tides).
float waterDepthHere(float3 worldPosition, constant FrameUniforms &frame, depth2d<float> seabedMap) {
    return max(worldPosition.y / frame.verticalScale - seabedHeight(worldPosition, frame, seabedMap), 0.0);
}

// waterBodyColor goes from the shallow to the deep color as the water under the fragment deepens.
float4 waterBodyColor(float4 deepColor, float3 worldPosition, constant FrameUniforms &frame,
                      constant DrawUniforms &draw, depth2d<float> seabedMap) {
    if (frame.seabedEnabled == 0 || draw.seaSurface == 0) return deepColor;
    float waterDepth = waterDepthHere(worldPosition, frame, seabedMap);
    float relativeDepth = waterDepth / max(frame.waterColorDepth, 1e-6);
    float4 color = mix(frame.waterShallowColor, deepColor, 1.0 - exp(-relativeDepth));
    // deep water hides the seabed completely, including where the geometry ends
    color.a = mix(color.a, 1.0, smoothstep(3.0, 6.0, relativeDepth));
    return color;
}

// softenShore fades the sea into the beach instead of cutting it where it meets the ground, with a
// thin moving line of foam just before the water ends.
float4 softenShore(float4 color, float3 worldPosition, constant FrameUniforms &frame, depth2d<float> seabedMap) {
    float depth = waterDepthHere(worldPosition, frame, seabedMap);
    float fade = max(frame.shoreFadeDepth, 1e-6);
    float surf = 0.5 + 0.5 * sin(frame.time * 1.4 + (worldPosition.x + worldPosition.z) * 3.0 / fade * 0.01);
    float foam = (1.0 - smoothstep(fade * 0.3, fade * 1.3, depth)) * smoothstep(0.0, fade * 0.3, depth) * (0.55 + 0.45 * surf);
    color.rgb = mix(color.rgb, float3(0.95) * frame.brightness, foam * 0.6);
    color.a *= smoothstep(0.0, fade, depth);
    return color;
}

float4 shadeWater(float4 color, float3 towardsCamera, float3 worldPosition, constant FrameUniforms &frame,
                  constant DrawUniforms &draw, depth2d<float> seabedMap) {
    color = waterBodyColor(color, worldPosition, frame, draw, seabedMap);
    color.rgb *= frame.brightness;
    float3 normal = waterNormal(worldPosition, frame);
    float fresnel = pow(1.0 - max(dot(normal, towardsCamera), 0.0), 4.0) * 0.85 + 0.03;
    float3 shaded = mix(color.rgb, mix(frame.horizonColor.xyz, frame.skyColor.xyz, 0.4), fresnel);
    float3 reflected = reflect(-normalize(frame.sunDirection.xyz), normal);
    shaded += frame.sunColor.xyz * pow(max(dot(reflected, towardsCamera), 0.0), 180.0) * 1.6;
    float4 result = float4(shaded, mix(color.a, 1.0, fresnel));
    if (draw.seaSurface == 1 && frame.seabedEnabled == 1) result = softenShore(result, worldPosition, frame, seabedMap);
    return result;
}

float waterfallOpacity(float baseOpacity, float3 worldPosition, constant FrameUniforms &frame) {
    float lane = floor((worldPosition.x + worldPosition.z) * 40.0);
    float streak = fract(worldPosition.y * 3.0 + frame.time * (0.9 + pseudoRandom(lane) * 0.6) + pseudoRandom(lane));
    return baseOpacity * (0.45 + 0.55 * smoothstep(0.0, 0.5, streak));
}

// sceneFragment shades by material kind: 0 lit, 1 unlit, 2 water, 3 waterfall, 4 glow; lit and
// glowing materials can carry a procedural surface pattern (see scene.Pattern).
fragment float4 sceneFragment(SceneVaryings input [[stage_in]],
                              bool frontFacing [[front_facing]],
                              constant FrameUniforms &frame [[buffer(0)]],
                              constant DrawUniforms &draw [[buffer(1)]],
                              depth2d<float> shadowMap [[texture(0)]],
                              depth2d<float> seabedMap [[texture(1)]]) {
    float4 color = input.surfaceColor * draw.baseColor;
    float3 worldPosition = input.worldPosition;
    float3 towardsCamera = normalize(frame.cameraPosition.xyz - worldPosition);
    if (draw.kind == 0 || draw.kind == 4) {
        float3 normal = normalize(input.worldNormal);
        if (!frontFacing) normal = -normal;
        color.rgb *= patternFactor(normal, worldPosition, draw);
        if (draw.foliage == 1 && frame.foliageTint.a > 0.0) {
            float lightness = 0.6 + 0.8 * dot(color.rgb, float3(0.333));
            color.rgb = mix(color.rgb, frame.foliageTint.rgb * lightness, frame.foliageTint.a);
        }
        if (frame.snowEnabled == 1) {
            float height = worldPosition.y / max(frame.verticalScale, 1e-6);
            float covered = smoothstep(frame.snowLevel - frame.snowBlend, frame.snowLevel + frame.snowBlend, height) * smoothstep(0.45, 0.8, normal.y);
            color.rgb = mix(color.rgb, frame.snowColor.xyz, covered);
        }
    }
    if (draw.kind == 0) {
        color.rgb = shadeLit(color.rgb, input, frontFacing, frame, draw, shadowMap);
    } else if (draw.kind == 4) {
        // glows: lit by day, its own colour in the dark
        color.rgb = max(shadeLit(color.rgb, input, frontFacing, frame, draw, shadowMap), color.rgb * (1.0 - frame.brightness) * 1.4);
    } else if (draw.kind == 2) {
        color = shadeWater(color, towardsCamera, worldPosition, frame, draw, seabedMap);
    } else if (draw.kind == 3) {
        color.a = waterfallOpacity(color.a, worldPosition, frame);
        color.rgb *= frame.brightness;
    } else {
        color.rgb *= frame.brightness;
    }
    if (frame.fogRange.y > frame.fogRange.x) {
        float fogAmount = smoothstep(frame.fogRange.x, frame.fogRange.y, distance(frame.cameraPosition.xyz, worldPosition));
        color.rgb = mix(color.rgb, frame.fogColor.xyz, fogAmount);
        color.a = mix(color.a, 1.0, fogAmount * step(1.5, float(draw.kind)));
    }
    return float4(pow(max(color.rgb, float3(0.0)), float3(1.0 / 2.2)), color.a);
}

// ---------------------------------------------------------------------------------------------
// The sky for any time of day: a day gradient with the sun, a twilight glow, a night with
// procedural stars and a moon with phases, and a cloud layer over both. A day or night image
// (equirectangular or cube) can replace the procedural sky of that time.
// ---------------------------------------------------------------------------------------------

struct SkyVaryings {
    float4 position [[position]];
    float2 screenPosition;
};

// skyVertex draws a full-screen triangle at the far plane (clip z = w is depth 1 in both APIs).
vertex SkyVaryings skyVertex(uint vertexIndex [[vertex_id]]) {
    int corner = int(vertexIndex);
    float2 position = float2(float((corner << 1) & 2), float(corner & 2)) * 2.0 - 1.0;
    SkyVaryings output;
    output.screenPosition = position;
    output.position = float4(position, 1.0, 1.0);
    return output;
}

constant float pi = 3.14159265;
constant float starCells = 220.0;  // star cells per cube face side

float fractalNoise(float2 position) {
    float total = 0.0;
    float amplitude = 0.5;
    for (int octave = 0; octave < 5; octave++) {
        total += valueNoise(position) * amplitude;
        position = position * 2.03 + float2(17.0, 9.0);
        amplitude *= 0.5;
    }
    return total;
}

// viewDirection unprojects with OpenGL's inverse view-projection: the screen position is the same
// in both clip spaces (y up), and the far point at depth 1 is OpenGL's z = 1.
float3 viewDirection(float2 screenPosition, constant SkyUniforms &sky) {
    float4 farPoint = sky.inverseViewProjection * float4(screenPosition, 1.0, 1.0);
    return normalize(farPoint.xyz / farPoint.w - sky.cameraPosition.xyz);
}

// samplePanorama reads an equirectangular image. The horizontal coordinate jumps where the
// panorama wraps around, so its screen derivative is taken from a copy shifted by half a turn.
// (Metal's screen y runs down, so dfdy has the opposite sign of GLSL's dFdy; only the size of the
// gradient picks the mipmap level, so the result is the same.)
float3 samplePanorama(texture2d<float> panorama, float3 direction) {
    float2 coordinates = float2(atan2(direction.x, -direction.z) / (2.0 * pi) + 0.5, acos(clamp(direction.y, -1.0, 1.0)) / pi);
    float shifted = fract(coordinates.x + 0.5);
    float2 alongX = float2(dfdx(coordinates.x), dfdx(shifted));
    float2 alongY = float2(dfdy(coordinates.x), dfdy(shifted));
    float gradientX = abs(alongX.x) < abs(alongX.y) ? alongX.x : alongX.y;
    float gradientY = abs(alongY.x) < abs(alongY.y) ? alongY.x : alongY.y;
    return panorama.sample(panoramaSampler, coordinates,
                           gradient2d(float2(gradientX, dfdx(coordinates.y)), float2(gradientY, dfdy(coordinates.y)))).rgb;
}

float3 dayGradient(float3 direction, constant SkyUniforms &sky) {
    float elevation = max(direction.y, 0.0);
    float3 color = mix(sky.horizonColor.xyz, sky.skyColor.xyz, pow(elevation, 0.55));
    // the twilight glow is strongest on the side of the sun
    float sunSide = max(dot(normalize(direction.xz + 1e-5), normalize(sky.sunDirection.xz + 1e-5)), 0.0);
    color += sky.twilightColor.xyz * sky.twilightAmount * pow(1.0 - elevation, 6.0) * (0.25 + 0.75 * sunSide) * 0.6;
    return color;
}

// cubeCell finds the star cell of a celestial direction: the cube face and the cell on it.
float3 cubeCell(float3 celestial, thread float2 &insideCell) {
    float3 magnitude = abs(celestial);
    float2 face;
    float faceNumber;
    if (magnitude.x >= magnitude.y && magnitude.x >= magnitude.z) { face = celestial.yz / magnitude.x; faceNumber = sign(celestial.x); }
    else if (magnitude.y >= magnitude.z) { face = celestial.xz / magnitude.y; faceNumber = 2.0 * sign(celestial.y); }
    else { face = celestial.xy / magnitude.z; faceNumber = 3.0 * sign(celestial.z); }
    float2 grid = (face * 0.5 + 0.5) * starCells;
    insideCell = fract(grid);
    return float3(floor(grid), faceNumber);
}

float3 starField(float3 direction, constant SkyUniforms &sky) {
    float3 celestial = upperLeft(sky.starRotation) * direction;
    // derivatives first: they are undefined inside the branches below
    float cellInPixels = (2.0 / starCells) / max(length(fwidth(celestial)), 1e-6);
    float2 insideCell;
    float3 cell = cubeCell(celestial, insideCell);
    if (hashOf(cell) > sky.starDensity) return float3(0.0);
    float2 center = float2(hashOf(cell + 1.3), hashOf(cell + 2.7)) * 0.6 + 0.2;
    float distanceInPixels = length(insideCell - center) * cellInPixels;
    float magnitude = pow(hashOf(cell + 4.1), 10.0) * 3.0 + 0.12;
    float twinkle = 1.0 + sky.starTwinkle * sin(sky.time * (2.0 + hashOf(cell + 5.9) * 3.0) + hashOf(cell) * 40.0);
    float temperature = hashOf(cell + 7.3);
    float3 tint = mix(float3(1.0, 0.78, 0.6), float3(0.72, 0.82, 1.0), temperature);
    float point = exp(-distanceInPixels * distanceInPixels * 1.6);
    float extinction = smoothstep(-0.02, 0.18, direction.y);
    return tint * point * magnitude * twinkle * sky.starBrightness * extinction;
}

// moonDisc returns the moon color (rgb) and coverage (a) with its phase.
float4 moonDisc(float3 direction, constant SkyUniforms &sky) {
    if (sky.moonEnabled == 0) return float4(0.0);
    float3 moonDirection = sky.moonDirection.xyz;
    float angle = acos(clamp(dot(direction, moonDirection), -1.0, 1.0));
    float edge = sky.moonRadius * 0.04;
    float coverage = 1.0 - smoothstep(sky.moonRadius - edge, sky.moonRadius + edge, angle);
    float3 right = normalize(cross(moonDirection, float3(0.0, 1.0, 0.0)) + float3(1e-5, 0.0, 0.0));
    float3 up = cross(right, moonDirection);
    float2 onDisc = float2(dot(direction, right), dot(direction, up)) / max(sky.moonRadius, 1e-5);
    float3 surfaceNormal = float3(onDisc, sqrt(max(1.0 - dot(onDisc, onDisc), 0.0)));
    float phaseAngle = 2.0 * pi * sky.moonPhase;
    float3 towardsLight = float3(sin(phaseAngle), 0.0, -cos(phaseAngle));
    float lit = smoothstep(-0.03, 0.03, dot(surfaceNormal, towardsLight));
    float maria = 0.82 + 0.18 * valueNoise(onDisc * 3.0 + 5.0);
    float3 color = sky.moonColor.xyz * (lit * maria + 0.025);
    float illuminated = 0.5 - 0.5 * cos(phaseAngle);
    float halo = pow(max(dot(direction, moonDirection), 0.0), 2000.0) * 0.25 * illuminated * (1.0 - coverage);
    return float4(color * coverage + sky.moonColor.xyz * halo, max(coverage, halo));
}

float4 cloudLayer(float3 direction, constant SkyUniforms &sky) {
    if (sky.cloudCoverage <= 0.0 || direction.y <= 0.0) return float4(0.0);
    float2 drift = sky.cloudOffset + sky.windDirection * sky.time * sky.cloudSpeed;
    float2 position = direction.xz / (direction.y + 0.12) * (1.6 / max(sky.cloudScale, 1e-3)) + drift;
    // the noise stays mostly between 0.3 and 0.7, so the coverage moves the threshold inside that range
    float threshold = mix(0.72, 0.3, sky.cloudCoverage);
    float density = smoothstep(threshold, threshold + 0.14, fractalNoise(position));
    density *= smoothstep(0.0, 0.18, direction.y);
    float towardsSun = max(dot(direction, normalize(sky.sunDirection.xyz)), 0.0);
    float3 litColor = sky.cloudColor.xyz * (sky.sunColor.xyz * (0.55 + 0.45 * pow(towardsSun, 8.0)) + sky.horizonColor.xyz * 0.5);
    float3 nightColor = sky.horizonColor.xyz * 1.6 + sky.skyColor.xyz;
    float3 color = mix(nightColor, litColor, sky.dayAmount) + sky.twilightColor.xyz * sky.twilightAmount * 0.35;
    return float4(color, density * 0.92);
}

float3 daySky(float3 direction, constant SkyUniforms &sky, texture2d<float> dayPanorama, texturecube<float> dayCube) {
    if (sky.dayImageKind == 1) return samplePanorama(dayPanorama, direction);
    if (sky.dayImageKind == 2) return dayCube.sample(cubeSampler, direction).rgb;
    float3 color = dayGradient(direction, sky);
    float towardsSun = max(dot(direction, normalize(sky.sunDirection.xyz)), 0.0);
    color += sky.sunColor.xyz * sky.sunVisible * (pow(towardsSun, 900.0) * 3.0 + pow(towardsSun, 12.0) * 0.25);
    return color;
}

float3 nightSky(float3 direction, constant SkyUniforms &sky, texture2d<float> nightPanorama, texturecube<float> nightCube) {
    float3 celestial = upperLeft(sky.starRotation) * direction;
    if (sky.nightImageKind == 1) return samplePanorama(nightPanorama, celestial);
    if (sky.nightImageKind == 2) return nightCube.sample(cubeSampler, celestial).rgb;
    return dayGradient(direction, sky);
}

fragment float4 skyFragment(SkyVaryings input [[stage_in]],
                            constant SkyUniforms &sky [[buffer(0)]],
                            texture2d<float> dayPanorama [[texture(0)]],
                            texture2d<float> nightPanorama [[texture(1)]],
                            texturecube<float> dayCube [[texture(2)]],
                            texturecube<float> nightCube [[texture(3)]]) {
    float3 direction = viewDirection(input.screenPosition, sky);
    float nightAmount = 1.0 - sky.dayAmount;
    float3 color = mix(nightSky(direction, sky, nightPanorama, nightCube), daySky(direction, sky, dayPanorama, dayCube), sky.dayAmount);
    if (sky.nightImageKind == 0) {
        // stars appear only once the sky is dark enough
        float darkness = pow(nightAmount, 4.0);
        color += starField(direction, sky) * clamp(darkness + sky.starDaytimeVisibility * (1.0 - darkness), 0.0, 1.0);
    }
    float4 moon = moonDisc(direction, sky);
    color = mix(color, moon.rgb + color * 0.2, moon.a * smoothstep(-0.02, 0.02, direction.y) * mix(0.35, 1.0, nightAmount));
    float4 clouds = cloudLayer(direction, sky);
    color = mix(color, clouds.rgb, clouds.a);
    if (direction.y < 0.0) color = mix(sky.horizonColor.xyz, sky.horizonColor.xyz * 0.8, min(-direction.y * 4.0, 1.0));
    return float4(pow(max(color, float3(0.0)), float3(1.0 / 2.2)), 1.0);
}

// ---------------------------------------------------------------------------------------------
// Constellations: stars and lines on the sky sphere. Directions turn with the night sky and are
// drawn at the far plane around the camera.
// ---------------------------------------------------------------------------------------------

struct ConstellationVaryings {
    float4 position [[position]];
    float pointSize [[point_size]];
};

vertex ConstellationVaryings constellationVertex(uint vertexIndex [[vertex_id]],
                                                 const device packed_float3 *directions [[buffer(0)]],
                                                 constant ConstellationUniforms &constellation [[buffer(1)]]) {
    float3 world = transpose(upperLeft(constellation.starRotation)) * normalize(float3(directions[vertexIndex]));
    float4 position = constellation.viewProjection * float4(constellation.cameraPosition.xyz + world * 1000.0, 1.0);
    position.z = position.w * 0.99999;
    ConstellationVaryings output;
    output.position = position;
    output.pointSize = constellation.pointSize;
    return output;
}

// constellationLineFragment colours constellation lines, fading with daylight.
fragment float4 constellationLineFragment(constant ConstellationUniforms &constellation [[buffer(0)]]) {
    float alpha = constellation.lineColor.a * constellation.visibility;
    return float4(pow(constellation.lineColor.rgb, float3(1.0 / 2.2)), alpha);
}

// constellationPointFragment colours round constellation stars, fading with daylight.
fragment float4 constellationPointFragment(float2 pointCoordinate [[point_coord]],
                                           constant ConstellationUniforms &constellation [[buffer(0)]]) {
    float alpha = constellation.lineColor.a * constellation.visibility;
    float distanceFromCenter = length(pointCoordinate - 0.5) * 2.0;
    alpha *= 1.0 - smoothstep(0.4, 1.0, distanceFromCenter);
    return float4(pow(constellation.lineColor.rgb, float3(1.0 / 2.2)), alpha);
}

// ---------------------------------------------------------------------------------------------
// The overlay: rectangles in screen pixels from the top-left corner, painted in their colour where
// the atlas (a grey coverage mask: glyphs, icons, rounded corners) covers. One instance per
// rectangle, drawn as a four-corner triangle strip, as the flat path does.
// ---------------------------------------------------------------------------------------------

struct OverlayVaryings {
    float4 position [[position]];
    float2 atlasCoordinate;
    float4 color;
};

vertex OverlayVaryings overlayVertex(uint corner [[vertex_id]],
                                     uint index [[instance_id]],
                                     const device OverlayInstance *instances [[buffer(0)]],
                                     constant OverlaySizes &sizes [[buffer(1)]]) {
    OverlayInstance quad = instances[index];
    float2 along = float2(corner & 1, corner >> 1);
    float2 position = mix(float2(quad.screen[0], quad.screen[1]), float2(quad.screen[2], quad.screen[3]), along);
    float2 atlas = mix(float2(quad.atlas[0], quad.atlas[1]), float2(quad.atlas[2], quad.atlas[3]), along);
    float2 normalized = position / sizes.screen;
    OverlayVaryings output;
    output.position = float4(normalized.x * 2.0 - 1.0, 1.0 - normalized.y * 2.0, 0.0, 1.0);
    output.atlasCoordinate = atlas / sizes.atlas;
    output.color = float4(quad.color) / 255.0;
    return output;
}

// overlayFragment's colours are already the ones shown on screen, so they are not gamma-corrected
// like the lit scene.
fragment float4 overlayFragment(OverlayVaryings input [[stage_in]],
                                texture2d<float> atlas [[texture(0)]]) {
    float coverage = atlas.sample(atlasSampler, input.atlasCoordinate).r;
    return float4(input.color.rgb, input.color.a * coverage);
}
`
