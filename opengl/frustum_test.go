package opengl

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
)

func TestFrustumKeepsWhatTheCameraSees(t *testing.T) {
	projection := graphics.PerspectiveOpenGL(0.8, 1, 0.1, 100)
	view := graphics.LookAt(graphics.Vec4{0, 0, 0, 1}, graphics.Vec4{0, 0, -1, 1}, graphics.Vec4{0, 1, 0, 0})
	volume := frustumFromMatrix(projection.Multiply(view))
	ahead := box{graphics.Vec4{-1, -1, -11, 1}, graphics.Vec4{1, 1, -9, 1}}
	behind := box{graphics.Vec4{-1, -1, 9, 1}, graphics.Vec4{1, 1, 11, 1}}
	beyond := box{graphics.Vec4{-1, -1, -300, 1}, graphics.Vec4{1, 1, -200, 1}}
	aside := box{graphics.Vec4{50, -1, -11, 1}, graphics.Vec4{52, 1, -9, 1}}
	if !volume.containsBox(ahead) {
		t.Errorf("a box ahead should be visible")
	}
	for name, hidden := range map[string]box{"behind": behind, "beyond the far plane": beyond, "aside": aside} {
		if volume.containsBox(hidden) {
			t.Errorf("a box %s should be culled", name)
		}
	}
}

func TestBoxDistanceAndTransform(t *testing.T) {
	unit := box{graphics.Vec4{0, 0, 0, 1}, graphics.Vec4{1, 1, 1, 1}}
	if unit.distanceTo(graphics.Vec4{0.5, 0.5, 0.5, 1}) != 0 || unit.distanceTo(graphics.Vec4{4, 0.5, 0.5, 1}) != 3 {
		t.Errorf("wrong distance to box")
	}
	moved := unit.transformed(graphics.Translate(graphics.Vec4{10, 2, 0, 0}), 2)
	if moved.minimum[0] != 10 || moved.minimum[1] != 4 || moved.maximum[1] != 6 {
		t.Errorf("wrong transformed box %+v", moved)
	}
}
