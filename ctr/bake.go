package ctr

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/sceneplan"
)

// The PICA200 runs vertex shaders slowly enough that lighting every vertex every frame held the
// sample island to 30 frames a second on a New 3DS XL. Sun and sky change slowly, so the light is
// worked out here, once, when a mesh is uploaded, and again only when the light, the mesh or the
// node's turn changes. The GPU then only places each vertex and fogs it (baked.v.pica).

// floatsPerBakedVertex is a baked vertex: its position, and its colour as four bytes (red first)
// in the bits of the fourth float, so a vertex is 16 bytes instead of 40.
const floatsPerBakedVertex = 4

// bakeKey is what a mesh's baked colours depend on besides the mesh itself.
type bakeKey struct {
	normalMatrix [9]float32
	baseColour   graphics.Vec4
	kind         scene.MaterialKind
	light        lightKey
	lamps        lampKey
}

// lampKey is the point lights a draw takes (see sceneplan.DrawLights), and where the mesh is,
// which their light depends on; all zero for a draw no lamp reaches, so those bake as before.
type lampKey struct {
	count  int
	lights [sceneplan.MaxLightsPerDraw]sceneplan.FrameLight
	model  graphics.Mat4
}

func lampKeyOf(lights sceneplan.DrawLights, frameLights []sceneplan.FrameLight, model graphics.Mat4) lampKey {
	if lights.Count == 0 {
		return lampKey{}
	}
	key := lampKey{count: lights.Count, model: model}
	for slot, index := range lights.Indices[:lights.Count] {
		key.lights[slot] = frameLights[index]
	}
	return key
}

// lampUniforms are the lamps a draw takes as scene.v.pica reads them, for a node lit on the GPU
// (a moving one): each position with its inner radius in w, and each colour with one over the
// width of its fade in w. A lamp the draw does not take has no colour, so it adds nothing.
type lampUniforms struct {
	positions, colours [sceneplan.MaxLightsPerDraw]graphics.Vec4
}

// smallestFadeWidth keeps a lamp's fade from dividing by zero; sceneplan already keeps the inner
// radius below the outer one.
const smallestFadeWidth = 1e-4

func lampUniformsOf(lights sceneplan.DrawLights, frameLights []sceneplan.FrameLight) lampUniforms {
	var uniforms lampUniforms
	for slot, index := range lights.Indices[:lights.Count] {
		light := &frameLights[index]
		uniforms.positions[slot] = graphics.Vec4{light.Position[0], light.Position[1], light.Position[2], light.Inner}
		fadeWidth := max(light.Outer-light.Inner, smallestFadeWidth)
		uniforms.colours[slot] = graphics.Vec4{light.Color[0], light.Color[1], light.Color[2], 1 / fadeWidth}
	}
	return uniforms
}

// closeTo tells whether two draws' lamps light alike: a lamp that moves by less than about a
// colour step does not make its meshes bake again.
func (lamps lampKey) closeTo(other lampKey) bool {
	const step = 1.0 / 512
	if lamps.count != other.count {
		return false
	}
	for slot := range lamps.lights[:lamps.count] {
		light, otherLight := &lamps.lights[slot], &other.lights[slot]
		for component := range 3 {
			if math.Abs(float64(light.Position[component]-otherLight.Position[component])) > step ||
				math.Abs(float64(light.Color[component]-otherLight.Color[component])) > step {
				return false
			}
		}
		if light.Inner != otherLight.Inner || light.Outer != otherLight.Outer {
			return false
		}
	}
	return true
}

// lightKey is the part of a frame's shading that lights surfaces.
type lightKey struct {
	sunDirection, sunColour, ambientGround, ambientSpan graphics.Vec4
	brightness, verticalScale                           float32
}

func lightKeyOf(shading frameShading) lightKey {
	return lightKey{
		sunDirection: shading.sunDirection, sunColour: shading.sunColour,
		ambientGround: shading.ambientGround, ambientSpan: shading.ambientSpan,
		brightness: shading.settings[3], verticalScale: shading.settings[0],
	}
}

// closeTo tells whether two lights look the same: a slowly turning sun does not make every mesh
// bake again every frame, only when a colour would change by about one step of 255.
func (light lightKey) closeTo(other lightKey) bool {
	const step = 1.0 / 512
	vectors := [][2]graphics.Vec4{
		{light.sunDirection, other.sunDirection}, {light.sunColour, other.sunColour},
		{light.ambientGround, other.ambientGround}, {light.ambientSpan, other.ambientSpan},
	}
	for _, pair := range vectors {
		for component := range 4 {
			if math.Abs(float64(pair[0][component]-pair[1][component])) > step {
				return false
			}
		}
	}
	return math.Abs(float64(light.brightness-other.brightness)) <= step && light.verticalScale == other.verticalScale
}

