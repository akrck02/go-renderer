package metal

import (
	"testing"
	"unsafe"

	"github.com/akrck02/go-renderer/opengl/overlay"
)

func TestLightsHaveTheSizesTheObjectiveCSideReads(test *testing.T) {
	if size := unsafe.Sizeof(light{}); size != 64 {
		test.Errorf("a light is %d bytes; flat_darwin.h reads 64", size)
	}
	if size := unsafe.Sizeof(lighting{}); size != 48 {
		test.Errorf("a lighting is %d bytes; flat_darwin.h reads 48", size)
	}
}

func TestAPointLightReadsNoPictureAndALightingItsAmbientAtHalf(test *testing.T) {
	lights := packLights([]overlay.LightQuad{
		{Screen: [4]float32{0, 0, 10, 10}, Color: [4]float32{1, 1, 1, 2}, Inner: 0.5},
		{Screen: [4]float32{0, 0, 10, 10}, Picture: [4]float32{4, 4, 8, 8}, Pictured: true},
	}, nil)
	if lights[0].picture != noPicture || lights[0].inner != 0.5 || lights[1].picture[2] != 8 {
		test.Fatalf("the lights are %+v", lights)
	}
	lightings := packLightings([]overlay.Lighting{{At: 3, First: 0, Last: 2, Ambient: [4]float32{1, 1, 1, 1}, Area: [4]float32{0, 0, 64, 32}}}, nil)
	if lighting := lightings[0]; lighting.ambient != [4]float32{0.5, 0.5, 0.5, 1} || lighting.at != 3 || lighting.area[2] != 64 {
		test.Fatalf("the lighting is %+v", lighting)
	}
}
