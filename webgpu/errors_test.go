package webgpu

import (
	"errors"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
)

func TestEveryFunctionSaysItIsNotImplemented(test *testing.T) {
	renderer := WebGPURenderer{}
	backend := &WebGPU{}
	calls := map[string]error{
		"RenderPolygon":   renderer.RenderPolygon(graphics.CoordinateSpace(0), nil, nil, false),
		"RenderTriangle":  renderer.RenderTriangle(graphics.CoordinateSpace(0), nil, nil, false),
		"RenderRectangle": renderer.RenderRectangle(graphics.CoordinateSpace(0), graphics.Vec4{}, 1, 1, nil, false),
		"RenderImage":     renderer.RenderImage(graphics.CoordinateSpace(0), nil, graphics.Vec4{}, 1, 1, nil),
		"Render3dObject":  renderer.Render3dObject(graphics.CoordinateSpace(0), nil, nil),
		"Render3dModel":   renderer.Render3dModel(graphics.CoordinateSpace(0), nil, nil),
		"StartLoop":       backend.StartLoop(nil),
		"Draw":            backend.Draw(nil),
	}
	for name, err := range calls {
		if !errors.Is(err, NotImplemented) {
			test.Errorf("%s returned %v, not %s", name, err, NotImplemented.Name())
		}
	}
}
