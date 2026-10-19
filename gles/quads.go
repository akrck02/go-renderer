package gles

import (
	"image"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
)

// floatsPerQuadVertex is a corner of a rectangle: screen position 2, atlas position 2, colour 4,
// picture position 2.
const floatsPerQuadVertex = 10

// atlasTextureUnit and picturesTextureUnit are where the overlay's two atlases are bound; the
// scene uses no textures.
const (
	atlasTextureUnit    = 0
	picturesTextureUnit = 1
)

// pictureAttribute is the attribute only the quads read: it is on while they draw, and off for
// the scene's program, which reads three.
const pictureAttribute = 3

// quadPass paints the overlay (labels, header, notices and a draw2d list) as rectangles of one
// atlas, in one draw call. It is the OpenGL overlay renderer's last steps, for OpenGL ES: the
// painter that lays out the rectangles and rasterises the atlas is shared with every backend.
//
// OpenGL ES 2 cannot draw instances, so each rectangle is two triangles of six corners, as on the
// desktop's OpenGL path.
type quadPass struct {
	functions    Functions
	painter      *overlay.Painter
	program      *program
	vertexBuffer Buffer
	// attributesOn is whether the vertex attributes were switched on: they stay on in the
	// context, and both programs read the same three
	attributesOn bool
	atlasTexture Texture
	atlasVersion int // the version of the atlas on the GPU; -1 before the first upload
	// picturesTexture holds the images' colours; picturesSize is its side on the GPU
	picturesTexture Texture
	picturesSize    int
	areaPixels      []byte // a changed area's rows, one after another, for uploading
	vertices        []float32
}

func newQuadPass(functions Functions) (*quadPass, error) {
	painter, err := overlay.NewPainter()
	if err != nil {
		return nil, err
	}
	shaders, err := newProgram(functions, quadVertexShader, quadFragmentShader, quadAttributes)
	if err != nil {
		return nil, err
	}
	pass := &quadPass{functions: functions, painter: painter, program: shaders, atlasVersion: -1}
	pass.vertexBuffer = functions.CreateBuffer()
	pass.atlasTexture = functions.CreateTexture()
	pass.picturesTexture = functions.CreateTexture()
	return pass, nil
}

// describeVertexLayout tells where each attribute is in the bound vertex buffer. With no vertex
// array objects in OpenGL ES 2, it is done before every draw.
func (pass *quadPass) describeVertexLayout() {
	functions := pass.functions
	if !pass.attributesOn {
		switchAttributesOn(functions)
		pass.attributesOn = true
	}
	stride := floatsPerQuadVertex * 4
	functions.VertexAttribPointer(0, 2, Float, false, stride, 0)
	functions.VertexAttribPointer(1, 2, Float, false, stride, 2*4)
	functions.VertexAttribPointer(2, 4, Float, false, stride, 4*4)
	functions.EnableVertexAttribArray(pictureAttribute)
	functions.VertexAttribPointer(pictureAttribute, 2, Float, false, stride, 8*4)
}

// switchAttributesOn turns on the three vertex attributes both programs read; they stay on.
func switchAttributesOn(functions Functions) {
	for index := range uint32(3) {
		functions.EnableVertexAttribArray(index)
	}
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
	pass.uploadPictures()
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
	functions.TexImage2D(Texture2D, 0, Luminance, size, size, Luminance, UnsignedByte, atlas.Image.Pix)
	functions.PixelStorei(UnpackAlignment, 4)
	functions.TexParameteri(Texture2D, TextureMinFilter, int32(Linear))
	functions.TexParameteri(Texture2D, TextureMagFilter, int32(Linear))
	functions.TexParameteri(Texture2D, TextureWrapS, int32(ClampToEdge))
	functions.TexParameteri(Texture2D, TextureWrapT, int32(ClampToEdge))
	pass.atlasVersion = atlas.Version
}

