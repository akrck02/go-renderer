package ctr

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/internal/sceneplan"
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

func TestALitSurfaceFacingTheSunGetsSkyAndSun(test *testing.T) {
	// Facing up: sky and ground light 0.1 + 0.2, sun 0.5, on white: 0.8, to the screen's gamma.
	mesh := &scene.Mesh{Positions: []float32{1, 2, 3}, Normals: []float32{0, 1, 0}}
	baked := bakeVertices(nil, mesh, bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindLit, light: upwardLight()})
	if baked[0] != 1 || baked[1] != 2 || baked[2] != 3 {
		test.Fatalf("position %v, want 1 2 3", baked[0:3])
	}
	want := uint32(math.Round(math.Pow(0.8, 1/2.2) * 255))
	if got := bakedColour(baked, 0); got != [4]uint32{want, want, want, 255} {
		test.Fatalf("colour %v, want %d %d %d 255", got, want, want, want)
	}
}

func TestASurfaceFacingDownGetsOnlyTheGroundLight(test *testing.T) {
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0}, Normals: []float32{0, -1, 0}}
	baked := bakeVertices(nil, mesh, bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindLit, light: upwardLight()})
	want := uint32(math.Round(math.Pow(0.1, 1/2.2) * 255))
	if got := bakedColour(baked, 0); got[0] != want {
		test.Fatalf("red %d, want %d", got[0], want)
	}
}

func TestAnUnlitSurfaceTakesTheBrightness(test *testing.T) {
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0}, Normals: []float32{0, -1, 0}, Colors: []float32{1, 0, 0, 0.5}}
	baked := bakeVertices(nil, mesh, bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindUnlit, light: upwardLight()})
	want := uint32(math.Round(math.Pow(0.5, 1/2.2) * 255))
	if got := bakedColour(baked, 0); got != [4]uint32{want, 0, 0, 128} {
		test.Fatalf("colour %v, want %d 0 0 128", got, want)
	}
}

func TestATurnedNodeTurnsItsLight(test *testing.T) {
	// A half turn around x: the normal that pointed up points down.
	upsideDown := [9]float32{1, 0, 0, 0, -1, 0, 0, 0, -1}
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0}, Normals: []float32{0, 1, 0}}
	baked := bakeVertices(nil, mesh, bakeKey{normalMatrix: upsideDown, baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindLit, light: upwardLight()})
	want := uint32(math.Round(math.Pow(0.1, 1/2.2) * 255))
	if got := bakedColour(baked, 0); got[0] != want {
		test.Fatalf("red %d, want the ground light only, %d", got[0], want)
	}
}

func TestASlowlyTurningSunDoesNotBakeAgain(test *testing.T) {
	light := upwardLight()
	nudged := light
	nudged.sunDirection[0] += 0.001
	if !light.closeTo(nudged) {
		test.Fatal("a turn of a thousandth should not bake again")
	}
	nudged.sunDirection[0] += 0.01
	if light.closeTo(nudged) {
		test.Fatal("a turn of a hundredth should bake again")
	}
}

func TestTheGammaTableMatchesThePower(test *testing.T) {
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
		test.Fatalf("the table is %d steps of 255 away from the power", worst)
	}
}

func TestBakingAgainReusesTheBuffer(test *testing.T) {
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0, 1, 0, 0, 0, 1, 0}, Normals: []float32{0, 1, 0, 0, 1, 0, 0, 1, 0}}
	key := bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{1, 1, 1, 1}, kind: scene.KindLit, light: upwardLight()}
	first := bakeVertices(nil, mesh, key)
	again := bakeVertices(first, mesh, key)
	if &again[0] != &first[0] {
		test.Fatal("a second bake of the same size should write into the same buffer")
	}
}

// lampAbove is a white lamp at intensity 1, one unit over the origin, reaching three.
func lampAbove() lampKey {
	return lampKey{count: 1, model: graphics.Identity(), lights: [4]sceneplan.FrameLight{
		{Position: graphics.Vec4{0, 1, 0, 1}, Color: graphics.Vec4{1, 1, 1, 1}, Inner: 0.5, Outer: 3},
	}}
}

