package scene

import "github.com/akrck02/go-renderer/graphics"

// PointLight is a light that shines from a point all around, as a lamp: full within Inner of it
// and fading smoothly to nothing at Outer (world units), the same fade as draw2d's point lights.
// It lights the lit materials it reaches, besides the sun and the sky; each draw takes the few
// that reach it (see sceneplan.MaxLightsPerDraw).
type PointLight struct {
	Position graphics.Vec4 // in the world, before the vertical scale
	// Color is linear, like the materials'; its alpha is the intensity (1 a plain lamp, up to a few).
	Color        graphics.Vec4
	Inner, Outer float32
}
