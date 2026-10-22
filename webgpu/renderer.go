package webgpu

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
)

type WebGPURenderer struct{}

func (WebGPURenderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error {
	return NotImplemented
}

func (WebGPURenderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error {
	return NotImplemented
}

func (WebGPURenderer) RenderRectangle(space graphics.CoordinateSpace, coordinates graphics.Vector, width float32, height float32, shader *string, fill bool) error {
	return NotImplemented
}

func (WebGPURenderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, coordinates graphics.Vector, width float32, height float32, shader *string) error {
	return NotImplemented
}

func (WebGPURenderer) Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vector, shader *string) error {
	return NotImplemented
}

func (WebGPURenderer) Render3dModel(space graphics.CoordinateSpace, model *models.Model, shader *string) error {
	return NotImplemented
}

func (WebGPURenderer) SetMatrices(projection, view, model graphics.Matrix) {}
