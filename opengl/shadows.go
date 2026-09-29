package opengl

import (
	"fmt"
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// ShadowSettings controls the sun shadows of a SceneRenderer.
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

func (settings ShadowSettings) resolution() int32 {
	if settings.Resolution <= 0 {
		return 4096
	}
	return settings.Resolution
}

// extent returns the half-size of the shadowed region for this camera and scene size.
func (settings ShadowSettings) extent(camera models.Camera, sceneSize float32) float32 {
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
	cameraDistance := distance(camera.Position, camera.Target)
	return clampFloat(cameraDistance*perDistance, minimum, float32(math.Max(float64(minimum), float64(maximum))))
}

// shadowCenter places the shadowed region in front of the camera: half an extent ahead along the
// view direction, or at the target when it is nearer. The region then covers what is closest to
// the viewer, including the ground under a walking camera whose target is only a direction.
func shadowCenter(camera models.Camera, extent float32) graphics.Vec4 {
	targetDistance := distance(camera.Position, camera.Target)
	if targetDistance == 0 {
		return camera.Target
	}
	ahead := float32(math.Min(float64(targetDistance), float64(extent*0.5))) / targetDistance
	return graphics.Vec4{
		camera.Position[0] + (camera.Target[0]-camera.Position[0])*ahead,
		camera.Position[1] + (camera.Target[1]-camera.Position[1])*ahead,
		camera.Position[2] + (camera.Target[2]-camera.Position[2])*ahead,
		1,
	}
}

func distance(first, second graphics.Vec4) float32 {
	deltaX, deltaY, deltaZ := first[0]-second[0], first[1]-second[1], first[2]-second[2]
	return float32(math.Sqrt(float64(deltaX*deltaX + deltaY*deltaY + deltaZ*deltaZ)))
}

func clampFloat(value, minimum, maximum float32) float32 {
	return float32(math.Max(float64(minimum), math.Min(float64(maximum), float64(value))))
}

// shadowMap is the depth texture rendered from the sun and its framebuffer.
type shadowMap struct {
	framebuffer, depthTexture uint32
	resolution                int32
	program                   *shaderProgram
}

// shadowRegion is what the lighting pass needs to look up the shadow map.
type shadowRegion struct {
	lightView, lightProjection graphics.Mat4
	texelSize                  float32 // in shadow map coordinates
	normalOffset               float32 // in world units
}

func (region shadowRegion) lightViewProjection() graphics.Mat4 {
	return region.lightProjection.Multiply(region.lightView)
}

func newShadowMap(resolution int32) (*shadowMap, error) {
	program, err := newShaderProgram(graphics.SceneVertexShader, graphics.DepthOnlyFragmentShader)
	if err != nil {
		return nil, err
	}
	shadow := &shadowMap{resolution: resolution, program: program}
	shadow.depthTexture = newDepthTexture(resolution)
	enableDepthComparison(shadow.depthTexture)
	if shadow.framebuffer, err = newDepthFramebuffer(shadow.depthTexture); err != nil {
		return nil, fmt.Errorf("shadow map: %w", err)
	}
	return shadow, nil
}

// newDepthTexture creates a square depth texture; lookups outside it read the farthest depth.
func newDepthTexture(resolution int32) uint32 {
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT24, resolution, resolution, 0, gl.DEPTH_COMPONENT, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	border := []float32{1, 1, 1, 1}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &border[0])
	gl.BindTexture(gl.TEXTURE_2D, 0)
	return texture
}

// enableDepthComparison makes lookups compare against the stored depth (hardware filtered
// shadows); everything outside the texture counts as lit.
func enableDepthComparison(texture uint32) {
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_COMPARE_MODE, gl.COMPARE_REF_TO_TEXTURE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_COMPARE_FUNC, gl.LEQUAL)
	gl.BindTexture(gl.TEXTURE_2D, 0)
}

