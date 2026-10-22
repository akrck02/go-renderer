package opengl

import (
	"strconv"

	"github.com/akrck02/go-renderer/failure"
	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// shadowMap is the depth texture rendered from the sun and its framebuffer.
type shadowMap struct {
	framebuffer, depthTexture uint32
	resolution                int32
	program                   *shaderProgram
}

func newShadowMap(resolution int32) (*shadowMap, error) {
	program, err := newShaderProgram(sceneVertexShader, depthOnlyFragmentShader)
	if err != nil {
		return nil, err
	}
	shadow := &shadowMap{resolution: resolution, program: program}
	shadow.depthTexture = newDepthTexture(resolution)
	enableDepthComparison(shadow.depthTexture)
	if shadow.framebuffer, err = newDepthFramebuffer(shadow.depthTexture); err != nil {
		return nil, err
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
		return 0, failure.Wrap(FramebufferIncomplete, "shadow map, status 0x"+strconv.FormatUint(uint64(status), 16), nil)
	}
	return framebuffer, nil
}

// renderShadowPass draws the depth of every opaque command as seen from the sun.
// drawCaster draws one command in the shadow pass (the renderer culls it and picks its detail).
type drawCaster func(program *shaderProgram, command sceneplan.Command)

func (shadow *shadowMap) renderShadowPass(commands []sceneplan.Command, region sceneplan.ShadowRegion, verticalScale float32, wind scene.Wind, seconds float64, draw drawCaster) {
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
	program.setMatrix("projection", region.LightProjection)
	program.setMatrix("view", region.LightView)
	program.setFloat("verticalScale", verticalScale)
	program.setFloat("time", float32(seconds))
	setWind(program, wind)
	for _, command := range commands {
		if !sceneplan.CastsShadow(command) {
			continue
		}
		draw(program, command)
	}
	gl.Disable(gl.POLYGON_OFFSET_FILL)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

// bindForLighting gives the lighting program the shadow map (texture unit 1) and its region.
func (shadow *shadowMap) bindForLighting(program *shaderProgram, region sceneplan.ShadowRegion, enabled bool) {
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, shadow.depthTexture)
	gl.ActiveTexture(gl.TEXTURE0)
	program.setInteger("shadowMap", 1)
	program.setMatrix("lightViewProjection", region.LightViewProjection())
	program.setFloat("shadowTexelSize", region.TexelSize)
	program.setFloat("shadowNormalOffset", region.NormalOffset)
	if enabled {
		program.setInteger("shadowsEnabled", 1)
	} else {
		program.setInteger("shadowsEnabled", 0)
	}
}
