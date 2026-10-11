package vulkan

import (
	"errors"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
)

type VulkanRenderer struct{}

func (VulkanRenderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error {
	return errors.New("Graphics API not implemented yet!")
}

func (VulkanRenderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error {
	return errors.New("Graphics API not implemented yet!")
}

func (VulkanRenderer) RenderRectangle(space graphics.CoordinateSpace, coordinates graphics.Vec4, width float32, height float32, shader *string, fill bool) error {
	return errors.New("Graphics API not implemented yet!")
}

func (VulkanRenderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, coordinates graphics.Vec4, width float32, height float32, shader *string) error {
	return errors.New("Graphics API not implemented yet!")
}

func (VulkanRenderer) Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vec4, shader *string) error {
	return errors.New("Graphics API not implemented yet!")
}

func (VulkanRenderer) Render3dModel(space graphics.CoordinateSpace, model *models.Model, shader *string) error {
	return errors.New("Graphics API not implemented yet!")
}

func (VulkanRenderer) SetMatrices(projection, view, model graphics.Mat4) {}
