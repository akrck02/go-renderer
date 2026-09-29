package scene

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
)

// Daylight is the state of the sky at the current sun height: the colors to paint and the single
// directional light (sun or moon) that lights and shadows the scene. Backends compute it once
// per frame with Environment.Daylight.
type Daylight struct {
	DayAmount      float32 // 1 in full day, 0 at night
	TwilightAmount float32 // 1 with the sun on the horizon, 0 far from it
	Zenith         graphics.Vec4
	Horizon        graphics.Vec4
	Fog            graphics.Vec4
	SkyLight       graphics.Vec4 // ambient light from the sky
	GroundLight    graphics.Vec4 // ambient light bounced from the ground
	SunVisible     float32       // how much of the sun disc is above the horizon
	LightDirection graphics.Vec4 // towards the light that shades the scene
	LightColor     graphics.Vec4
	MoonDirection  graphics.Vec4
	StarRotation   graphics.Mat4 // turns world directions into celestial directions
	Brightness     float32       // overall light level (1 day, low at night) for materials that are not lit
}

// Daylight computes the sky and the lighting for the current sun direction.
func (environment Environment) Daylight() Daylight {
	sky := environment.EffectiveSky()
	sunHeight := environment.SunDirection.Normalize()[1]
	light := Daylight{
		DayAmount:      smoothStep(-0.12, 0.08, sunHeight),
		TwilightAmount: clampUnit(1 - float32(math.Abs(float64(sunHeight+0.03)))/0.22),
		SunVisible:     smoothStep(-0.03, 0.02, sunHeight),
		StarRotation:   starRotation(environment.SunDirection, sky.CelestialPole),
	}
	light.paintSky(environment, sky)
	light.chooseLight(environment, sky, sunHeight)
	light.Brightness = 0.12 + 0.88*light.DayAmount
	return light
}

// paintSky blends the day and night colors and adds the twilight glow on the horizon.
func (light *Daylight) paintSky(environment Environment, sky *Sky) {
	glow := light.TwilightAmount * 0.55
	light.Zenith = mixColor(sky.NightZenith, environment.SkyZenith, light.DayAmount)
	light.Horizon = mixColor(mixColor(sky.NightHorizon, environment.SkyHorizon, light.DayAmount), sky.TwilightColor, glow)
	light.Fog = mixColor(mixColor(sky.NightHorizon, environment.FogColor, light.DayAmount), sky.TwilightColor, glow*0.6)
	light.SkyLight = mixColor(sky.NightAmbient, environment.SkyZenith, light.DayAmount)
	light.GroundLight = scaleColor(environment.GroundAmbient, 0.25+0.75*light.DayAmount)
}

// chooseLight picks the sun while it is up and the moon at night.
func (light *Daylight) chooseLight(environment Environment, sky *Sky, sunHeight float32) {
	warmth := light.TwilightAmount * 0.6
	sunStrength := smoothStep(-0.02, 0.12, sunHeight)
	sunColor := mixColor(environment.SunColor, multiplyColor(environment.SunColor, graphics.Vec4{1, 0.55, 0.35, 1}), warmth)
	light.LightDirection, light.LightColor = environment.SunDirection.Normalize(), scaleColor(sunColor, sunStrength)
	light.MoonDirection = moonDirection(environment.SunDirection, sky.Moon)
	if sky.Moon == nil {
		return
	}
	moonStrength := sky.Moon.LightIntensity * moonIllumination(sky.Moon.Phase) *
		smoothStep(-0.02, 0.12, light.MoonDirection[1]) * (1 - light.DayAmount)
	if moonStrength > sunStrength {
		light.LightDirection, light.LightColor = light.MoonDirection, scaleColor(sky.Moon.Color, moonStrength)
	}
}

// moonDirection is the configured direction or, when unset, the opposite of the sun.
func moonDirection(sunDirection graphics.Vec4, moon *Moon) graphics.Vec4 {
	if moon == nil || moon.Direction == (graphics.Vec4{}) {
		unit := sunDirection.Normalize()
		return graphics.Vec4{-unit[0], -unit[1], -unit[2], 0}
	}
	return moon.Direction.Normalize()
}

// moonIllumination is the lit share of the disc for a phase (0 new, 0.5 full).
func moonIllumination(phase float32) float32 {
	return float32(0.5 - 0.5*math.Cos(2*math.Pi*float64(phase)))
}

// starRotation turns the night sky around the celestial pole together with the sun, so the stars
// move as the day passes: it is the angle of the sun around the pole, measured from midnight (the
// sun at the lowest point of its circle), when celestial and world directions coincide.
func starRotation(sunDirection, pole graphics.Vec4) graphics.Mat4 {
	axis := pole.Normalize()
	if axis == (graphics.Vec4{}) {
		axis = graphics.Vec4{0, 1, 0, 0}
	}
	reference := perpendicularTo(axis)
	sideways := axis.Cross(reference)
	sun := sunDirection.Normalize()
	angle := math.Atan2(float64(sun.Dot(sideways)), float64(sun.Dot(reference)))
	return graphics.Rotate(axis, -angle)
}

// perpendicularTo returns a unit vector perpendicular to the axis, preferring the horizon plane.
func perpendicularTo(axis graphics.Vec4) graphics.Vec4 {
	helper := graphics.Vec4{0, 1, 0, 0}
	if math.Abs(float64(axis[1])) > 0.99 {
		helper = graphics.Vec4{0, 0, -1, 0}
	}
	return helper.Cross(axis).Cross(axis).Normalize()
}

func smoothStep(edgeLow, edgeHigh, value float32) float32 {
	amount := clampUnit((value - edgeLow) / (edgeHigh - edgeLow))
	return amount * amount * (3 - 2*amount)
}

func clampUnit(value float32) float32 {
	return float32(math.Max(0, math.Min(1, float64(value))))
}

func mixColor(from, to graphics.Vec4, amount float32) graphics.Vec4 {
	var result graphics.Vec4
	for component := range result {
		result[component] = from[component] + (to[component]-from[component])*amount
	}
	return result
}

func scaleColor(color graphics.Vec4, factor float32) graphics.Vec4 {
	return graphics.Vec4{color[0] * factor, color[1] * factor, color[2] * factor, color[3]}
}

func multiplyColor(first, second graphics.Vec4) graphics.Vec4 {
	return graphics.Vec4{first[0] * second[0], first[1] * second[1], first[2] * second[2], first[3] * second[3]}
}
