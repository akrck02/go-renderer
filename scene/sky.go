package scene

import (
	"image"

	"github.com/akrck02/go-renderer/graphics"
)

// Sky describes what is drawn behind the scene over a whole day and how it lights the scene at
// night. The height of the sun decides the time of day: above the horizon it is day, near it
// twilight, below it night.
//
// By default the sky is procedural: the day gradient of the environment, a twilight glow, and a
// night with stars, a moon and constellations; clouds float over both. Images can replace the day
// or the night (a skybox): an equirectangular panorama or six cube faces.
type Sky struct {
	NightZenith   graphics.Vec4 // sky color straight up at night (linear)
	NightHorizon  graphics.Vec4 // sky color at the horizon at night
	NightAmbient  graphics.Vec4 // sky light that reaches the scene at night
	TwilightColor graphics.Vec4 // horizon glow at dawn and dusk
	CelestialPole graphics.Vec4 // axis the night sky turns around (towards the visible pole)

	Stars          Stars
	Moon           *Moon // nil = no moon
	Clouds         Clouds
	Constellations []Constellation

	DayImage   *SkyImage // replaces the procedural day sky
	NightImage *SkyImage // replaces the procedural night sky; it turns with the stars
}

// Stars is a procedural star field.
type Stars struct {
	Density           float32 // share of the sky cells that hold a star (0 = no stars, 1 = one per cell)
	Brightness        float32 // brightness multiplier
	Twinkle           float32 // amount of twinkling (0 = steady)
	DaytimeVisibility float32 // how visible the stars stay in full daylight (0 = only at night)
}

// Moon is a disc with phases that also lights the scene at night.
type Moon struct {
	Direction      graphics.Vec4 // towards the moon; zero = opposite the sun
	AngularSize    float32       // apparent diameter in radians
	Color          graphics.Vec4
	Phase          float32 // 0 new, 0.25 first quarter, 0.5 full, 0.75 last quarter
	LightIntensity float32 // moonlight on the scene at full moon, relative to the sun
}

// Clouds is a procedural cloud layer.
type Clouds struct {
	Coverage float32 // share of the sky covered (0 = clear)
	Color    graphics.Vec4
	Speed    float32       // drift in cloud-texture units per second
	Scale    float32       // size of the cloud pattern (larger = bigger clouds)
	Offset   graphics.Vec4 // how far the clouds have drifted (x, z), moved by a simulation
}

// Constellation is a figure of stars joined by lines, fixed to the night sky.
type Constellation struct {
	Name  string
	Stars []graphics.Vec4 // directions as seen at midnight (the sun at the lowest point of its circle)
	Lines [][2]int        // pairs of indices into Stars
	Color graphics.Vec4
}

// SkyImage is a skybox: an equirectangular panorama or six cube faces (+X, -X, +Y, -Y, +Z, -Z).
type SkyImage struct {
	Equirectangular image.Image
	Faces           [6]image.Image
	GPU             any // backend handle
}

// IsCube reports whether the image is made of six faces.
func (skyImage *SkyImage) IsCube() bool { return skyImage.Faces[0] != nil }

// DefaultSky returns an Earth-like sky: stars, a full moon opposite the sun and no clouds.
func DefaultSky() *Sky {
	return &Sky{
		NightZenith:   graphics.Vec4{0.004, 0.007, 0.018, 1},
		NightHorizon:  graphics.Vec4{0.02, 0.03, 0.055, 1},
		NightAmbient:  graphics.Vec4{0.06, 0.08, 0.14, 1},
		TwilightColor: graphics.Vec4{0.95, 0.42, 0.2, 1},
		CelestialPole: graphics.Vec4{0, 0.7, -0.71, 0}.Normalize(),
		Stars:         Stars{Density: 0.35, Brightness: 1, Twinkle: 0.25},
		Moon: &Moon{
			AngularSize:    0.02,
			Color:          graphics.Vec4{0.92, 0.93, 1, 1},
			Phase:          0.5,
			LightIntensity: 0.12,
		},
		Clouds: Clouds{Color: graphics.Vec4{1, 1, 1, 1}, Speed: 0.01, Scale: 1},
	}
}