// newDepthFramebuffer creates a framebuffer that renders only into the given depth texture.
func newDepthFramebuffer(depthTexture uint32) (uint32, error) {
	var framebuffer uint32
	gl.GenFramebuffers(1, &framebuffer)
	gl.BindFramebuffer(gl.FRAMEBUFFER, framebuffer)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, depthTexture, 0)
	gl.DrawBuffer(gl.NONE)
	gl.ReadBuffer(gl.NONE)
	status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	if status != gl.FRAMEBUFFER_COMPLETE {
		return 0, fmt.Errorf("depth framebuffer incomplete: 0x%x", status)
	}
	return framebuffer, nil
}

// fitShadowRegion places the light's orthographic box around the camera target. The box centre
// is snapped to whole texels so shadows do not shimmer when the camera moves.
func fitShadowRegion(sunDirection, center graphics.Vec4, extent, depthRange float32, resolution int32) shadowRegion {
	lightView := sunOrientation(sunDirection)
	centerInLight := lightView.Apply(graphics.Vec4{center[0], center[1], center[2], 1})
	texelWorldSize := 2 * extent / float32(resolution)
	snappedX := float32(math.Round(float64(centerInLight[0]/texelWorldSize))) * texelWorldSize
	snappedY := float32(math.Round(float64(centerInLight[1]/texelWorldSize))) * texelWorldSize
	distanceAlongView := -centerInLight[2]
	lightProjection := graphics.OrthographicOpenGL(snappedX-extent, snappedX+extent, snappedY-extent, snappedY+extent,
		distanceAlongView-depthRange, distanceAlongView+depthRange)
	return shadowRegion{lightView: lightView, lightProjection: lightProjection,
		texelSize: 1 / float32(resolution), normalOffset: texelWorldSize * 1.5}
}

// sunOrientation is a view matrix looking from the sun towards the world origin.
func sunOrientation(sunDirection graphics.Vec4) graphics.Mat4 {
	towardsSun := sunDirection.Normalize()
	up := graphics.Vec4{0, 1, 0, 0}
	if math.Abs(float64(towardsSun[1])) > 0.99 {
		up = graphics.Vec4{0, 0, -1, 0}
	}
	return graphics.LookAt(graphics.Vec4{towardsSun[0], towardsSun[1], towardsSun[2], 1}, graphics.Vec4{0, 0, 0, 1}, up)
}

// renderShadowPass draws the depth of every opaque command as seen from the sun.
func (shadow *shadowMap) renderShadowPass(commands []drawCommand, region shadowRegion, verticalScale float32, wind scene.Wind, seconds float64) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, shadow.framebuffer)
	gl.Viewport(0, 0, shadow.resolution, shadow.resolution)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.DepthMask(true)
	gl.Enable(gl.POLYGON_OFFSET_FILL)
	gl.PolygonOffset(2, 4)
	program := shadow.program
	gl.UseProgram(program.handle)
	program.setMatrix("projection", region.lightProjection)
	program.setMatrix("view", region.lightView)
	program.setFloat("verticalScale", verticalScale)
	program.setFloat("time", float32(seconds))
	setWind(program, wind)
	for _, command := range commands {
		if !castsShadow(command) {
			continue
		}
		castShadow(program, command)
	}
	gl.Disable(gl.POLYGON_OFFSET_FILL)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

func castShadow(program *shaderProgram, command drawCommand) {
	program.setMatrix("model", command.world)
	program.setFloat("sway", command.material.Sway)
	buffers := uploadMesh(command.node.Mesh)
	if command.node.Instances != nil {
		drawInstanced(program, buffers, uploadInstances(command.node.Instances, buffers))
		return
	}
	drawSingle(program, buffers)
}

// bindForLighting gives the lighting program the shadow map (texture unit 1) and its region.
func (shadow *shadowMap) bindForLighting(program *shaderProgram, region shadowRegion, enabled bool) {
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, shadow.depthTexture)
	gl.ActiveTexture(gl.TEXTURE0)
	program.setInteger("shadowMap", 1)
	program.setMatrix("lightViewProjection", region.lightViewProjection())
	program.setFloat("shadowTexelSize", region.texelSize)
	program.setFloat("shadowNormalOffset", region.normalOffset)
	if enabled {
		program.setInteger("shadowsEnabled", 1)
	} else {
		program.setInteger("shadowsEnabled", 0)
	}
}
