package gles

import "github.com/akrck02/go-renderer/internal/sceneplan"

// lightUniformNames are the scene shader's light uniforms as OpenGL ES 2 names each element of an
// array: positions, then colours.
var lightUniformNames = [2][sceneplan.MaxLightsPerDraw]string{
	{"lightPositions[0]", "lightPositions[1]", "lightPositions[2]", "lightPositions[3]"},
	{"lightColors[0]", "lightColors[1]", "lightColors[2]", "lightColors[3]"},
}

// setDrawLights gives the scene shader the point lights a draw takes: positions with the outer
// radius, colours with the inner one. The program sends only what changed since the last draw.
func (renderer *SceneRenderer) setDrawLights(lights sceneplan.DrawLights) {
	program := renderer.program
	program.setInteger("lightCount", int32(lights.Count))
	for slot, index := range lights.Indices[:lights.Count] {
		light := &renderer.frameLights[index]
		position, colour := light.Position, light.Color
		position[3], colour[3] = light.Outer, light.Inner
		program.setVector4(lightUniformNames[0][slot], position)
		program.setVector4(lightUniformNames[1][slot], colour)
	}
}
