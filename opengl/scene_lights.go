package opengl

import (
	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// setDrawLights gives the scene shader the point lights a draw takes: their positions with the
// outer radius, and their colours with the inner one (graphics.SceneFragmentShader's lampLight).
func (renderer *SceneRenderer) setDrawLights(lights sceneplan.DrawLights) {
	program := renderer.sceneProgram
	program.setInteger("lightCount", int32(lights.Count))
	if lights.Count == 0 {
		return
	}
	positions, colours := &renderer.lightValues[0], &renderer.lightValues[1]
	for slot, index := range lights.Indices[:lights.Count] {
		light := &renderer.frameLights[index]
		copy(positions[slot*4:], light.Position[:3])
		positions[slot*4+3] = light.Outer
		copy(colours[slot*4:], light.Color[:3])
		colours[slot*4+3] = light.Inner
	}
	gl.Uniform4fv(program.location("lightPositions"), int32(lights.Count), &positions[0])
	gl.Uniform4fv(program.location("lightColors"), int32(lights.Count), &colours[0])
}
