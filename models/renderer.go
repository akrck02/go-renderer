package models

import "github.com/akrck02/go-renderer/graphics"

type Mesh struct {
	Vertices []graphics.Vec4
	Indices  []uint32
}

type Model struct {
	Meshes []Mesh
}

type Renderer interface {
	RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error
	RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error
	RenderRectangle(space graphics.CoordinateSpace, coordinates graphics.Vec4, width float32, height float32, shader *string, fill bool) error
	RenderImage(space graphics.CoordinateSpace, bytes []byte, coordinates graphics.Vec4, width float32, height float32, shader *string) error
	Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vec4, shader *string) error
	Render3dModel(space graphics.CoordinateSpace, model *Model, shader *string) error
	SetMatrices(projection, view, model graphics.Mat4)
}
