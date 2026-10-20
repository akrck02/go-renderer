package ctr

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/sceneplan"
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
	baked := bakeVertices(nil, mesh, bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindLit, light: upwardLight()})
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
	baked := bakeVertices(nil, mesh, bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindLit, light: upwardLight()})
	want := uint32(math.Round(math.Pow(0.1, 1/2.2) * 255))
	if got := bakedColour(baked, 0); got[0] != want {
		t.Fatalf("red %d, want %d", got[0], want)
	}
}

func TestAnUnlitSurfaceTakesTheBrightness(t *testing.T) {
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0}, Normals: []float32{0, -1, 0}, Colors: []float32{1, 0, 0, 0.5}}
	baked := bakeVertices(nil, mesh, bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindUnlit, light: upwardLight()})
	want := uint32(math.Round(math.Pow(0.5, 1/2.2) * 255))
	if got := bakedColour(baked, 0); got != [4]uint32{want, 0, 0, 128} {
		t.Fatalf("colour %v, want %d 0 0 128", got, want)
	}
}

func TestATurnedNodeTurnsItsLight(t *testing.T) {
	// A half turn around x: the normal that pointed up points down.
	upsideDown := [9]float32{1, 0, 0, 0, -1, 0, 0, 0, -1}
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0}, Normals: []float32{0, 1, 0}}
	baked := bakeVertices(nil, mesh, bakeKey{normalMatrix: upsideDown, baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindLit, light: upwardLight()})
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

func TestTheGammaTableMatchesThePower(t *testing.T) {
	// Away from black, where the curve is steepest, the table's steps are finer than a byte.
	worst := 0
	for value := float32(0.01); value <= 1; value += 0.001 {
		want := int(math.Round(math.Pow(float64(value), 1/2.2) * 255))
		got := int(screenByte(value))
		if difference := got - want; difference > worst || -difference > worst {
			worst = max(difference, -difference)
		}
	}
	if worst > 1 {
		t.Fatalf("the table is %d steps of 255 away from the power", worst)
	}
}

func TestBakingAgainReusesTheBuffer(t *testing.T) {
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}, Normals: []float32{0, 1, 0, 0, 1, 0, 0, 1, 0}}
	key := bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindLit, light: upwardLight()}
	first := bakeVertices(nil, mesh, key)
	again := bakeVertices(first, mesh, key)
	if &again[0] != &first[0] {
		t.Fatal("a second bake of the same size should write into the same buffer")
	}
}

// lampAbove is a white lamp at intensity 1, one unit over the origin, reaching three.
func lampAbove() lampKey {
	return lampKey{count: 1, model: graphics.Identity(), lights: [4]sceneplan.FrameLight{
		{Position: graphics.Vec4{0, 1, 0, 1}, Color: graphics.Vec4{1, 1, 1, 1}, Inner: 0.5, Outer: 3},
	}}
}

func TestALampLightsTheVerticesItReaches(t *testing.T) {
	// two vertices facing up: one under the lamp, one out of its reach
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0, 10, 0, 0}, Normals: []float32{0, 1, 0, 0, 1, 0}}
	key := bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{0.5, 0.5, 0.5, 1}, kind: scene.KindLit, light: upwardLight()}
	unlit := bakeVertices(nil, mesh, key)
	key.lamps = lampAbove()
	lit := bakeVertices(nil, mesh, key)
	if under, before := bakedColour(lit, 0)[0], bakedColour(unlit, 0)[0]; under <= before+20 {
		t.Errorf("under the lamp red went from %d to %d; wanted it much brighter", before, under)
	}
	if far, before := bakedColour(lit, 1)[0], bakedColour(unlit, 1)[0]; far != before {
		t.Errorf("out of the lamp's reach red went from %d to %d", before, far)
	}
}

func TestAMovingLampBakesAgainAndAStillOneDoesNot(t *testing.T) {
	still, moved := lampAbove(), lampAbove()
	if !still.closeTo(lampAbove()) {
		t.Error("the same lamp would bake again")
	}
	moved.lights[0].Position[0] += 0.5
	if still.closeTo(moved) {
		t.Error("a lamp half a unit away would not bake again")
	}
	if still.closeTo(lampKey{}) {
		t.Error("a lamp switched off would not bake again")
	}
}
