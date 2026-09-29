// Package scene is a retained, backend-independent description of a 3D scene: nodes with
// transforms, meshes with per-vertex normals and colors, materials, GPU instancing and the
// environment (sun, sky, fog, water).
//
// Backends upload meshes once and draw them every frame. Simulations can change geometry or
// instances later and mark them Dirty so the backend uploads them again.
package scene

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
)

// MaterialKind selects the shading model.
type MaterialKind int

const (
	KindLit       MaterialKind = iota // diffuse sun + sky ambient + fog
	KindUnlit                         // flat color (+ fog)
	KindWater                         // animated water surface
	KindWaterfall                     // animated, translucent falling water
)

// Material describes how a mesh is shaded.
type Material struct {
	Name        string
	Kind        MaterialKind
	BaseColor   graphics.Vec4 // linear RGBA multiplied with vertex colors
	DoubleSided bool
	Transparent bool
	Sway        float32 // how much the wind bends the mesh (0 = rigid); for plants modelled with height 1
}

// Mesh is triangle geometry. Positions and normals are xyz triples, colors rgba quads (linear).
// Indices are optional: without them every three vertices form a triangle.
type Mesh struct {
	Name      string
	Positions []float32
	Normals   []float32
	Colors    []float32
	Indices   []uint32
	Material  *Material
	Ground    bool // walkable surface (used by GroundHeight)
	Dirty     bool // geometry changed: backends upload it again
	GPU       any  // backend handle
}

// VertexCount returns the number of vertices.
func (mesh *Mesh) VertexCount() int { return len(mesh.Positions) / 3 }

// Vertex returns the position of a vertex.
func (mesh *Mesh) Vertex(index uint32) graphics.Vec4 {
	return graphics.Vec4{mesh.Positions[3*index], mesh.Positions[3*index+1], mesh.Positions[3*index+2], 1}
}

// ForEachTriangle calls visit with the vertex indices of every triangle.
func (mesh *Mesh) ForEachTriangle(visit func(first, second, third uint32)) {
	if len(mesh.Indices) > 0 {
		for index := 0; index+2 < len(mesh.Indices); index += 3 {
			visit(mesh.Indices[index], mesh.Indices[index+1], mesh.Indices[index+2])
		}
		return
	}
	for index := uint32(0); int(index)+2 < mesh.VertexCount(); index += 3 {
		visit(index, index+1, index+2)
	}
}

// Instances draws a mesh many times with a transform and a color per instance.
type Instances struct {
	Transforms []graphics.Mat4
	Colors     []graphics.Vec4
	Dirty      bool
	GPU        any
}

// Node places a mesh (optionally instanced) in the world; children inherit the transform.
type Node struct {
	Name      string
	Transform graphics.Mat4
	Mesh      *Mesh
	Instances *Instances
	Children  []*Node
	Hidden    bool
	Tags      map[string]any // free data from the file (glTF extras) for the application
}

// Water is an infinite animated water plane.
type Water struct {
	Level      float32
	Shallow    graphics.Vec4
	Deep       graphics.Vec4
	Size       float32 // extent of the plane (world units)
	WaveLength float32 // length of the main wave (world units); 0 = Size / 2000
	FloorDepth float32 // depth of an opaque floor drawn under the water; 0 = Size / 500
	ColorDepth float32 // water depth at which the color is about two thirds of the way from Shallow to Deep; 0 = FloorDepth / 8
}

// EffectiveWaveLength returns the wave length, deriving it from the size when unset.
func (water *Water) EffectiveWaveLength() float32 {
	if water.WaveLength > 0 {
		return water.WaveLength
	}
	return water.Size / 2000
}

// EffectiveFloorDepth returns the floor depth, deriving it from the size when unset.
func (water *Water) EffectiveFloorDepth() float32 {
	if water.FloorDepth > 0 {
		return water.FloorDepth
	}
	return water.Size / 500
}

// EffectiveColorDepth returns the color depth, deriving it from the floor depth when unset.
func (water *Water) EffectiveColorDepth() float32 {
	if water.ColorDepth > 0 {
		return water.ColorDepth
	}
	return water.EffectiveFloorDepth() / 8
}

// Environment holds the lighting and atmosphere.
type Environment struct {
	SunDirection  graphics.Vec4 // towards the sun, normalized
	SunColor      graphics.Vec4
	SkyZenith     graphics.Vec4
	SkyHorizon    graphics.Vec4
	GroundAmbient graphics.Vec4
	Ambient       float32
	FogColor      graphics.Vec4
	FogNear       float32
	FogFar        float32 // 0 = no fog
	Water         *Water
	VerticalScale float32 // vertical exaggeration applied to the whole scene (0 = 1)
	Sky           *Sky    // sky over the day and night (nil = DefaultSky)
	Wind          Wind
}

// Wind moves clouds and bends plants (materials with Sway).
type Wind struct {
	Direction graphics.Vec4 // horizontal direction the wind blows towards
	Strength  float32       // 0 calm, 1 strong
}

