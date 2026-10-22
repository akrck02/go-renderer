package gles

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/internal/shading"
	"github.com/akrck02/go-renderer/opengl/overlay"
)

// lightTextureUnit is where the picture of light is bound while it multiplies the screen.
const lightTextureUnit = 3

// floatsPerLightVertex: screen position 2, corner 2, colour 4, picture position 2, inner 1.
const floatsPerLightVertex = 11

// The lights' attributes past the three every program reads: on only while the lights draw.
const (
	lightPictureAttribute = 3
	lightInnerAttribute   = 4
)

// lightLayer draws the overlay's 2D lights (see overlay.Lighting) as the desktop's OpenGL does:
// it adds the lights into a picture of light at half the screen's resolution, cleared to the
// ambient colour, and multiplies the screen by it. Its texture is made once, and again only when
// the screen changes size.
type lightLayer struct {
	functions       Functions
	addProgram      *program
	multiplyProgram *program
	vertexBuffer    Buffer
	corners         Buffer // two triangles from 0 to 1, stretched over the area lit
	target          target
	vertices        []float32
}

func newLightLayer(functions Functions) (*lightLayer, error) {
	addProgram, err := newProgram(functions, lightVertexShader, lightFragmentShader, lightAttributes)
	if err != nil {
		return nil, err
	}
	multiplyProgram, err := newProgram(functions, lightAreaVertexShader, lightMultiplyShader, backdropAttributes)
	if err != nil {
		return nil, err
	}
	layer := &lightLayer{functions: functions, addProgram: addProgram, multiplyProgram: multiplyProgram,
		vertexBuffer: functions.CreateBuffer(), corners: functions.CreateBuffer()}
	functions.BindBuffer(ArrayBuffer, layer.corners)
	functions.BufferData(ArrayBuffer, floatBytes(cornerFloats), StaticDraw)
	return layer, nil
}

// fillVertices turns every light of the frame into two triangles, for the lightings to draw in turn.
func (layer *lightLayer) fillVertices(lights []overlay.LightQuad) {
	vertices := layer.vertices[:0]
	for _, light := range lights {
		picture := [4]float32{-1, -1, -1, -1}
		if light.Pictured {
			picture = light.Picture
		}
		for _, corner := range quadCorners {
			x, y := corner[0], corner[1]
			vertices = append(vertices, light.Screen[x], light.Screen[y+1], float32(x-1), float32(y-1),
				light.Color[0], light.Color[1], light.Color[2], light.Color[3], picture[x], picture[y+1], light.Inner)
		}
	}
	layer.vertices = vertices
	if len(vertices) == 0 {
		return
	}
	layer.functions.BindBuffer(ArrayBuffer, layer.vertexBuffer)
	layer.functions.BufferData(ArrayBuffer, floatBytes(vertices), StreamDraw)
}

// light lights what the target framebuffer (width × height pixels) shows with one lighting. The
// target is bound again when it returns, with blending off; the caller sets its own program and
// buffer up.
func (layer *lightLayer) light(lighting overlay.Lighting, target Framebuffer, width, height, picturesSize int) {
	functions := layer.functions
	layer.prepare(width, height)
	functions.Disable(DepthTest)
	layer.addLights(lighting, width, height, picturesSize)

	functions.BindFramebuffer(FramebufferTarget, target)
	functions.Viewport(0, 0, width, height)
	layer.multiplyProgram.use()
	functions.ActiveTexture(Texture0 + lightTextureUnit)
	functions.BindTexture(Texture2D, layer.target.texture)
	layer.multiplyProgram.setInteger("light", lightTextureUnit)
	layer.multiplyProgram.setVector4("area", graphics.Vec4(lighting.Area))
	layer.multiplyProgram.setVector2("screenSize", float32(width), float32(height))
	functions.Enable(Blend)
	// screen × light + light × screen: twice the light, so half is neutral; the screen's alpha stays
	functions.BlendFuncSeparate(DestinationColor, SourceColor, Zero, One)
	functions.BindBuffer(ArrayBuffer, layer.corners)
	functions.VertexAttribPointer(0, 2, Float, false, 2*4, 0)
	functions.DrawArrays(Triangles, 0, len(cornerFloats)/2)
	functions.Disable(Blend)
}

// addLights clears the picture of light to the ambient colour and adds the lighting's lights to it.
func (layer *lightLayer) addLights(lighting overlay.Lighting, width, height, picturesSize int) {
	functions := layer.functions
	functions.BindFramebuffer(FramebufferTarget, layer.target.framebuffer)
	functions.Viewport(0, 0, layer.target.width, layer.target.height)
	ambient := shading.AmbientLight(lighting.Ambient)
	functions.ClearColor(ambient[0], ambient[1], ambient[2], 1)
	functions.Clear(ColorBufferBit)
	if lighting.Last <= lighting.First {
		return
	}
	program := layer.addProgram
	program.use()
	program.setVector2("screenSize", float32(width), float32(height))
	program.setVector2("picturesSize", float32(picturesSize), float32(picturesSize))
	program.setInteger("pictures", picturesTextureUnit)
	functions.Enable(Blend)
	functions.BlendFuncSeparate(One, One, One, One)
	functions.BindBuffer(ArrayBuffer, layer.vertexBuffer)
	layer.describeVertexLayout()
	functions.DrawArrays(Triangles, lighting.First*cornersPerQuad, (lighting.Last-lighting.First)*cornersPerQuad)
	functions.DisableVertexAttribArray(lightPictureAttribute)
	functions.DisableVertexAttribArray(lightInnerAttribute)
	functions.Disable(Blend)
}

// describeVertexLayout tells where each attribute is in the bound light vertices.
func (layer *lightLayer) describeVertexLayout() {
	functions := layer.functions
	stride := floatsPerLightVertex * 4
	functions.VertexAttribPointer(0, 2, Float, false, stride, 0)
	functions.VertexAttribPointer(1, 2, Float, false, stride, 2*4)
	functions.VertexAttribPointer(2, 4, Float, false, stride, 4*4)
	functions.EnableVertexAttribArray(lightPictureAttribute)
	functions.VertexAttribPointer(lightPictureAttribute, 2, Float, false, stride, 8*4)
	functions.EnableVertexAttribArray(lightInnerAttribute)
	functions.VertexAttribPointer(lightInnerAttribute, 1, Float, false, stride, 10*4)
}

// prepare makes the picture of light for a screen of this size, when it changed.
func (layer *lightLayer) prepare(width, height int) {
	const lightResolution = 2 // screen pixels a side to one of the light's
	lightWidth, lightHeight := max(width/lightResolution, 1), max(height/lightResolution, 1)
	if layer.target.width == lightWidth && layer.target.height == lightHeight {
		return
	}
	if layer.target.texture != 0 {
		layer.functions.DeleteFramebuffer(layer.target.framebuffer)
		layer.functions.DeleteTexture(layer.target.texture)
	}
	layer.target = newTarget(layer.functions, lightWidth, lightHeight, RGBA)
}
