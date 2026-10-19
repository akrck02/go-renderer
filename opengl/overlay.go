package opengl

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/sceneplan"
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
	picturesTexture           uint32
	picturesSize              int // the side of the pictures on the GPU; 0 before any
	vertices                  []float32
}

// floatsPerOverlayVertex: screen position 2, atlas position 2, colour 4, picture position 2.
const floatsPerOverlayVertex = 10

// overlayTextureUnit and picturesTextureUnit are free of the scene's textures (shadows, seabed,
// level variation, sky).
const (
	overlayTextureUnit  = 8
	picturesTextureUnit = 9
)

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
	gl.GenTextures(1, &renderer.picturesTexture)
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
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointerWithOffset(3, 2, gl.FLOAT, false, stride, 8*bytesPerFloat)
	gl.BindVertexArray(0)
}

// draw lays out and paints the overlay in screen space: no depth test, alpha blending, so it
// covers the scene whatever is in front.
func (renderer *overlayRenderer) draw(world *scene.Scene, matrices sceneplan.CameraMatrices, width, height int, scale float32) {
	if world.Overlay.IsEmpty() {
		return
	}
	frame := overlay.Frame{ViewProjection: matrices.ViewProjection, Width: width, Height: height,
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
	renderer.uploadPictures()
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

// uploadPictures sends the images' colours to the GPU: the whole picture atlas when it is new,
// grew or was emptied, else only the areas pictures were copied into since the last frame.
func (renderer *overlayRenderer) uploadPictures() {
	pictures := renderer.painter.Pictures()
	size := pictures.Size()
	if size == 0 {
		return // no picture drawn yet
	}
	gl.ActiveTexture(gl.TEXTURE0 + picturesTextureUnit)
	gl.BindTexture(gl.TEXTURE_2D, renderer.picturesTexture)
	whole, areas := pictures.Pending()
	if whole || size != renderer.picturesSize {
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(size), int32(size), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pictures.Image.Pix))
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		renderer.picturesSize = size
	} else {
		// the rows are read from the whole image: the row length skips the rest of each row
		gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(size))
		for _, area := range areas {
			start := pictures.Image.PixOffset(area.Min.X, area.Min.Y)
			gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(area.Min.X), int32(area.Min.Y), int32(area.Dx()), int32(area.Dy()),
				gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pictures.Image.Pix[start:]))
		}
		gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	}
	pictures.Uploaded()
}

// fillVertices turns every quad into two triangles: position, atlas position, colour and picture
// position (-1 for a quad that shows no picture).
func (renderer *overlayRenderer) fillVertices(quads []overlay.Quad) {
	vertices := renderer.vertices[:0]
	corners := [6][2]int{{0, 0}, {0, 2}, {2, 2}, {0, 0}, {2, 2}, {2, 0}} // edges: 0 left or top, 2 right or bottom
	for _, quad := range quads {
		picture := [4]float32{-1, -1, -1, -1}
		if quad.Pictured {
			picture = quad.Picture
		}
		if quad.Pictured && quad.Sharp {
			picture[0] += graphics.SharpPictureOffset
			picture[2] += graphics.SharpPictureOffset
		}
		for _, corner := range corners {
			x, y := corner[0], corner[1]
			vertices = append(vertices, quad.Screen[x], quad.Screen[y+1], quad.Atlas[x], quad.Atlas[y+1],
				quad.Color[0], quad.Color[1], quad.Color[2], quad.Color[3], picture[x], picture[y+1])
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
	gl.Uniform2f(program.location("picturesSize"), float32(renderer.picturesSize), float32(renderer.picturesSize))
	program.setInteger("pictures", picturesTextureUnit)
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
