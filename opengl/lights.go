package opengl

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// lightLayer draws the overlay's 2D lights (see overlay.Lighting): it adds the lights into a
// picture of light at half the screen's resolution (light is soft, so a quarter of the pixels is
// plenty), cleared to the ambient colour, and then multiplies the frame by it. Its target is made
// once and remade only when the framebuffer changes size.
type lightLayer struct {
	addProgram, multiplyProgram *shaderProgram
	vertexArray, vertexBuffer   uint32
	emptyArray                  uint32 // the multiply's triangles come from gl_VertexID
	target                      target
	vertices                    []float32
}

// floatsPerLightVertex: screen position 2, corner 2, colour 4, picture position 2, inner 1.
const floatsPerLightVertex = 11

// lightTextureUnit is free of the overlay's and the scene's textures.
const lightTextureUnit = 11

func newLightLayer() (*lightLayer, error) {
	addProgram, err := newShaderProgram(graphics.LightVertexShader, graphics.LightFragmentShader)
	if err != nil {
		return nil, err
	}
	multiplyProgram, err := newShaderProgram(graphics.LightAreaVertexShader, graphics.LightMultiplyShader)
	if err != nil {
		return nil, err
	}
	layer := &lightLayer{addProgram: addProgram, multiplyProgram: multiplyProgram}
	gl.GenVertexArrays(1, &layer.emptyArray)
	gl.GenVertexArrays(1, &layer.vertexArray)
	gl.GenBuffers(1, &layer.vertexBuffer)
	gl.BindVertexArray(layer.vertexArray)
	gl.BindBuffer(gl.ARRAY_BUFFER, layer.vertexBuffer)
	stride := int32(floatsPerLightVertex * bytesPerFloat)
	sizes := []int32{2, 2, 4, 2, 1}
	offset := 0
	for attribute, size := range sizes {
		gl.EnableVertexAttribArray(uint32(attribute))
		gl.VertexAttribPointerWithOffset(uint32(attribute), size, gl.FLOAT, false, stride, uintptr(offset*bytesPerFloat))
		offset += int(size)
	}
	gl.BindVertexArray(0)
	return layer, nil
}

// fillVertices turns every light of the frame into two triangles, for the lightings to draw in turn.
func (layer *lightLayer) fillVertices(lights []overlay.LightQuad) {
	vertices := layer.vertices[:0]
	corners := [6][2]int{{0, 0}, {0, 2}, {2, 2}, {0, 0}, {2, 2}, {2, 0}} // edges: 0 left or top, 2 right or bottom
	for _, light := range lights {
		picture := [4]float32{-1, -1, -1, -1}
		if light.Pictured {
			picture = light.Picture
		}
		for _, corner := range corners {
			x, y := corner[0], corner[1]
			vertices = append(vertices, light.Screen[x], light.Screen[y+1], float32(x-1), float32(y-1),
				light.Color[0], light.Color[1], light.Color[2], light.Color[3], picture[x], picture[y+1], light.Inner)
		}
	}
	layer.vertices = vertices
	gl.BindBuffer(gl.ARRAY_BUFFER, layer.vertexBuffer)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*bytesPerFloat, firstFloat(vertices), gl.STREAM_DRAW)
}

// light lights what framebuffer drawn (width × height pixels) shows with one lighting, in its
// area. drawn is bound again when it returns, with the viewport over all of it and blending off.
func (layer *lightLayer) light(lighting overlay.Lighting, drawn uint32, width, height int, picturesSize int) {
	gl.ActiveTexture(gl.TEXTURE0 + lightTextureUnit) // a new target binds its texture here, not over the atlas
	layer.prepare(width, height)
	gl.Disable(gl.DEPTH_TEST)
	gl.DepthMask(false)
	layer.addLights(lighting, width, height, picturesSize)

	gl.BindFramebuffer(gl.FRAMEBUFFER, drawn)
	gl.Viewport(0, 0, int32(width), int32(height))
	gl.UseProgram(layer.multiplyProgram.handle)
	gl.BindTexture(gl.TEXTURE_2D, layer.target.texture)
	layer.multiplyProgram.setInteger("light", lightTextureUnit)
	area := lighting.Area
	gl.Uniform4f(layer.multiplyProgram.location("area"), area[0], area[1], area[2], area[3])
	gl.Uniform2f(layer.multiplyProgram.location("screenSize"), float32(width), float32(height))
	gl.Enable(gl.BLEND)
	// frame × light + light × frame: twice the light, so half is neutral; the frame's alpha stays
	gl.BlendFuncSeparate(gl.DST_COLOR, gl.SRC_COLOR, gl.ZERO, gl.ONE)
	gl.BindVertexArray(layer.emptyArray)
	gl.DrawArrays(gl.TRIANGLES, 0, verticesPerQuad)
	gl.BindVertexArray(0)
	gl.Disable(gl.BLEND)
	gl.DepthMask(true)
	gl.Enable(gl.DEPTH_TEST)
}

// addLights clears the picture of light to the ambient colour and adds the lighting's lights to it.
func (layer *lightLayer) addLights(lighting overlay.Lighting, width, height int, picturesSize int) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, layer.target.framebuffer)
	gl.Viewport(0, 0, layer.target.width, layer.target.height)
	ambient := graphics.AmbientLight(lighting.Ambient)
	gl.ClearColor(ambient[0], ambient[1], ambient[2], 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	if lighting.Last <= lighting.First {
		return
	}
	program := layer.addProgram
	gl.UseProgram(program.handle)
	gl.Uniform2f(program.location("screenSize"), float32(width), float32(height))
	gl.Uniform2f(program.location("picturesSize"), float32(picturesSize), float32(picturesSize))
	program.setInteger("pictures", picturesTextureUnit)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)
	gl.BindVertexArray(layer.vertexArray)
	gl.DrawArrays(gl.TRIANGLES, int32(lighting.First*verticesPerQuad), int32((lighting.Last-lighting.First)*verticesPerQuad))
	gl.BindVertexArray(0)
	gl.Disable(gl.BLEND)
}

// prepare makes the picture of light for a framebuffer of this size, when it changed.
func (layer *lightLayer) prepare(width, height int) {
	const lightResolution = 2 // screen pixels a side to one of the light's
	lightWidth, lightHeight := int32(max(width/lightResolution, 1)), int32(max(height/lightResolution, 1))
	if layer.target.width == lightWidth && layer.target.height == lightHeight {
		return
	}
	deleteTarget(layer.target)
	layer.target = newTarget(lightWidth, lightHeight)
}
