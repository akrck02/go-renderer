package vulkan

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/immediate"
)

// Renderer will draw the immediate-mode shapes (immediate.Renderer) with Vulkan. Until it is
// built, every method returns NotImplemented.
type Renderer struct{}

// RenderPolygon returns NotImplemented.
func (Renderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error {
	return NotImplemented
}

// RenderTriangle returns NotImplemented.
func (Renderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vector, shader *string, fill bool) error {
	return NotImplemented
}

// RenderRectangle returns NotImplemented.
func (Renderer) RenderRectangle(space graphics.CoordinateSpace, centre graphics.Vector, width float32, height float32, shader *string, fill bool) error {
	return NotImplemented
}

// RenderImage returns NotImplemented.
func (Renderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, centre graphics.Vector, width float32, height float32, shader *string) error {
	return NotImplemented
}

// RenderObject returns NotImplemented.
func (Renderer) RenderObject(space graphics.CoordinateSpace, vertices []graphics.Vector, shader *string) error {
	return NotImplemented
}

// RenderModel returns NotImplemented.
func (Renderer) RenderModel(space graphics.CoordinateSpace, model *immediate.Model, shader *string) error {
	return NotImplemented
}

// SetMatrices does nothing yet: it has no error to return.
func (Renderer) SetMatrices(projection, view, model graphics.Matrix) {}
