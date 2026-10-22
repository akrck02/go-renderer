package ctr

import (
	"math"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// The PICA200 has no fragment shaders, so the scene is shaded per vertex (scene.v.pica). A shader
// instruction reads one uniform at most, so everything that is the same for the whole frame is
// worked out here first. The shading follows gles's scene shader.

// frameShading holds the per-frame uniforms of scene.v.pica, as x, y, z, w.
type frameShading struct {
	camera        graphics.Vector // the camera's position
	sunDirection  graphics.Vector // towards the light, of unit length
	sunColour     graphics.Vector
	ambientGround graphics.Vector // the ground's light times the ambient amount
	ambientSpan   graphics.Vector // (sky - ground) × ambient: added as the normal turns up
	waterLight    graphics.Vector // sky × ambient + sun × its height: water's flat light
	skyReflection graphics.Vector // the sky × brightness, which water reflects at grazing angles
	fogColour     graphics.Vector
	settings      graphics.Vector // vertical scale, its inverse, -, brightness
	fog           graphics.Vector // fog near, 1 / (far - near) (0 without fog), -, -
}

func shadingFor(environment scene.Environment, light scene.Daylight, camera camera.Camera) frameShading {
	sunDirection := light.LightDirection.Normalize()
	ambient := environment.Ambient
	verticalScale := environment.EffectiveVerticalScale()
	shading := frameShading{
		camera:        graphics.Vector{camera.Position[0], camera.Position[1], camera.Position[2], 1},
		sunDirection:  graphics.Vector{sunDirection[0], sunDirection[1], sunDirection[2], 0},
		sunColour:     light.LightColor,
		ambientGround: scaled(light.GroundLight, ambient),
		ambientSpan:   scaled(difference(light.SkyLight, light.GroundLight), ambient),
		waterLight:    sum(scaled(light.SkyLight, ambient), scaled(light.LightColor, float32(math.Max(float64(sunDirection[1]), 0)))),
		skyReflection: scaled(light.SkyLight, light.Brightness),
		fogColour:     light.Fog,
		settings:      graphics.Vector{verticalScale, 1 / verticalScale, 0, light.Brightness},
	}
	if environment.FogFar > environment.FogNear {
		shading.fog = graphics.Vector{environment.FogNear, 1 / (environment.FogFar - environment.FogNear), 0, 0}
	}
	return shading
}

// materialWeights selects a material's shading in scene.v.pica without branches: x lit, y flat
// (unlit, waterfalls), z water, each 1 or 0; w is the glow of glowing materials, which shine with
// their own colour as the light goes.
func materialWeights(kind scene.MaterialKind, brightness float32) graphics.Vector {
	switch kind {
	case scene.KindLit:
		return graphics.Vector{1, 0, 0, 0}
	case scene.KindGlow:
		return graphics.Vector{1, 0, 0, (1 - brightness) * 1.4}
	case scene.KindWater:
		return graphics.Vector{0, 0, 1, 0}
	default:
		return graphics.Vector{0, 1, 0, 0}
	}
}

// minimumAlpha is how opaque a material is at least: water is never more than a quarter clear.
func minimumAlpha(kind scene.MaterialKind) float32 {
	if kind == scene.KindWater {
		return 0.75
	}
	return 0
}

// screenColour turns a linear colour into the screen's gamma and packs it as 0xRRGGBBAA, for
// clearing the screen to the horizon the fog fades into.
func screenColour(colour graphics.Vector) uint32 {
	channel := func(value float32) uint32 {
		if value <= 0 {
			return 0
		}
		gamma := math.Pow(math.Min(float64(value), 1), 1/2.2)
		return uint32(math.Round(gamma * 255))
	}
	return channel(colour[0])<<24 | channel(colour[1])<<16 | channel(colour[2])<<8 | 0xFF
}

// inScreenGamma turns a linear colour into the screen's gamma, keeping it as floats.
func inScreenGamma(colour graphics.Vector) graphics.Vector {
	gamma := func(value float32) float32 {
		if value <= 0 {
			return 0
		}
		return float32(math.Pow(float64(value), 1/2.2))
	}
	return graphics.Vector{gamma(colour[0]), gamma(colour[1]), gamma(colour[2]), colour[3]}
}

func scaled(vector graphics.Vector, factor float32) graphics.Vector {
	return graphics.Vector{vector[0] * factor, vector[1] * factor, vector[2] * factor, vector[3] * factor}
}

func difference(first, second graphics.Vector) graphics.Vector {
	return graphics.Vector{first[0] - second[0], first[1] - second[1], first[2] - second[2], first[3] - second[3]}
}

func sum(first, second graphics.Vector) graphics.Vector {
	return graphics.Vector{first[0] + second[0], first[1] + second[1], first[2] + second[2], first[3] + second[3]}
}
