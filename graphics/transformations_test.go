package graphics

import (
	"testing"
)

func TestIdentity(t *testing.T) {
	id := Identity()
	if id[0] != 1 || id[5] != 1 || id[10] != 1 || id[15] != 1 {
		t.Errorf("Identity matrix incorrect: %v", id)
	}
}

func TestNormalize(t *testing.T) {
	v := Vec4{1, 0, 0, 0}
	n := v.Normalize()
	if n[0] != 1 {
		t.Errorf("Normalize failed: %v", n)
	}

	v2 := Vec4{0, 0, 0, 0}
	n2 := v2.Normalize()
	if n2[0] != 0 || n2[1] != 0 || n2[2] != 0 {
		t.Errorf("Normalize of zero vector should be zero: %v", n2)
	}
}
