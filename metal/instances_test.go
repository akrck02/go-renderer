package metal

import (
	"testing"
	"unsafe"

	"github.com/akrck02/go-renderer/internal/shading"
	"github.com/akrck02/go-renderer/opengl/overlay"
)

func TestAnInstanceIsFiftyTwoBytes(test *testing.T) {
	if size := unsafe.Sizeof(instance{}); size != 52 {
		test.Fatalf("an instance is %d bytes, the shader reads 52", size)
	}
}

func TestOnlyAPicturedQuadReadsThePictures(test *testing.T) {
	quads := []overlay.Quad{{}, {Picture: [4]float32{1, 2, 3, 4}, Pictured: true}}
	packed := packInstances(quads, nil, true)
	if packed[0].picture != noPicture || packed[1].picture != [4]float32{1, 2, 3, 4} {
		test.Fatalf("pictures %v and %v; want none for a plain quad, its place for a pictured one", packed[0].picture, packed[1].picture)
	}
}

func TestPackingKeepsPlacesAndRoundsColours(test *testing.T) {
	quads := []overlay.Quad{{
		Screen: [4]float32{10, 20, 30, 40},
		Atlas:  [4]float32{1.5, 2.5, 3.5, 4.5},
		Color:  [4]float32{1, 0.5, 0, 1.2},
	}}
	packed := packInstances(quads, nil, true)
	if len(packed) != 1 {
		test.Fatalf("packed %d instances, want 1", len(packed))
	}
	got := packed[0]
	if got.screen != quads[0].Screen || got.atlas != quads[0].Atlas {
		test.Errorf("places changed: %+v", got)
	}
	if got.color != [4]uint8{255, 128, 0, 255} {
		test.Errorf("colour %v, want [255 128 0 255]", got.color)
	}
}

func TestPackingReusesMemory(test *testing.T) {
	buffer := make([]instance, 0, 8)
	packed := packInstances(make([]overlay.Quad, 3), buffer, true)
	if len(packed) != 3 || &packed[0] != &buffer[:1][0] {
		test.Error("packing did not reuse the buffer it was given")
	}
}

func TestABackdropsQuadIsMarkedInItsPictureWithoutGrowing(test *testing.T) {
	quads := []overlay.Quad{{Screen: [4]float32{0, 0, 10, 10}}, {Screen: [4]float32{0, 0, 10, 10}, Backdrop: true}}
	packed := packInstances(quads, nil, true)
	if packed[0].picture != noPicture || packed[1].picture != backdropPicture {
		test.Fatalf("a backdrop's picture place is -2, a plain quad's -1: got %v and %v", packed[0].picture, packed[1].picture)
	}
	if unsafe.Sizeof(packed[0]) != 52 {
		test.Fatal("an instance stays 52 bytes")
	}
}

func TestARendererThatDoesNotBlurLeavesBackdropsOut(test *testing.T) {
	quads := []overlay.Quad{{}, {Backdrop: true}, {}}
	if packed := packInstances(quads, nil, false); len(packed) != 2 {
		test.Fatalf("over a 3D scene the backdrops are left out, got %d instances", len(packed))
	}
}

func TestASnapshotCarriesItsHalvingsAndWeights(test *testing.T) {
	packed := packSnapshots([]overlay.Snapshot{{At: 7, Blur: 24}}, nil)
	halvings := shading.BackdropHalvings(24)
	if len(packed) != 1 || packed[0].at != 7 || int(packed[0].halvings) != halvings {
		test.Fatalf("got %+v", packed)
	}
	if packed[0].weights != shading.BackdropWeights(24/float32(int(1)<<halvings)) {
		test.Fatal("the weights are for the deviation at the halved size")
	}
	if unsafe.Sizeof(packed[0]) != 28 {
		test.Fatal("a snapshot is 28 bytes, as flatSnapshot in flat_darwin.h")
	}
}
