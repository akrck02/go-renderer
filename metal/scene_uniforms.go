package metal

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/sceneplan"
)

// The uniform blocks of the scene shaders (scene_shaders.go), byte for byte: every field is four
// bytes, and the matrices and vectors come first so nothing needs padding. Tests check the sizes
// and offsets against the shader's.

type frameUniforms struct {
	projection           graphics.Mat4 // Metal clip space
	view                 graphics.Mat4
	lightViewProjection  graphics.Mat4 // Metal clip space
	seabedViewProjection graphics.Mat4 // Metal clip space
	sunDirection         graphics.Vec4
	sunColor             graphics.Vec4
	skyColor             graphics.Vec4
	horizonColor         graphics.Vec4
	groundColor          graphics.Vec4
	fogColor             graphics.Vec4
	cameraPosition       graphics.Vec4
	foliageTint          graphics.Vec4
	snowColor            graphics.Vec4
	waterShallowColor    graphics.Vec4
	levelVariationArea   graphics.Vec4
	windDirection        [2]float32
	fogRange             [2]float32
	seabedHeightRange    [2]float32
	unusedPair           [2]float32
	verticalScale        float32
	windStrength         float32
	ambient              float32
	brightness           float32
	time                 float32
	waveLength           float32
	shadowTexelSize      float32
	shadowNormalOffset   float32
	waterColorDepth      float32
	shoreFadeDepth       float32
	snowLevel            float32
	snowBlend            float32
	levelVariationAngle  float32
	shadowsEnabled       int32
	seabedEnabled        int32
	snowEnabled          int32
}

type drawUniforms struct {
	model                 graphics.Mat4
	baseColor             graphics.Vec4
	sway                  float32
	depthBias             float32
	patternScale          float32
	instanced             int32
	kind                  int32
	pattern               int32
	foliage               int32
	levelVariationEnabled int32
	seaSurface            int32
	unused                [3]int32
}

type skyUniforms struct {
	inverseViewProjection graphics.Mat4 // OpenGL convention: the sky works it out from the screen
	starRotation          graphics.Mat4
	cameraPosition        graphics.Vec4
	sunDirection          graphics.Vec4
	sunColor              graphics.Vec4
	skyColor              graphics.Vec4
	horizonColor          graphics.Vec4
	twilightColor         graphics.Vec4
	moonDirection         graphics.Vec4
	moonColor             graphics.Vec4
	cloudColor            graphics.Vec4
	cloudOffset           [2]float32
	windDirection         [2]float32
	time                  float32
	sunVisible            float32
	dayAmount             float32
	twilightAmount        float32
	starDensity           float32
	starBrightness        float32
	starTwinkle           float32
	starDaytimeVisibility float32
	moonRadius            float32
	moonPhase             float32
	cloudCoverage         float32
	cloudSpeed            float32
	cloudScale            float32
	moonEnabled           int32
	dayImageKind          int32
	nightImageKind        int32
}

type constellationUniforms struct {
	viewProjection graphics.Mat4 // Metal clip space
	starRotation   graphics.Mat4
	cameraPosition graphics.Vec4
	lineColor      graphics.Vec4
	visibility     float32
	pointSize      float32
	unused         [2]int32
}

type overlaySizes struct {
	screen, atlas [2]float32
}

// metalClipSpace turns an OpenGL projection (depth from -1 to 1) into Metal's (depth from 0 to
// 1): the same picture, and the same depths once stored.
func metalClipSpace(projection graphics.Mat4) graphics.Mat4 {
	correction := graphics.Identity()
	correction[10], correction[14] = 0.5, 0.5 // z' = z/2 + w/2
	return correction.Multiply(projection)
}

// frameUniformsFor fills what every draw of the main pass shares, as the OpenGL renderer's
// setFrameUniforms, bindForLighting, bindForWater and the level variation do.
func frameUniformsFor(environment scene.Environment, frame sceneplan.Frame, camera models.Camera, seconds float64, shadowsReady bool) frameUniforms {
	light := frame.Light
	uniforms := frameUniforms{
		projection: metalClipSpace(frame.Matrices.Projection), view: frame.Matrices.View,
		sunDirection: light.LightDirection, sunColor: light.LightColor, skyColor: light.SkyLight,
		horizonColor: light.Horizon, groundColor: light.GroundLight, fogColor: light.Fog,
		cameraPosition: camera.Position, foliageTint: environment.FoliageTint,
		fogRange:      [2]float32{environment.FogNear, environment.FogFar},
		verticalScale: environment.EffectiveVerticalScale(), windStrength: environment.Wind.Strength,
		ambient: environment.Ambient, brightness: light.Brightness, time: float32(seconds),
		waveLength: sceneplan.WaveLengthFor(environment.Water),
	}
	uniforms.windDirection[0], uniforms.windDirection[1] = sceneplan.WindDirection(environment.Wind)
	if snow := environment.Snow; snow != nil {
		uniforms.snowEnabled, uniforms.snowLevel, uniforms.snowColor = 1, snow.Level, snow.Color
		uniforms.snowBlend = max(snow.Blend, 1e-6)
	}
	region := frame.Shadow.Region
	uniforms.lightViewProjection = metalClipSpace(region.LightViewProjection())
	uniforms.shadowTexelSize, uniforms.shadowNormalOffset = region.TexelSize, region.NormalOffset
	uniforms.shadowsEnabled = boolean(shadowsReady)
	if water := environment.Water; water != nil && frame.Seabed.Available {
		uniforms.seabedEnabled = 1
		uniforms.seabedViewProjection = metalClipSpace(frame.Seabed.Region.ViewProjection)
		uniforms.seabedHeightRange = [2]float32{frame.Seabed.Region.Top, frame.Seabed.Region.Bottom}
		uniforms.waterShallowColor = sceneplan.ShallowColorOf(water)
		uniforms.waterColorDepth, uniforms.shoreFadeDepth = water.EffectiveColorDepth(), water.EffectiveShoreFade()
	}
	if variation := levelVariationOf(environment.Water); variation != nil {
		uniforms.levelVariationArea = graphics.Vec4{variation.MinimumX, variation.MinimumZ, variation.MaximumX, variation.MaximumZ}
		uniforms.levelVariationAngle = variation.Angle
	}
	return uniforms
}

