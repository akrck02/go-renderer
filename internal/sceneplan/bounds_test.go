package sceneplan

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
)

func TestFrustumKeepsWhatTheCameraSees(test *testing.T) {
	projection := graphics.PerspectiveOpenGL(0.8, 1, 0.1, 100)
	view := graphics.LookAt(graphics.Vector{0, 0, 0, 1}, graphics.Vector{0, 0, -1, 1}, graphics.Vector{0, 1, 0, 0})
	volume := frustumFromMatrix(projection.Multiply(view))
	ahead := box{graphics.Vector{-1, -1, -11, 1}, graphics.Vector{1, 1, -9, 1}}
	behind := box{graphics.Vector{-1, -1, 9, 1}, graphics.Vector{1, 1, 11, 1}}
	beyond := box{graphics.Vector{-1, -1, -300, 1}, graphics.Vector{1, 1, -200, 1}}
	aside := box{graphics.Vector{50, -1, -11, 1}, graphics.Vector{52, 1, -9, 1}}
	if !volume.containsBox(ahead) {
		test.Errorf("a box ahead should be visible")
	}
	for name, hidden := range map[string]box{"behind": behind, "beyond the far plane": beyond, "aside": aside} {
		if volume.containsBox(hidden) {
			test.Errorf("a box %s should be culled", name)
		}
	}
}

func TestBoxDistanceAndTransform(test *testing.T) {
	unit := box{graphics.Vector{0, 0, 0, 1}, graphics.Vector{1, 1, 1, 1}}
	if unit.distanceTo(graphics.Vector{0.5, 0.5, 0.5, 1}) != 0 || unit.distanceTo(graphics.Vector{4, 0.5, 0.5, 1}) != 3 {
		test.Errorf("wrong distance to box")
	}
	moved := unit.transformed(graphics.Translate(graphics.Vector{10, 2, 0, 0}), 2)
	if moved.minimum[0] != 10 || moved.minimum[1] != 4 || moved.maximum[1] != 6 {
		test.Errorf("wrong transformed box %+v", moved)
	}
}