// DefaultEnvironment returns a daylight environment.
func DefaultEnvironment() Environment {
	return Environment{
		SunDirection:  graphics.Vec4{-0.5, 0.6, 0.6, 0}.Normalize(),
		SunColor:      graphics.Vec4{1, 0.95, 0.85, 1},
		SkyZenith:     graphics.Vec4{0.28, 0.48, 0.72, 1},
		SkyHorizon:    graphics.Vec4{0.74, 0.82, 0.86, 1},
		GroundAmbient: graphics.Vec4{0.23, 0.27, 0.2, 1},
		Ambient:       0.55,
		FogColor:      graphics.Vec4{0.74, 0.82, 0.86, 1},
		VerticalScale: 1,
		Sky:           DefaultSky(),
		Wind:          Wind{Direction: graphics.Vec4{1, 0, 0.3, 0}.Normalize(), Strength: 0.35},
	}
}

var fallbackSky = DefaultSky()

// EffectiveSky returns the sky, or the default sky when unset.
func (environment Environment) EffectiveSky() *Sky {
	if environment.Sky == nil {
		return fallbackSky
	}
	return environment.Sky
}

// EffectiveVerticalScale returns the vertical scale, treating 0 as 1.
func (environment Environment) EffectiveVerticalScale() float32 {
	if environment.VerticalScale == 0 {
		return 1
	}
	return environment.VerticalScale
}

// Scene is the root of a scene.
type Scene struct {
	Nodes       []*Node
	Environment Environment
	Tags        map[string]any

	ground *groundIndex
}

// New returns an empty scene with the default environment.
func New() *Scene { return &Scene{Environment: DefaultEnvironment()} }

// Walk visits every visible node with its world transform.
func (scene *Scene) Walk(visit func(node *Node, world graphics.Mat4)) {
	for _, node := range scene.Nodes {
		walkNode(node, graphics.Identity(), visit)
	}
}

func walkNode(node *Node, parentWorld graphics.Mat4, visit func(node *Node, world graphics.Mat4)) {
	if node.Hidden {
		return
	}
	world := parentWorld.Multiply(localTransform(node))
	visit(node, world)
	for _, child := range node.Children {
		walkNode(child, world, visit)
	}
}

func localTransform(node *Node) graphics.Mat4 {
	if node.Transform == (graphics.Mat4{}) {
		return graphics.Identity()
	}
	return node.Transform
}

// Bounds returns the world-space bounding box of all meshes (instances counted by their origin).
func (scene *Scene) Bounds() (minimum, maximum graphics.Vec4) {
	box := newBoundingBox()
	scene.Walk(func(node *Node, world graphics.Mat4) {
		switch {
		case node.Mesh == nil:
		case node.Instances != nil:
			for _, transform := range node.Instances.Transforms {
				box.include(world.Multiply(transform).Apply(graphics.Vec4{0, 0, 0, 1}))
			}
		default:
			for index := uint32(0); int(index) < node.Mesh.VertexCount(); index++ {
				box.include(world.Apply(node.Mesh.Vertex(index)))
			}
		}
	})
	return box.minimum, box.maximum
}

type boundingBox struct{ minimum, maximum graphics.Vec4 }

func newBoundingBox() *boundingBox {
	infinity := float32(math.Inf(1))
	return &boundingBox{graphics.Vec4{infinity, infinity, infinity, 1}, graphics.Vec4{-infinity, -infinity, -infinity, 1}}
}

func (box *boundingBox) include(point graphics.Vec4) {
	for axis := 0; axis < 3; axis++ {
		box.minimum[axis] = float32(math.Min(float64(box.minimum[axis]), float64(point[axis])))
		box.maximum[axis] = float32(math.Max(float64(box.maximum[axis]), float64(point[axis])))
	}
}

// ComputeNormals fills smooth vertex normals from the triangles (weighted by triangle area).
func (mesh *Mesh) ComputeNormals() {
	normals := make([]float32, len(mesh.Positions))
	mesh.ForEachTriangle(func(first, second, third uint32) {
		faceNormal := triangleNormal(mesh.Vertex(first), mesh.Vertex(second), mesh.Vertex(third))
		for _, vertex := range []uint32{first, second, third} {
			normals[3*vertex] += faceNormal[0]
			normals[3*vertex+1] += faceNormal[1]
			normals[3*vertex+2] += faceNormal[2]
		}
	})
	normalizeEachTriple(normals)
	mesh.Normals = normals
}

// triangleNormal returns the unnormalized normal of a triangle (its length is twice the area).
func triangleNormal(first, second, third graphics.Vec4) graphics.Vec4 {
	edgeOne := graphics.Vec4{second[0] - first[0], second[1] - first[1], second[2] - first[2], 0}
	edgeTwo := graphics.Vec4{third[0] - first[0], third[1] - first[1], third[2] - first[2], 0}
	return edgeOne.Cross(edgeTwo)
}

// normalizeEachTriple normalizes consecutive xyz triples; zero vectors become straight up.
func normalizeEachTriple(values []float32) {
	for index := 0; index+2 < len(values); index += 3 {
		unit := graphics.Vec4{values[index], values[index+1], values[index+2], 0}.Normalize()
		if unit == (graphics.Vec4{}) {
			unit = graphics.Vec4{0, 1, 0, 0}
		}
		values[index], values[index+1], values[index+2] = unit[0], unit[1], unit[2]
	}
}
