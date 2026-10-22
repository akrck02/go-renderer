package vulkan

import (
	"errors"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
)

func TestEveryFunctionOfTheVulkanStubSaysItIsNotImplemented(test *testing.T) {
	renderer := Renderer{}
	backend := &Window{}
	calls := map[string]error{
		"RenderPolygon":   renderer.RenderPolygon(graphics.LocalSpace, nil, nil, false),
		"RenderTriangle":  renderer.RenderTriangle(graphics.LocalSpace, nil, nil, false),
		"RenderRectangle": renderer.RenderRectangle(graphics.LocalSpace, graphics.Vector{}, 1, 1, nil, false),
		"RenderImage":     renderer.RenderImage(graphics.LocalSpace, nil, graphics.Vector{}, 1, 1, nil),
		"RenderObject":    renderer.RenderObject(graphics.LocalSpace, nil, nil),
		"RenderModel":     renderer.RenderModel(graphics.LocalSpace, nil, nil),
		"Run":             backend.Run(nil),
		"Draw":            backend.Draw(nil),
	}
	for name, err := range calls {
		if !errors.Is(err, NotImplemented) {
			test.Errorf("%s returned %v, not %s", name, err, NotImplemented.Name())
		}
	}
}
