package opengl

import (
	"unsafe"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/immediate"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// Sizes OpenGL is told about the immediate-mode vertices: each is a graphics.Vector of four
// float32, of which the shader reads x, y and z.
const (
	vectorBytes     = int(unsafe.Sizeof(graphics.Vector{}))
	indexBytes      = int(unsafe.Sizeof(uint32(0)))
	positionValues  = 3
	positionChannel = 0 // the shader's layout(location = 0)
)

// rectangleIndices joins a rectangle's four corners (top right, bottom right, bottom left, top
// left) into two triangles.
var rectangleIndices = [6]uint32{0, 1, 3, 1, 2, 3}

// white is the colour the immediate-mode shapes are drawn in.
var white = graphics.Vector{1, 1, 1, 1}

// Renderer draws immediate-mode shapes (immediate.Renderer) with OpenGL, in the window's current
// program. Its zero value is ready; it makes its buffers on the first shape and reuses them, so a
// shape allocates nothing on the Go side.
type Renderer struct {
	projection, view, model graphics.Matrix
	vertexArray             uint32
	vertexBuffer            uint32
	elementBuffer           uint32
	program                 int32 // the program the locations were looked up in
	currentProgram          int32 // filled by OpenGL each shape; a field so the pointer does not escape
	locations               uniformLocations
	corners                 [4]graphics.Vector
}

// uniformLocations are where the immediate-mode shader takes its matrices and colour.
type uniformLocations struct {
	projection, view, model, color int32
}

// SetMatrices sets the matrices the next shapes are drawn with.
func (renderer *Renderer) SetMatrices(projection, view, model graphics.Matrix) {
	renderer.projection = projection
	renderer.view = view
	renderer.model = model
}

// RenderPolygon returns NotImplemented: OpenGL cannot draw polygons yet.
func (renderer *Renderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error {
	return NotImplemented
}

// RenderTriangle draws triangles, three coordinates each.
func (renderer *Renderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error {
	renderer.drawVertices(coordinates)
	return nil
}

// RenderRectangle draws a rectangle of width × height centred on centre, in the plane z = centre's z.
func (renderer *Renderer) RenderRectangle(space graphics.CoordinateSpace, centre graphics.Vector, width float32, height float32, shader *string, fill bool) error {
	halfWidth, halfHeight := width/2, height/2
	renderer.corners = [4]graphics.Vector{
		{centre[0] + halfWidth, centre[1] + halfHeight, centre[2], 0}, // top right
		{centre[0] + halfWidth, centre[1] - halfHeight, centre[2], 0}, // bottom right
		{centre[0] - halfWidth, centre[1] - halfHeight, centre[2], 0}, // bottom left
		{centre[0] - halfWidth, centre[1] + halfHeight, centre[2], 0}, // top left
	}
	renderer.drawIndexed(renderer.corners[:], rectangleIndices[:])
	return nil
}

// RenderImage returns NotImplemented: OpenGL cannot draw pictures in immediate mode yet.
func (renderer *Renderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, centre graphics.Vector, width float32, height float32, shader *string) error {
	return NotImplemented
}

// RenderObject draws the triangles of a 3D object, three vertices each.
func (renderer *Renderer) RenderObject(space graphics.CoordinateSpace, vertices []graphics.Vector, shader *string) error {
	renderer.drawVertices(vertices)
	return nil
}

// RenderModel draws every mesh of the model.
func (renderer *Renderer) RenderModel(space graphics.CoordinateSpace, model *immediate.Model, shader *string) error {
	for _, mesh := range model.Meshes {
		if len(mesh.Indices) == 0 {
			renderer.drawVertices(mesh.Vertices)
			continue
		}
		renderer.drawIndexed(mesh.Vertices, mesh.Indices)
	}
	return nil
}

// drawVertices draws vertices as triangles, three each.
func (renderer *Renderer) drawVertices(vertices []graphics.Vector) {
	if len(vertices) == 0 {
		return
	}
	renderer.prepare(vertices)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(vertices)))
}

// drawIndexed draws triangles joining vertices, three indices each.
func (renderer *Renderer) drawIndexed(vertices []graphics.Vector, indices []uint32) {
	if len(vertices) == 0 || len(indices) == 0 {
		return
	}
	renderer.prepare(vertices)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, renderer.elementBuffer)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*indexBytes, unsafe.Pointer(&indices[0]), gl.STREAM_DRAW)
	gl.DrawElements(gl.TRIANGLES, int32(len(indices)), gl.UNSIGNED_INT, nil)
}

// prepare makes the buffers on first use, uploads the vertices and sets the uniforms.
func (renderer *Renderer) prepare(vertices []graphics.Vector) {
	if renderer.vertexArray == 0 {
		gl.GenVertexArrays(1, &renderer.vertexArray)
		gl.GenBuffers(1, &renderer.vertexBuffer)
		gl.GenBuffers(1, &renderer.elementBuffer)
		gl.BindVertexArray(renderer.vertexArray)
		gl.BindBuffer(gl.ARRAY_BUFFER, renderer.vertexBuffer)
		gl.EnableVertexAttribArray(positionChannel)
		gl.VertexAttribPointer(positionChannel, positionValues, gl.FLOAT, false, int32(vectorBytes), nil)
	}
	gl.BindVertexArray(renderer.vertexArray)
	gl.BindBuffer(gl.ARRAY_BUFFER, renderer.vertexBuffer)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*vectorBytes, unsafe.Pointer(&vertices[0]), gl.STREAM_DRAW)
	renderer.setUniforms()
}

// setUniforms gives the current program the matrices and the colour, looking their locations up
// again only when the program changed.
func (renderer *Renderer) setUniforms() {
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &renderer.currentProgram)
	current := renderer.currentProgram
	if current != renderer.program {
		renderer.program = current
		renderer.locations = lookUpUniforms(uint32(current))
	}
	setMatrixUniform(renderer.locations.projection, &renderer.projection)
	setMatrixUniform(renderer.locations.view, &renderer.view)
	setMatrixUniform(renderer.locations.model, &renderer.model)
	if renderer.locations.color != -1 {
		gl.Uniform4fv(renderer.locations.color, 1, &white[0])
	}
}

// lookUpUniforms finds the immediate-mode uniforms in program; -1 for one it does not use.
func lookUpUniforms(program uint32) uniformLocations {
	return uniformLocations{
		projection: gl.GetUniformLocation(program, gl.Str("projection\x00")),
		view:       gl.GetUniformLocation(program, gl.Str("view\x00")),
		model:      gl.GetUniformLocation(program, gl.Str("model\x00")),
		color:      gl.GetUniformLocation(program, gl.Str("color\x00")),
	}
}

// setMatrixUniform gives a matrix to the uniform at location, unless the program has none (-1).
func setMatrixUniform(location int32, matrix *graphics.Matrix) {
	if location != -1 {
		gl.UniformMatrix4fv(location, 1, false, &matrix[0])
	}
}
