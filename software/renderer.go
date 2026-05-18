package software

import (
	"errors"

	"github.com/akrck02/go-renderer/graphics"
)

type SoftwareRenderer struct{}

func (SoftwareRenderer) RenderPolygon(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error {
	return errors.New("Graphics API not implemented yet!")
}

func (SoftwareRenderer) RenderTriangle(space graphics.CoordinateSpace, coordinates []graphics.Vec4, shader *string, fill bool) error {
	return errors.New("Graphics API not implemented yet!")
}

func (SoftwareRenderer) RenderRectangle(space graphics.CoordinateSpace, coordinates graphics.Vec4, width float32, height float32, shader *string, fill bool) error {
	return errors.New("Graphics API not implemented yet!")
}

func (SoftwareRenderer) RenderImage(space graphics.CoordinateSpace, bytes []byte, coordinates graphics.Vec4, width float32, height float32, shader *string) error {
	return errors.New("Graphics API not implemented yet!")
}

func (SoftwareRenderer) Render3dObject(space graphics.CoordinateSpace, vertices []graphics.Vec4, shader *string) error {
	return errors.New("Graphics API not implemented yet!")
}
