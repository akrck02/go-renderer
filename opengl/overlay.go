package opengl

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// overlayRenderer draws the scene's overlay (labels, header, notices) over the finished frame:
// the overlay package lays it out as rectangles of one atlas, and this uploads the atlas and
// draws all the rectangles in a single call.
type overlayRenderer struct {
	painter                   *overlay.Painter
	program                   *shaderProgram
	vertexArray, vertexBuffer uint32
	atlasTexture              uint32
	atlasVersion              int // version of the atlas last uploaded
	vertices                  []float32
}

// floatsPerOverlayVertex: screen position 2, atlas position 2, colour 4.
const floatsPerOverlayVertex = 8

// overlayTextureUnit is free of the scene's textures (shadows, seabed, level variation, sky).
const overlayTextureUnit = 8

func newOverlayRenderer() (*overlayRenderer, error) {
	painter, err := overlay.NewPainter()
	if err != nil {
		return nil, err
	}
	program, err := newShaderProgram(graphics.OverlayVertexShader, graphics.OverlayFragmentShader)
	if err != nil {
		return nil, err
	}
	renderer := &overlayRenderer{painter: painter, program: program}
	gl.GenVertexArrays(1, &renderer.vertexArray)
	gl.GenBuffers(1, &renderer.vertexBuffer)
	gl.GenTextures(1, &renderer.atlasTexture)
	renderer.describeVertexLayout()
	return renderer, nil
}

func (renderer *overlayRenderer) describeVertexLayout() {
	gl.BindVertexArray(renderer.vertexArray)
	gl.BindBuffer(gl.ARRAY_BUFFER, renderer.vertexBuffer)
	stride := int32(floatsPerOverlayVertex * bytesPerFloat)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, stride, 0)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, stride, 2*bytesPerFloat)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 4, gl.FLOAT, false, stride, 4*bytesPerFloat)
	gl.BindVertexArray(0)
}

// draw lays out and paints the overlay in screen space: no depth test, alpha blending, so it
// covers the scene whatever is in front.
func (renderer *overlayRenderer) draw(world *scene.Scene, matrices cameraMatrices, width, height int, scale float32) {
	if world.Overlay.IsEmpty() {
		return
	}
	frame := overlay.Frame{ViewProjection: matrices.viewProjection, Width: width, Height: height,
		Scale: scale, VerticalScale: world.Environment.EffectiveVerticalScale()}
	quads := renderer.painter.Paint(&world.Overlay, world.WorldTransforms(anchoredNodes(world.Overlay.Labels)), frame)
	renderer.drawQuads(quads, width, height)
}

// drawFlat paints an overlay with nothing in the world to anchor labels to: a flat interface.
func (renderer *overlayRenderer) drawFlat(layer *scene.Overlay, width, height int, scale float32) {
	if layer.IsEmpty() {
		return
	}
	quads := renderer.painter.Paint(layer, nil, overlay.Frame{Width: width, Height: height, Scale: scale})
	renderer.drawQuads(quads, width, height)
}

func (renderer *overlayRenderer) drawQuads(quads []overlay.Quad, width, height int) {
	if len(quads) == 0 {
		return
	}
	renderer.uploadAtlas()
	renderer.fillVertices(quads)
	renderer.drawVertices(width, height)
	gl.ActiveTexture(gl.TEXTURE0) // the other passes bind their textures on the first unit
}

// anchoredNodes lists the nodes labels follow, so only their transforms are worked out.
func anchoredNodes(labels []*scene.Label) map[*scene.Node]bool {
	nodes := map[*scene.Node]bool{}
	for _, label := range labels {
		if label != nil && label.Node != nil && !label.Hidden {
			nodes[label.Node] = true
		}
	}
	return nodes
}

// uploadAtlas sends the atlas to the GPU when glyphs, icons or corners were added to it.
func (renderer *overlayRenderer) uploadAtlas() {
	atlas := renderer.painter.Atlas()
	gl.ActiveTexture(gl.TEXTURE0 + overlayTextureUnit)
	gl.BindTexture(gl.TEXTURE_2D, renderer.atlasTexture)
	if atlas.Version == renderer.atlasVersion {
		return
	}
	size := int32(atlas.Size())
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.R8, size, size, 0, gl.RED, gl.UNSIGNED_BYTE, gl.Ptr(atlas.Image.Pix))
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	renderer.atlasVersion = atlas.Version
}

// fillVertices turns every quad into two triangles.
func (renderer *overlayRenderer) fillVertices(quads []overlay.Quad) {
	vertices := renderer.vertices[:0]
	for _, quad := range quads {
		left, top, right, bottom := quad.Screen[0], quad.Screen[1], quad.Screen[2], quad.Screen[3]
		atlasLeft, atlasTop, atlasRight, atlasBottom := quad.Atlas[0], quad.Atlas[1], quad.Atlas[2], quad.Atlas[3]
		corners := [6][4]float32{
			{left, top, atlasLeft, atlasTop}, {left, bottom, atlasLeft, atlasBottom}, {right, bottom, atlasRight, atlasBottom},
			{left, top, atlasLeft, atlasTop}, {right, bottom, atlasRight, atlasBottom}, {right, top, atlasRight, atlasTop},
		}
		for _, corner := range corners {
			vertices = append(vertices, corner[0], corner[1], corner[2], corner[3], quad.Color[0], quad.Color[1], quad.Color[2], quad.Color[3])
		}
	}
	renderer.vertices = vertices
	gl.BindBuffer(gl.ARRAY_BUFFER, renderer.vertexBuffer)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*bytesPerFloat, gl.Ptr(vertices), gl.STREAM_DRAW)
}

func (renderer *overlayRenderer) drawVertices(width, height int) {
	program := renderer.program
	gl.UseProgram(program.handle)
	gl.Uniform2f(program.location("screenSize"), float32(width), float32(height))
	size := float32(renderer.painter.Atlas().Size())
	gl.Uniform2f(program.location("atlasSize"), size, size)
	program.setInteger("atlas", overlayTextureUnit)
	gl.Disable(gl.DEPTH_TEST)
	gl.DepthMask(false)
	gl.Enable(gl.BLEND)
	gl.BlendFuncSeparate(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA, gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
	gl.BindVertexArray(renderer.vertexArray)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(renderer.vertices)/floatsPerOverlayVertex))
	gl.BindVertexArray(0)
	gl.Disable(gl.BLEND)
	gl.DepthMask(true)
	gl.Enable(gl.DEPTH_TEST)
}