// shadowPassUniforms are the shadow pass's: the sun's view, and only what moves the geometry (the
// OpenGL depth program sets nothing else, so the rest stays zero, as it does there).
func shadowPassUniforms(environment scene.Environment, region sceneplan.ShadowRegion, seconds float64) frameUniforms {
	uniforms := frameUniforms{projection: metalClipSpace(region.LightProjection), view: region.LightView,
		verticalScale: environment.EffectiveVerticalScale(), time: float32(seconds), windStrength: environment.Wind.Strength}
	uniforms.windDirection[0], uniforms.windDirection[1] = sceneplan.WindDirection(environment.Wind)
	return uniforms
}

// seabedPassUniforms look straight down on the ground, without vertical exaggeration.
func seabedPassUniforms(region sceneplan.SeabedRegion) frameUniforms {
	return frameUniforms{projection: metalClipSpace(region.ViewProjection), view: graphics.Identity(), verticalScale: 1}
}

// drawUniformsFor fills a draw of a command's material, as the OpenGL renderer's drawNode does.
func drawUniformsFor(command sceneplan.Command, levelVariationActive bool) drawUniforms {
	material := command.Material
	return drawUniforms{model: command.World, baseColor: sceneplan.BaseColorOf(material),
		sway: material.Sway, depthBias: material.DepthBias, patternScale: material.PatternScale,
		kind: int32(material.Kind), pattern: int32(material.Pattern), foliage: boolean(material.Foliage),
		levelVariationEnabled: boolean(command.SeaSurface && levelVariationActive), seaSurface: boolean(command.SeaSurface)}
}

// skyUniformsFor fills the sky dome's, as the OpenGL sky renderer does.
func skyUniformsFor(environment scene.Environment, light scene.Daylight, camera models.Camera, inverse graphics.Mat4, seconds float64) skyUniforms {
	sky := environment.EffectiveSky()
	uniforms := skyUniforms{inverseViewProjection: inverse, starRotation: light.StarRotation,
		cameraPosition: camera.Position, sunDirection: environment.SunDirection.Normalize(), sunColor: environment.SunColor,
		skyColor: light.Zenith, horizonColor: light.Horizon, twilightColor: sky.TwilightColor,
		cloudColor: sceneplan.ColorOrWhite(sky.Clouds.Color), cloudOffset: [2]float32{sky.Clouds.Offset[0], sky.Clouds.Offset[2]},
		time: float32(seconds), sunVisible: light.SunVisible, dayAmount: light.DayAmount, twilightAmount: light.TwilightAmount,
		starDensity: sky.Stars.Density, starBrightness: sky.Stars.Brightness, starTwinkle: sky.Stars.Twinkle,
		starDaytimeVisibility: sky.Stars.DaytimeVisibility, cloudCoverage: sky.Clouds.Coverage,
		cloudSpeed: sky.Clouds.Speed, cloudScale: sceneplan.ScaleOrOne(sky.Clouds.Scale),
		dayImageKind: imageKind(sky.DayImage), nightImageKind: imageKind(sky.NightImage)}
	uniforms.windDirection[0], uniforms.windDirection[1] = sceneplan.WindDirection(environment.Wind)
	if moon := sky.Moon; moon != nil {
		uniforms.moonEnabled, uniforms.moonDirection = 1, light.MoonDirection
		uniforms.moonRadius, uniforms.moonColor, uniforms.moonPhase = moon.AngularSize/2, moon.Color, moon.Phase
	}
	return uniforms
}

// imageKind tells the sky shader whether an image is absent (0), a panorama (1) or a cube (2).
func imageKind(image *scene.SkyImage) int32 {
	switch {
	case image == nil:
		return 0
	case image.IsCube():
		return 2
	default:
		return 1
	}
}

func levelVariationOf(water *scene.Water) *scene.LevelVariation {
	if water == nil {
		return nil
	}
	return water.Variation
}

func boolean(value bool) int32 {
	if value {
		return 1
	}
	return 0
}