// bakeVertices lays out a mesh's vertices with their final colour: lit like gles's scene shader
// (sun, sky and ground light, glow, or flat by brightness), raised to the screen's gamma and packed
// in bytes. Fog is left to the GPU, since it depends on the camera. It writes into into, grown
// when too small, and returns it: the caller keeps one buffer for every bake, since a new one each
// time fragments a heap that does not move what it holds (TinyGo's).
func bakeVertices(into []float32, mesh *scene.Mesh, key bakeKey) []float32 {
	if len(mesh.Normals) != len(mesh.Positions) {
		mesh.ComputeNormals()
	}
	weights := materialWeights(key.kind, key.light.brightness)
	vertexCount := mesh.VertexCount()
	baked := grownTo(into, vertexCount*floatsPerBakedVertex)
	for vertex := range vertexCount {
		colour := vertexColour(mesh, vertex)
		for channel := range 4 {
			colour[channel] *= key.baseColour[channel]
		}
		normal := worldNormal(key.normalMatrix, mesh.Normals[3*vertex:3*vertex+3], key.light.verticalScale)
		lamps := key.lamps.lightAt(mesh.Positions[3*vertex:3*vertex+3], normal, key.light.verticalScale)
		shaded := shade(colour, normal, weights, key.light, lamps)
		shaded[3] = float32(math.Max(float64(colour[3]), float64(minimumAlpha(key.kind))))
		offset := vertex * floatsPerBakedVertex
		copy(baked[offset:offset+3], mesh.Positions[3*vertex:3*vertex+3])
		baked[offset+3] = math.Float32frombits(packColour(shaded))
	}
	return baked
}

// grownTo is a slice of length size, reusing buffer's memory when it is large enough.
func grownTo[Element any](buffer []Element, size int) []Element {
	if cap(buffer) >= size {
		return buffer[:size]
	}
	return make([]Element, size)
}

func vertexColour(mesh *scene.Mesh, vertex int) graphics.Vec4 {
	if len(mesh.Colors) >= 4*(vertex+1) {
		return graphics.Vec4(mesh.Colors[4*vertex : 4*vertex+4])
	}
	return graphics.Vec4{1, 1, 1, 1}
}

// worldNormal turns a normal by the normal matrix (column by column), undoes the vertical scale
// on it, as gles's shader does, and keeps it of unit length.
func worldNormal(turn [9]float32, normal []float32, verticalScale float32) graphics.Vec4 {
	x := turn[0]*normal[0] + turn[3]*normal[1] + turn[6]*normal[2]
	y := (turn[1]*normal[0] + turn[4]*normal[1] + turn[7]*normal[2]) / verticalScale
	z := turn[2]*normal[0] + turn[5]*normal[1] + turn[8]*normal[2]
	length := float32(math.Sqrt(float64(x*x + y*y + z*z)))
	if length == 0 {
		return graphics.Vec4{0, 1, 0, 0}
	}
	return graphics.Vec4{x / length, y / length, z / length, 0}
}

// lightAt is the light the lamps bring to a vertex of the mesh (its position in the mesh, its
// normal in the world), as the scene shaders' lampLight: each fades from full at its inner
// radius to nothing at its outer one, and lights what faces it.
func (lamps *lampKey) lightAt(position []float32, normal graphics.Vec4, verticalScale float32) graphics.Vec4 {
	var total graphics.Vec4
	if lamps.count == 0 {
		return total
	}
	world := lamps.model.Apply(graphics.Vec4{position[0], position[1], position[2], 1})
	world[1] *= verticalScale
	for slot := range lamps.lights[:lamps.count] {
		light := &lamps.lights[slot]
		towards := graphics.Vec4{light.Position[0] - world[0], light.Position[1] - world[1], light.Position[2] - world[2], 0}
		distance := float32(math.Sqrt(float64(towards.Dot(towards))))
		fade := 1 - float32(smoothStep(float64(light.Inner), float64(light.Outer), float64(distance)))
		facing := max(normal.Dot(towards)/max(distance, 1e-4), 0)
		for channel := range 3 {
			total[channel] += light.Color[channel] * fade * facing
		}
	}
	return total
}

// shade is gles's lit, glowing and flat shading, chosen by the material's weights (water is
// drawn by the full shader instead), with the lamps' light added to the sun's.
func shade(colour, normal, weights graphics.Vec4, light lightKey, lamps graphics.Vec4) graphics.Vec4 {
	facingUp := normal[1]*0.5 + 0.5
	sunlight := float32(math.Max(float64(normal.Dot(light.sunDirection)), 0))
	var shaded graphics.Vec4
	for channel := range 3 {
		received := light.ambientGround[channel] + light.ambientSpan[channel]*facingUp + light.sunColour[channel]*sunlight + lamps[channel]
		lit := float32(math.Max(float64(colour[channel]*received), float64(colour[channel]*weights[3])))
		flat := colour[channel] * light.brightness
		shaded[channel] = weights[0]*lit + weights[1]*flat
	}
	return shaded
}

// packColour turns a linear colour into the screen's gamma, in bytes: red, green, blue, alpha.
// The PICA200 reads them in that order from little-endian memory.
func packColour(colour graphics.Vec4) uint32 {
	return screenByte(colour[0]) | screenByte(colour[1])<<8 | screenByte(colour[2])<<16 | linearByte(colour[3])<<24
}

// gammaSteps is how finely the gamma table covers 0 to 1: fine enough that every byte the screen
// can show is reached, as math.Pow would give it.
const gammaSteps = 4096

// gammaTable holds each step's linear value raised to the screen's gamma, as a byte. A table
// instead of math.Pow per channel makes baking several times faster on the 3DS's processor.
var gammaTable = func() [gammaSteps + 1]uint8 {
	var table [gammaSteps + 1]uint8
	for step := range table {
		table[step] = uint8(math.Round(math.Pow(float64(step)/gammaSteps, 1/2.2) * 255))
	}
	return table
}()

func screenByte(value float32) uint32 {
	clamped := min(max(value, 0), 1)
	return uint32(gammaTable[int(clamped*gammaSteps+0.5)])
}

func linearByte(value float32) uint32 {
	clamped := min(max(value, 0), 1)
	return uint32(clamped*255 + 0.5)
}
