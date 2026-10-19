package ctr

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/opengl/overlay"
)

func TestADiscIsFullInsideAndDarkAtItsEdge(t *testing.T) {
	pixels := lightDiscs()
	const half = discSide / 2
	for _, level := range []int{0, discLevels / 2, discLevels - 1} {
		originX, originY := (level%discsPerRow)*discSide, (level/discsPerRow)*discSide
		at := func(x, y int) byte { return pixels[(originY+y)*discTextureSide+originX+x] }
		if centre := at(half, half); centre < 250 {
			t.Errorf("disc %d: the centre is %d, wanted full", level, centre)
		}
		for _, edge := range [][2]int{{0, half}, {discSide - 1, half}, {half, 0}, {half, discSide - 1}, {0, 0}} {
			if value := at(edge[0], edge[1]); value != 0 {
				t.Errorf("disc %d: the edge at %v is %d, wanted dark so neighbours do not bleed", level, edge, value)
			}
		}
	}
}

func TestADiscWithALargerInnerShareStaysFullFurther(t *testing.T) {
	pixels := lightDiscs()
	const half = discSide / 2
	halfway := func(level int) byte {
		originX, originY := (level%discsPerRow)*discSide, (level/discsPerRow)*discSide
		return pixels[(originY+half)*discTextureSide+originX+half+half/2]
	}
	if soft, hard := halfway(0), halfway(discLevels-1); soft >= hard {
		t.Fatalf("halfway out the softest disc is %d and the hardest %d; wanted the hardest brighter", soft, hard)
	}
}

func TestALightTakesTheNearestDisc(t *testing.T) {
	cases := []struct {
		inner float32
		disc  int
	}{{0, 0}, {1, discLevels - 1}, {0.5, 8}, {-1, 0}, {2, discLevels - 1}}
	for _, testCase := range cases {
		if disc := discFor(testCase.inner); disc != testCase.disc {
			t.Errorf("inner %v: disc %d, wanted %d", testCase.inner, disc, testCase.disc)
		}
	}
}

func TestLightVerticesKeepTheLightsAtHalfAndEndWithTheScreen(t *testing.T) {
	lights := []overlay.LightQuad{
		{Screen: [4]float32{10, 10, 50, 50}, Color: [4]float32{1, 0.5, 0, 1}, Inner: 0},
		{Screen: [4]float32{60, 10, 100, 50}, Color: [4]float32{1, 1, 1, 2}, Picture: [4]float32{0, 0, 32, 32}, Pictured: true},
	}
	vertices := make([]float32, (len(lights)+1)*4*floatsPerQuadVertex)
	runs := writeLightVertices(vertices, nil, lights, 64, 400, 240)
	if len(runs) != 2 || runs[0].pictured || !runs[1].pictured {
		t.Fatalf("runs %+v, wanted a disc run then a picture run", runs)
	}
	if red, green := vertices[4], vertices[5]; red != 0.5 || green != 0.25 {
		t.Errorf("the first light's colour is %v, %v; wanted it at half", red, green)
	}
	second := vertices[4*floatsPerQuadVertex:]
	if second[4] != 1 || second[8] != 0 || second[2*floatsPerQuadVertex+8] != 0.5 {
		t.Errorf("the sprite light is %v; wanted twice the intensity at half, and its picture's place", second[:3*floatsPerQuadVertex])
	}
	last := vertices[2*4*floatsPerQuadVertex:]
	// the third corner is the bottom right: the screen's, and how much of the light canvas it covers
	bottomRight := last[2*floatsPerQuadVertex:]
	if bottomRight[0] != 400 || bottomRight[1] != 240 || bottomRight[2] != 400.0/512 || bottomRight[3] != 240.0/256 {
		t.Errorf("the screen's rectangle ends at %v", bottomRight[:4])
	}
}

func TestTheAmbientIsClearedAtHalf(t *testing.T) {
	if colour := ambientColour([4]float32{1, 1, 1, 1}); colour != 0x808080ff {
		t.Errorf("white daylight clears to %08x, wanted 808080ff", colour)
	}
	if colour := ambientColour([4]float32{1, 0, 0, 2}); colour != 0xff0000ff {
		t.Errorf("a red at twice the intensity clears to %08x, wanted ff0000ff", colour)
	}
}

func TestOrthographicPutsTheLightCanvasCornersOnTheClipCorners(t *testing.T) {
	projection := Orthographic(0, lightCanvasWidth, lightCanvasHeight, 0, 0, 1)
	cases := []struct {
		point        graphics.Vec4
		clipX, clipY float32
		name         string
	}{
		{graphics.Vec4{0, 0, 0, 1}, -1, 1, "top left"},
		{graphics.Vec4{lightCanvasWidth, lightCanvasHeight, 0, 1}, 1, -1, "bottom right"},
	}
	for _, testCase := range cases {
		clip := projection.Apply(testCase.point)
		if !near(clip[0], testCase.clipX) || !near(clip[1], testCase.clipY) || !near(clip[2], -1) {
			t.Errorf("%s: clip %v, want (%v, %v, -1)", testCase.name, clip, testCase.clipX, testCase.clipY)
		}
	}
}