// uploadPictures sends the images' colours to the GPU: the whole picture atlas when it is new,
// grew or was emptied, else only the areas pictures were copied into since the last frame.
func (pass *quadPass) uploadPictures() {
	functions := pass.functions
	pictures := pass.painter.Pictures()
	if pictures.Size() == 0 {
		return // no picture drawn yet
	}
	functions.ActiveTexture(Texture0 + picturesTextureUnit)
	functions.BindTexture(Texture2D, pass.picturesTexture)
	whole, areas := pictures.Pending()
	size := pictures.Size()
	if whole || size != pass.picturesSize {
		functions.TexImage2D(Texture2D, 0, RGBA, size, size, RGBA, UnsignedByte, pictures.Image.Pix)
		functions.TexParameteri(Texture2D, TextureMinFilter, int32(Linear))
		functions.TexParameteri(Texture2D, TextureMagFilter, int32(Linear))
		functions.TexParameteri(Texture2D, TextureWrapS, int32(ClampToEdge))
		functions.TexParameteri(Texture2D, TextureWrapT, int32(ClampToEdge))
		pass.picturesSize = size
	} else {
		for _, area := range areas {
			pass.areaPixels = areaRows(pass.areaPixels[:0], pictures.Image, area)
			functions.TexSubImage2D(Texture2D, 0, area.Min.X, area.Min.Y, area.Dx(), area.Dy(), RGBA, UnsignedByte, pass.areaPixels)
		}
	}
	pictures.Uploaded()
	functions.ActiveTexture(Texture0 + atlasTextureUnit)
}

// areaRows copies an area of an image's pixels row after row, as OpenGL ES 2 reads them (it
// cannot skip the rest of each row).
func areaRows(into []byte, picture *image.NRGBA, area image.Rectangle) []byte {
	for y := area.Min.Y; y < area.Max.Y; y++ {
		start := picture.PixOffset(area.Min.X, y)
		into = append(into, picture.Pix[start:start+area.Dx()*4]...)
	}
	return into
}

// fillVertices turns every rectangle into two triangles and sends them to the GPU.
func (pass *quadPass) fillVertices(quads []overlay.Quad) {
	pass.vertices = appendQuadVertices(pass.vertices[:0], quads)
	pass.functions.BindBuffer(ArrayBuffer, pass.vertexBuffer)
	pass.functions.BufferData(ArrayBuffer, floatBytes(pass.vertices), StreamDraw)
}

// appendQuadVertices writes six corners per rectangle: position, atlas position, colour and
// picture position (-1 for a quad that shows no picture).
func appendQuadVertices(vertices []float32, quads []overlay.Quad) []float32 {
	for _, quad := range quads {
		picture := [4]float32{-1, -1, -1, -1}
		if quad.Pictured {
			picture = quad.Picture
		}
		if quad.Pictured && quad.Sharp {
			picture[0] += graphics.SharpPictureOffset
			picture[2] += graphics.SharpPictureOffset
		}
		for _, corner := range quadCorners {
			x, y := corner[0], corner[1] // 0 for the left or top edge, 2 for the right or bottom
			vertices = append(vertices, quad.Screen[x], quad.Screen[y+1], quad.Atlas[x], quad.Atlas[y+1],
				quad.Color[0], quad.Color[1], quad.Color[2], quad.Color[3], picture[x], picture[y+1])
		}
	}
	return vertices
}

// quadCorners are a rectangle's two triangles: for each corner, which edges it is on (0 left or
// top, 2 right or bottom), into a quad's left, top, right, bottom.
var quadCorners = [6][2]int{{0, 0}, {0, 2}, {2, 2}, {0, 0}, {2, 2}, {2, 0}}

func (pass *quadPass) drawVertices(width, height int) {
	functions := pass.functions
	pass.program.use()
	pass.program.setVector2("screenSize", float32(width), float32(height))
	size := float32(pass.painter.Atlas().Size())
	pass.program.setVector2("atlasSize", size, size)
	pass.program.setInteger("atlas", atlasTextureUnit)
	pass.program.setVector2("picturesSize", float32(pass.picturesSize), float32(pass.picturesSize))
	pass.program.setInteger("pictures", picturesTextureUnit)
	functions.Disable(DepthTest)
	functions.DepthMask(false)
	functions.Enable(Blend)
	functions.BlendFuncSeparate(SourceAlpha, OneMinusSrcAlpha, One, OneMinusSrcAlpha)
	functions.BindBuffer(ArrayBuffer, pass.vertexBuffer)
	pass.describeVertexLayout()
	functions.DrawArrays(Triangles, 0, len(pass.vertices)/floatsPerQuadVertex)
	functions.DisableVertexAttribArray(pictureAttribute)
	functions.Disable(Blend)
	functions.DepthMask(true)
}