func TestALampLightsTheVerticesItReaches(test *testing.T) {
	// two vertices facing up: one under the lamp, one out of its reach
	mesh := &scene.Mesh{Positions: []float32{0, 0, 0, 10, 0, 0}, Normals: []float32{0, 1, 0, 0, 1, 0}}
	key := bakeKey{normalMatrix: identityNormals(), baseColour: graphics.Vec4{0.5, 0.5, 0.5, 1}, kind: scene.KindLit, light: upwardLight()}
	unlit := bakeVertices(nil, mesh, key)
	key.lamps = lampAbove()
	lit := bakeVertices(nil, mesh, key)
	if under, before := bakedColour(lit, 0)[0], bakedColour(unlit, 0)[0]; under <= before+20 {
		test.Errorf("under the lamp red went from %d to %d; wanted it much brighter", before, under)
	}
	if far, before := bakedColour(lit, 1)[0], bakedColour(unlit, 1)[0]; far != before {
		test.Errorf("out of the lamp's reach red went from %d to %d", before, far)
	}
}

func TestAMovingLampBakesAgainAndAStillOneDoesNot(test *testing.T) {
	still, moved := lampAbove(), lampAbove()
	if !still.closeTo(lampAbove()) {
		test.Error("the same lamp would bake again")
	}
	moved.lights[0].Position[0] += 0.5
	if still.closeTo(moved) {
		test.Error("a lamp half a unit away would not bake again")
	}
	if still.closeTo(lampKey{}) {
		test.Error("a lamp switched off would not bake again")
	}
}

// shaderLampLight is scene.v.pica's lamp light, step by step, from the uniforms it reads.
func shaderLampLight(uniforms lampUniforms, world, normal graphics.Vec4) graphics.Vec4 {
	var total graphics.Vec4
	for slot := range uniforms.positions {
		position, colour := uniforms.positions[slot], uniforms.colours[slot]
		towards := graphics.Vec4{position[0] - world[0], position[1] - world[1], position[2] - world[2], 0}
		distance := float32(math.Sqrt(float64(max(towards.Dot(towards), 1e-5))))
		facing := max(normal.Dot(towards)/distance, 0)
		share := min(max((distance-position[3])*colour[3], 0), 1)
		fade := 1 - share*share*(3-2*share)
		for channel := range 3 {
			total[channel] += colour[channel] * facing * fade
		}
	}
	return total
}

func TestAMovingNodeIsLitOnTheGPUAsABakedOneIs(test *testing.T) {
	lamps := lampAbove()
	lamps.lights[1] = sceneplan.FrameLight{Position: graphics.Vec4{2, 0.5, 0, 1}, Color: graphics.Vec4{0, 0.5, 1, 1}, Inner: 0, Outer: 4}
	lamps.count = 2
	frameLights := lamps.lights[:2]
	uniforms := lampUniformsOf(sceneplan.DrawLights{Indices: [4]uint16{0, 1}, Count: 2}, frameLights)
	up := graphics.Vec4{0, 1, 0, 0}
	for _, position := range [][]float32{{0, 0, 0}, {1, 0, 0}, {2.5, 0, 0}, {10, 0, 0}} {
		baked := lamps.lightAt(position, up, 1)
		shaded := shaderLampLight(uniforms, graphics.Vec4{position[0], position[1], position[2], 1}, up)
		for channel := range 3 {
			if math.Abs(float64(baked[channel]-shaded[channel])) > 1e-4 {
				test.Errorf("at %v the GPU's lamps give %v, the bake %v", position, shaded, baked)
				break
			}
		}
	}
	if uniforms.colours[2] != (graphics.Vec4{}) {
		test.Errorf("an unused lamp's colour is %v, wanted none", uniforms.colours[2])
	}
}
