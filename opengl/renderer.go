package opengl

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"os"
	"strings"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/go-gl/gl/v4.1-core/gl"
)

type OpenGLRenderer struct {
	projection graphics.Mat4
	view       graphics.Mat4
	model      graphics.Mat4
}

func (r OpenGLRenderer) SetMatrices(projection, view, model graphics.Mat4) {
	r.projection = projection
	r.view = view
	r.model = model
}

func (r OpenGLRenderer) RenderPolygon(
	space graphics.CoordinateSpace,
	coordinates []graphics.Vec4,
	shader *string,
	fill bool,
) error {
	return nil
}

func (r OpenGLRenderer) RenderTriangle(
	space graphics.CoordinateSpace,
	coordinates []graphics.Vec4,
	shader *string,
	fill bool,
) error {

	setupUniforms(r.projection, r.view, r.model, graphics.Vec4{1, 1, 1, 1})

	// makeVao initializes and returns a vertex array from the points provided.
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(coordinates)*16, gl.Ptr(coordinates), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 16, nil)

	gl.BindVertexArray(vao)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(coordinates)))

	// Cleanup to avoid memory leaks
	gl.DeleteBuffers(1, &vbo)
	gl.DeleteVertexArrays(1, &vao)

	return nil
}

func (r OpenGLRenderer) RenderRectangle(
	space graphics.CoordinateSpace,
	coordinates graphics.Vec4,
	width float32,
	height float32,
	shader *string,
	fill bool,
) error {

	setupUniforms(r.projection, r.view, r.model, graphics.Vec4{1, 1, 1, 1})

	halfWidth := width / 2
	halfHeight := height / 2

	vertices := []graphics.Vec4{
		{coordinates[0] + halfWidth, coordinates[1] + halfHeight, coordinates[2], 0}, // top right
		{coordinates[0] + halfWidth, coordinates[1] - halfHeight, coordinates[2], 0}, // bottom right
		{coordinates[0] - halfWidth, coordinates[1] - halfHeight, coordinates[2], 0}, // bottom left
		{coordinates[0] - halfWidth, coordinates[1] + halfHeight, coordinates[2], 0}, // top left
	}

	indices := []uint32{ // using uint32 for consistency with gl.UNSIGNED_INT
		0, 1, 3, // first triangle
		1, 2, 3, // second triangle
	}

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*16, gl.Ptr(vertices), gl.STATIC_DRAW)

	var ebo uint32
	gl.GenBuffers(1, &ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)

	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 16, nil)
	gl.EnableVertexAttribArray(0)

	gl.BindVertexArray(vao)
	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, nil)

	// Cleanup to avoid memory leaks
	gl.DeleteBuffers(1, &vbo)
	gl.DeleteBuffers(1, &ebo)
	gl.DeleteVertexArrays(1, &vao)

	return nil
}

func (r OpenGLRenderer) RenderImage(
	space graphics.CoordinateSpace,
	bytes []byte,
	coordinates graphics.Vec4,
	width float32,
	height float32,
	shader *string,
) error {
	return nil
}

func (r OpenGLRenderer) Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vec4, shader *string) error {

	setupUniforms(r.projection, r.view, r.model, graphics.Vec4{1, 1, 1, 1})

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*16, gl.Ptr(vertices), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 16, nil)

	gl.BindVertexArray(vao)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(vertices)))

	// Cleanup
	gl.DeleteBuffers(1, &vbo)
	gl.DeleteVertexArrays(1, &vao)

	return nil
}

func (r OpenGLRenderer) Render3dModel(space graphics.CoordinateSpace, model *models.Model, shader *string) error {

	setupUniforms(r.projection, r.view, r.model, graphics.Vec4{1, 1, 1, 1})

	for _, mesh := range model.Meshes {

		var vbo uint32
		gl.GenBuffers(1, &vbo)
		gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
		gl.BufferData(gl.ARRAY_BUFFER, len(mesh.Vertices)*16, gl.Ptr(mesh.Vertices), gl.STATIC_DRAW)

		var vao uint32
		gl.GenVertexArrays(1, &vao)
		gl.BindVertexArray(vao)

		var ebo uint32
		gl.GenBuffers(1, &ebo)
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(mesh.Indices)*4, gl.Ptr(mesh.Indices), gl.STATIC_DRAW)

		gl.EnableVertexAttribArray(0)
		gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
		gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 16, nil)

		gl.BindVertexArray(vao)
		gl.DrawElements(gl.TRIANGLES, int32(len(mesh.Indices)), gl.UNSIGNED_INT, nil)

		// Cleanup
		gl.DeleteBuffers(1, &vbo)
		gl.DeleteBuffers(1, &ebo)
		gl.DeleteVertexArrays(1, &vao)
	}

	return nil
}

func setupUniforms(projection, view, model graphics.Mat4, color graphics.Vec4) {
	var program int32
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &program)

	projLoc := gl.GetUniformLocation(uint32(program), gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projLoc, 1, false, &projection[0])

	viewLoc := gl.GetUniformLocation(uint32(program), gl.Str("view\x00"))
	gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])

	modelLoc := gl.GetUniformLocation(uint32(program), gl.Str("model\x00"))
	gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])

	colorLoc := gl.GetUniformLocation(uint32(program), gl.Str("color\x00"))
	gl.Uniform4fv(colorLoc, 1, &color[0])
}

func CompileShader(
	source string,
	shaderType uint32,
) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func newTexture(file string) uint32 {
	imgFile, err := os.Open(file)
	if err != nil {
		log.Fatalf("texture %q not found on disk: %v\n", file, err)
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		panic(err)
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		panic("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.Enable(gl.TEXTURE_2D)
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return texture
}
