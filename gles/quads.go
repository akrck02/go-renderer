package gles

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
)

// floatsPerQuadVertex is a corner of a rectangle: screen position 2, atlas position 2, colour 4.
const floatsPerQuadVertex = 8

// atlasTextureUnit is where the overlay's atlas is bound; the scene uses no textures.
const atlasTextureUnit = 0

// quadPass paints the overlay (labels, header, notices and a draw2d list) as rectangles of one
// atlas, in one draw call. It is the OpenGL overlay renderer's last steps, for OpenGL ES: the
// painter that lays out the rectangles and rasterises the atlas is shared with every backend.
//
// OpenGL ES through golang.org/x/mobile/gl cannot draw instances, so each rectangle is two
// triangles of six corners, as on the desktop's OpenGL path.
type quadPass struct {
	functions    Functions
	painter      *overlay.Painter
	program      *program
	vertexArray  VertexArray
	vertexBuffer Buffer
	atlasTexture Texture
	atlasVersion int // the version of the atlas on the GPU; -1 before the first upload
	vertices     []float32
}

func newQuadPass(functions Functions) (*quadPass, error) {
	painter, err := overlay.NewPainter()
	if err != nil {
		return nil, err
	}
	shaders, err := newProgram(functions, quadVertexShader, quadFragmentShader)
	if err != nil {
		return nil, err
	}
	pass := &quadPass{functions: functions, painter: painter, program: shaders, atlasVersion: -1}
	pass.vertexArray = functions.CreateVertexArray()
	pass.vertexBuffer = functions.CreateBuffer()
	pass.atlasTexture = functions.CreateTexture()
	pass.describeVertexLayout()
	return pass, nil
}

func (pass *quadPass) describeVertexLayout() {
	functions := pass.functions
	functions.BindVertexArray(pass.vertexArray)
	functions.BindBuffer(ArrayBuffer, pass.vertexBuffer)
	stride := floatsPerQuadVertex * 4
	functions.EnableVertexAttribArray(0)
	functions.VertexAttribPointer(0, 2, Float, false, stride, 0)
	functions.EnableVertexAttribArray(1)
	functions.VertexAttribPointer(1, 2, Float, false, stride, 2*4)
	functions.EnableVertexAttribArray(2)
	functions.VertexAttribPointer(2, 4, Float, false, stride, 4*4)
	functions.BindVertexArray(0)
}

// paintFlat paints an overlay with nothing in the world to anchor labels to: a flat interface.
func (pass *quadPass) paintFlat(layer *scene.Overlay, width, height int, scale float32) {
	if layer.IsEmpty() {
		return
	}
	quads := pass.painter.Paint(layer, nil, overlay.Frame{Width: width, Height: height, Scale: scale})
	pass.drawQuads(quads, width, height)
}

// paintScene paints a scene's overlay, its labels following their nodes.
func (pass *quadPass) paintScene(world *scene.Scene, viewProjection graphics.Mat4, width, height int, scale float32) {
	if world.Overlay.IsEmpty() {
		return
	}
	frame := overlay.Frame{ViewProjection: viewProjection, Width: width, Height: height, Scale: scale,
		VerticalScale: world.Environment.EffectiveVerticalScale()}
	quads := pass.painter.Paint(&world.Overlay, world.WorldTransforms(anchoredNodes(world.Overlay.Labels)), frame)
	pass.drawQuads(quads, width, height)
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

func (pass *quadPass) drawQuads(quads []overlay.Quad, width, height int) {
	if len(quads) == 0 {
		return
	}
	pass.uploadAtlas()
	pass.fillVertices(quads)
	pass.drawVertices(width, height)
}

// uploadAtlas sends the atlas to the GPU when glyphs, icons or corners were added to it.
func (pass *quadPass) uploadAtlas() {
	functions := pass.functions
	atlas := pass.painter.Atlas()
	functions.ActiveTexture(Texture0 + atlasTextureUnit)
	functions.BindTexture(Texture2D, pass.atlasTexture)
	if atlas.Version == pass.atlasVersion {
		return
	}
	size := atlas.Size()
	functions.PixelStorei(UnpackAlignment, 1) // rows of a one-byte-per-pixel image are not padded
	functions.TexImage2D(Texture2D, 0, R8, size, size, Red, UnsignedByte, atlas.Image.Pix)
	functions.PixelStorei(UnpackAlignment, 4)
	functions.TexParameteri(Texture2D, TextureMinFilter, int32(Linear))
	functions.TexParameteri(Texture2D, TextureMagFilter, int32(Linear))
	functions.TexParameteri(Texture2D, TextureWrapS, int32(ClampToEdge))
	functions.TexParameteri(Texture2D, TextureWrapT, int32(ClampToEdge))
	pass.atlasVersion = atlas.Version
}

// fillVertices turns every rectangle into two triangles and sends them to the GPU.
func (pass *quadPass) fillVertices(quads []overlay.Quad) {
	pass.vertices = appendQuadVertices(pass.vertices[:0], quads)
	pass.functions.BindBuffer(ArrayBuffer, pass.vertexBuffer)
	pass.functions.BufferData(ArrayBuffer, floatBytes(pass.vertices), StreamDraw)
}

// appendQuadVertices writes six corners per rectangle: position, atlas position and colour.
func appendQuadVertices(vertices []float32, quads []overlay.Quad) []float32 {
	for _, quad := range quads {
		left, top, right, bottom := quad.Screen[0], quad.Screen[1], quad.Screen[2], quad.Screen[3]
		atlasLeft, atlasTop, atlasRight, atlasBottom := quad.Atlas[0], quad.Atlas[1], quad.Atlas[2], quad.Atlas[3]
		corners := [6][4]float32{
			{left, top, atlasLeft, atlasTop}, {left, bottom, atlasLeft, atlasBottom}, {right, bottom, atlasRight, atlasBottom},
			{left, top, atlasLeft, atlasTop}, {right, bottom, atlasRight, atlasBottom}, {right, top, atlasRight, atlasTop},
		}
		for _, corner := range corners {
			vertices = append(vertices, corner[0], corner[1], corner[2], corner[3],
				quad.Color[0], quad.Color[1], quad.Color[2], quad.Color[3])
		}
	}
	return vertices
}

func (pass *quadPass) drawVertices(width, height int) {
	functions := pass.functions
	pass.program.use()
	pass.program.setVector2("screenSize", float32(width), float32(height))
	size := float32(pass.painter.Atlas().Size())
	pass.program.setVector2("atlasSize", size, size)
	pass.program.setInteger("atlas", atlasTextureUnit)
	functions.Disable(DepthTest)
	functions.DepthMask(false)
	functions.Enable(Blend)
	functions.BlendFuncSeparate(SourceAlpha, OneMinusSrcAlpha, One, OneMinusSrcAlpha)
	functions.BindVertexArray(pass.vertexArray)
	functions.DrawArrays(Triangles, 0, len(pass.vertices)/floatsPerQuadVertex)
	functions.BindVertexArray(0)
	functions.Disable(Blend)
	functions.DepthMask(true)
}
