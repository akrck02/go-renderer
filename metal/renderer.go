package metal

import (
	"errors"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
)

type MetalRenderer struct{}

func (MetalRenderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error {
	return errors.New("Graphics API not implemented yet!")
}

func (MetalRenderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error {
	return errors.New("Graphics API not implemented yet!")
}

func (MetalRenderer) RenderRectangle(space graphics.CoordinateSpace, coordinates graphics.Vec4, width float32, height float32, shader *string, fill bool) error {
	return errors.New("Graphics API not implemented yet!")
}

func (MetalRenderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, coordinates graphics.Vec4, width float32, height float32, shader *string) error {
	return errors.New("Graphics API not implemented yet!")
}

func (MetalRenderer) Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vec4, shader *string) error {
	return errors.New("Graphics API not implemented yet!")
}

func (MetalRenderer) Render3dModel(space graphics.CoordinateSpace, model *models.Model, shader *string) error {
	return errors.New("Graphics API not implemented yet!")
}

func (MetalRenderer) SetMatrices(projection, view, model graphics.Mat4) {}
