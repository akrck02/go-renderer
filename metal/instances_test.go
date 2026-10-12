package metal

import (
	"testing"
	"unsafe"

	"github.com/akrck02/go-renderer/opengl/overlay"
)

func TestAnInstanceIsThirtySixBytes(t *testing.T) {
	if size := unsafe.Sizeof(instance{}); size != 36 {
		t.Fatalf("an instance is %d bytes, the shader reads 36", size)
	}
}

func TestPackingKeepsPlacesAndRoundsColours(t *testing.T) {
	quads := []overlay.Quad{{
		Screen: [4]float32{10, 20, 30, 40},
		Atlas:  [4]float32{1.5, 2.5, 3.5, 4.5},
		Color:  [4]float32{1, 0.5, 0, 1.2},
	}}
	packed := packInstances(quads, nil)
	if len(packed) != 1 {
		t.Fatalf("packed %d instances, want 1", len(packed))
	}
	got := packed[0]
	if got.screen != quads[0].Screen || got.atlas != quads[0].Atlas {
		t.Errorf("places changed: %+v", got)
	}
	if got.color != [4]uint8{255, 128, 0, 255} {
		t.Errorf("colour %v, want [255 128 0 255]", got.color)
	}
}

func TestPackingReusesMemory(t *testing.T) {
	buffer := make([]instance, 0, 8)
	packed := packInstances(make([]overlay.Quad, 3), buffer)
	if len(packed) != 3 || &packed[0] != &buffer[:1][0] {
		t.Error("packing did not reuse the buffer it was given")
	}
}
