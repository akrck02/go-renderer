package sceneplan

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// MaxLightsPerDraw is how many point lights one draw takes at most: the nearest whose reach meets
// what it draws. Few enough for OpenGL ES 2's uniforms and a fixed loop in every shader; a scene
// with more lights still lights each part of it by its nearest ones, as Godot and Unity's forward
// renderers do.
const MaxLightsPerDraw = 4

// FrameLight is a scene's point light as the shaders take it: its position with the vertical scale,
// its colour times its intensity, and its radii.
type FrameLight struct {
	Position     graphics.Vec4
	Color        graphics.Vec4 // linear, times the intensity
	Inner, Outer float32
}

// DrawLights are the lights a command takes: indices into the frame's Lights.
type DrawLights struct {
	Indices [MaxLightsPerDraw]uint16
	Count   int
}

// mostLights is how many of a scene's lights a frame takes; the rest are left out.
const mostLights = 1 << 16

// innerShareMost is the largest share of its outer radius a light's inner radius can be.
const innerShareMost = 0.999

// prepareLights lays out a scene's lights for the frame, in the planner's own list.
func (planner *Planner) prepareLights(world *scene.Scene) []FrameLight {
	lights := planner.lights[:0]
	for _, light := range world.Lights[:min(len(world.Lights), mostLights)] {
		position := light.Position
		position[1] *= planner.verticalScale
		intensity := light.Color[3]
		colour := graphics.Vec4{light.Color[0] * intensity, light.Color[1] * intensity, light.Color[2] * intensity, 1}
		// the inner radius stays short of the outer one: a fade over no distance is undefined in GLSL
		inner := min(max(light.Inner, 0), light.Outer*innerShareMost)
		lights = append(lights, FrameLight{Position: position, Color: colour, Inner: inner, Outer: light.Outer})
	}
	planner.lights = lights
	return lights
}

// nearestLights chooses the lights nearest a box of the world, nearest first: only those that
// reach it, when reaching.
func nearestLights(lights []FrameLight, bounds box, reaching bool) DrawLights {
	var chosen DrawLights
	var distances [MaxLightsPerDraw]float32
	for index := range lights {
		light := &lights[index]
		distance := bounds.squaredDistanceTo(light.Position)
		if light.Outer <= 0 || reaching && distance >= light.Outer*light.Outer {
			continue
		}
		chosen.insert(uint16(index), distance, &distances)
	}
	return chosen
}

// insert keeps a light among the chosen when it is nearer than one of them, the nearest first.
func (chosen *DrawLights) insert(index uint16, distance float32, distances *[MaxLightsPerDraw]float32) {
	place := chosen.Count
	for place > 0 && distances[place-1] > distance {
		place--
	}
	if place == MaxLightsPerDraw {
		return
	}
	last := min(chosen.Count, MaxLightsPerDraw-1)
	for move := last; move > place; move-- {
		chosen.Indices[move], distances[move] = chosen.Indices[move-1], distances[move-1]
	}
	chosen.Indices[place], distances[place] = index, distance
	chosen.Count = min(chosen.Count+1, MaxLightsPerDraw)
}

// squaredDistanceTo is the squared distance from the box to a point: 0 inside it.
func (bounds box) squaredDistanceTo(point graphics.Vec4) float32 {
	total := float32(0)
	for axis := range 3 {
		outside := max(bounds.minimum[axis]-point[axis], 0, point[axis]-bounds.maximum[axis])
		total += outside * outside
	}
	return total
}

// lightsOfNode chooses the lights a node's mesh takes: those that reach its box, nearest first.
// Instances are spread over the scene, too far apart for one box: they take the lights nearest
// the camera, where they are seen best.
func (planner *Planner) lightsOfNode(node *scene.Node, transform graphics.Mat4) DrawLights {
	if len(planner.lights) == 0 {
		return DrawLights{}
	}
	if node.Instances != nil {
		camera := planner.walkCamera.Position
		return nearestLights(planner.lights, box{minimum: camera, maximum: camera}, false)
	}
	bounds := planner.boundsOf(node.Mesh).transformed(transform, planner.verticalScale)
	return nearestLights(planner.lights, bounds, true)
}
