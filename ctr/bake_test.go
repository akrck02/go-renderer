package ctr

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

func upwardLight() lightKey {
	return lightKey{
		sunDirection: graphics.Vec4{0, 1, 0, 0}, sunColour: graphics.Vec4{0.5, 0.5, 0.5, 0},
		ambientGround: graphics.Vec4{0.1, 0.1, 0.1, 0}, ambientSpan: graphics.Vec4{0.2, 0.2, 0.2, 0},
		brightness: 0.5, verticalScale: 1,
	}
}

func identityNormals() [9]float32 {
	return [9]float32{1, 0, 0, 0, 1, 0, 0, 0, 1}
}

func bakedColour(baked []float32, vertex int) [4]uint32 {
	bits := math.Float32bits(baked[vertex*floatsPerBakedVertex+3])
	return [4]uint32{bits & 0xFF, bits >> 8 & 0xFF, bits >> 16 & 0xFF, bits >> 24}
}

func TestALitSurfaceFacingTheSunGetsSkyAndSun(t *testing.T) {
	// Facing up: sky and ground light 0.1 + 0.2, sun 0.5, on white: 0.8, to the screen's gamma.
	mesh := &scene.Mesh{Positions: []float32{1, 2, 3}, Normals: []float32{0, 1, 0}}
	baked := bakeVertices(mesh, bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindLit, light: upwardLight()})
	if baked[0] != 1 || baked[1] != 2 || baked[2] != 3 {
		t.Fatalf("position %v, want 1 2 3", baked[0:3])
	}
	want := uint32(math.Round(math.Pow(0.8, 1/2.2) * 255))
	if got := bakedColour(baked, 0); got != [4]uint32{want, want, want, 255} {
		t.Fatalf("colour %v, want %d %d %d 255", got, want, want, want)
	}
}

func TestASurfaceFacingDownGetsOnlyTheGroundLight(t *testing.T) {
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0}, Normals: []float32{0, -1, 0}}
	baked := bakeVertices(mesh, bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindLit, light: upwardLight()})
	want := uint32(math.Round(math.Pow(0.1, 1/2.2) * 255))
	if got := bakedColour(baked, 0); got[0] != want {
		t.Fatalf("red %d, want %d", got[0], want)
	}
}

func TestAnUnlitSurfaceTakesTheBrightness(t *testing.T) {
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0}, Normals: []float32{0, -1, 0}, Colors: []float32{1, 0, 0, 0.5}}
	baked := bakeVertices(mesh, bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindUnlit, light: upwardLight()})
	want := uint32(math.Round(math.Pow(0.5, 1/2.2) * 255))
	if got := bakedColour(baked, 0); got != [4]uint32{want, 0, 0, 128} {
		t.Fatalf("colour %v, want %d 0 0 128", got, want)
	}
}

func TestATurnedNodeTurnsItsLight(t *testing.T) {
	// A half turn around x: the normal that pointed up points down.
	upsideDown := [9]float32{1, 0, 0, 0, -1, 0, 0, 0, -1}
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0}, Normals: []float32{0, 1, 0}}
	baked := bakeVertices(mesh, bakeKey{normalMatrix: upsideDown, baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindLit, light: upwardLight()})
	want := uint32(math.Round(math.Pow(0.1, 1/2.2) * 255))
	if got := bakedColour(baked, 0); got[0] != want {
		t.Fatalf("red %d, want the ground light only, %d", got[0], want)
	}
}

func TestASlowlyTurningSunDoesNotBakeAgain(t *testing.T) {
	light := upwardLight()
	nudged := light
	nudged.sunDirection[0] += 0.001
	if !light.closeTo(nudged) {
		t.Fatal("a turn of a thousandth should not bake again")
	}
	nudged.sunDirection[0] += 0.01
	if light.closeTo(nudged) {
		t.Fatal("a turn of a hundredth should bake again")
	}
}
