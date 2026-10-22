package sceneplan

import (
	"math"

	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
)

// ShadowSettings controls the sun shadows.
//
// A single shadow map covers a square region around the camera target. The region grows with the
// distance between the camera and its target, so a far view shadows the whole scene coarsely and
// a close (or walking) view gets sharp shadows nearby.
type ShadowSettings struct {
	Enabled                 bool
	Resolution              int32   // texels per side of the shadow map (0 = 4096)
	MinimumExtent           float32 // smallest half-size of the region in world units (0 = 1% of the scene)
	MaximumExtent           float32 // largest half-size (0 = the whole scene)
	ExtentPerCameraDistance float32 // half-size per unit of camera distance (0 = 1.2)
}

// DefaultShadowSettings returns enabled shadows with automatic sizes.
func DefaultShadowSettings() ShadowSettings { return ShadowSettings{Enabled: true} }

// EffectiveResolution is the shadow map's texels per side.
func (settings ShadowSettings) EffectiveResolution() int32 {
	if settings.Resolution <= 0 {
		return 4096
	}
	return settings.Resolution
}

// extent returns the half-size of the shadowed region for this camera and scene size.
func (settings ShadowSettings) extent(camera camera.Lens, sceneSize float32) float32 {
	perDistance := settings.ExtentPerCameraDistance
	if perDistance <= 0 {
		perDistance = 1.2
	}
	minimum, maximum := settings.MinimumExtent, settings.MaximumExtent
	if minimum <= 0 {
		minimum = sceneSize * 0.01
	}
	if maximum <= 0 {
		maximum = sceneSize * 0.75
	}
	reach := distance(camera.Position, camera.Target) * perDistance
	if camera.IsOrthographic() {
		// without perspective the camera stands far away whatever it shows: what counts is the
		// ground on screen, about this wide on a wide window
		reach = camera.OrthographicHeight * orthographicShadowReach
	}
	return clampFloat(reach, minimum, float32(math.Max(float64(minimum), float64(maximum))))
}

// orthographicShadowReach is the half-size of the shadow box, as a share of the height an
// orthographic camera shows: enough for the corners of a 16:10 screen.
const orthographicShadowReach = 0.95

// shadowCenter places the shadowed region in front of the camera: half an extent ahead along the
// view direction, or at the target when it is nearer. The region then covers what is closest to
// the viewer, including the ground under a walking camera whose target is only a direction.
func shadowCenter(camera camera.Lens, extent float32) graphics.Vector {
	if camera.IsOrthographic() {
		return camera.Target // the middle of the screen
	}
	targetDistance := distance(camera.Position, camera.Target)
	if targetDistance == 0 {
		return camera.Target
	}
	ahead := float32(math.Min(float64(targetDistance), float64(extent*0.5))) / targetDistance
	return graphics.Vector{
		camera.Position[0] + (camera.Target[0]-camera.Position[0])*ahead,
		camera.Position[1] + (camera.Target[1]-camera.Position[1])*ahead,
		camera.Position[2] + (camera.Target[2]-camera.Position[2])*ahead,
		1,
	}
}

func distance(first, second graphics.Vector) float32 {
	deltaX, deltaY, deltaZ := first[0]-second[0], first[1]-second[1], first[2]-second[2]
	return float32(math.Sqrt(float64(deltaX*deltaX + deltaY*deltaY + deltaZ*deltaZ)))
}

func clampFloat(value, minimum, maximum float32) float32 {
	return float32(math.Max(float64(minimum), math.Min(float64(maximum), float64(value))))
}

// ShadowRegion is what the lighting pass needs to look up the shadow map.
type ShadowRegion struct {
	LightView, LightProjection graphics.Matrix // OpenGL clip space
	TexelSize                  float32         // in shadow map coordinates
	NormalOffset               float32         // in world units
}

// LightViewProjection is the light's projection times its view.
func (region ShadowRegion) LightViewProjection() graphics.Matrix {
	return region.LightProjection.Multiply(region.LightView)
}

// fitShadowRegion places the light's orthographic box around the camera target. The box centre
// is snapped to whole texels so shadows do not shimmer when the camera moves.
func fitShadowRegion(sunDirection, center graphics.Vector, extent, depthRange float32, resolution int32) ShadowRegion {
	lightView := sunOrientation(sunDirection)
	centerInLight := lightView.Apply(graphics.Vector{center[0], center[1], center[2], 1})
	texelWorldSize := 2 * extent / float32(resolution)
	snappedX := float32(math.Round(float64(centerInLight[0]/texelWorldSize))) * texelWorldSize
	snappedY := float32(math.Round(float64(centerInLight[1]/texelWorldSize))) * texelWorldSize
	distanceAlongView := -centerInLight[2]
	lightProjection := graphics.OrthographicOpenGL(snappedX-extent, snappedX+extent, snappedY-extent, snappedY+extent,
		distanceAlongView-depthRange, distanceAlongView+depthRange)
	return ShadowRegion{LightView: lightView, LightProjection: lightProjection,
		TexelSize: 1 / float32(resolution), NormalOffset: texelWorldSize * 1.5}
}

// sunOrientation is a view matrix looking from the sun towards the world origin.
func sunOrientation(sunDirection graphics.Vector) graphics.Matrix {
	towardsSun := sunDirection.Normalize()
	up := graphics.Vector{0, 1, 0, 0}
	if math.Abs(float64(towardsSun[1])) > 0.99 {
		up = graphics.Vector{0, 0, -1, 0}
	}
	return graphics.LookAt(graphics.Vector{towardsSun[0], towardsSun[1], towardsSun[2], 1}, graphics.Vector{0, 0, 0, 1}, up)
}
