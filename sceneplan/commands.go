package sceneplan

import (
	"math"
	"slices"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
)

// Command is one node to draw this frame.
type Command struct {
	Node                  *scene.Node
	World                 graphics.Mat4
	Material              *scene.Material
	SquaredCameraDistance float32
	Transparent           bool
	SeaSurface            bool // the environment's water plane: the only surface the level variation moves
}

// CameraMatrices are the matrices of one frame, in OpenGL's clip space (depth from -1 to 1); a
// backend with another clip space converts them.
type CameraMatrices struct {
	Projection, View, ViewProjection graphics.Mat4
}

// ComputeCameraMatrices returns the camera's matrices for a framebuffer of this size.
func ComputeCameraMatrices(camera models.Camera, width, height int) CameraMatrices {
	aspect := float64(width) / math.Max(float64(height), 1)
	projection := camera.ProjectionMatrix(aspect)
	view := graphics.LookAt(camera.Position, camera.Target, camera.Up)
	return CameraMatrices{Projection: projection, View: view, ViewProjection: projection.Multiply(view)}
}

// IsTransparent tells whether a material is blended over what is behind it.
func IsTransparent(material *scene.Material) bool {
	return material.Transparent || material.Kind == scene.KindWater || material.Kind == scene.KindWaterfall
}

// CastsShadow tells whether a command is drawn into the shadow map.
func CastsShadow(command Command) bool {
	return !command.Transparent && command.Material.Kind != scene.KindUnlit
}

// BaseColorOf returns a material's colour, white when it is unset.
func BaseColorOf(material *scene.Material) graphics.Vec4 {
	if material.BaseColor == (graphics.Vec4{}) {
		return graphics.Vec4{1, 1, 1, 1}
	}
	return material.BaseColor
}

func materialOrDefault(material *scene.Material) *scene.Material {
	if material == nil {
		return &scene.Material{BaseColor: graphics.Vec4{1, 1, 1, 1}}
	}
	return material
}

func squaredHorizontalDistance(first, second graphics.Vec4) float32 {
	deltaX, deltaZ := first[0]-second[0], first[2]-second[2]
	return deltaX*deltaX + deltaZ*deltaZ
}

// collectCommands lists every node with a mesh, and the sea and its floor when there is water,
// in the planner's own list: it holds until the next call.
func (planner *Planner) collectCommands(world *scene.Scene, camera models.Camera) []Command {
	planner.commands, planner.walkCamera = planner.commands[:0], camera
	if planner.collectOne == nil {
		planner.collectOne = planner.collectNode
	}
	world.Walk(planner.collectOne)
	if world.Environment.Water != nil {
		planner.commands = append(planner.commands, planner.waterFloorCommand(world.Environment.Water, camera))
		planner.commands = append(planner.commands, planner.waterCommand(world.Environment.Water, camera))
	}
	return planner.commands
}

// collectNode adds a node's command, when it has a mesh.
func (planner *Planner) collectNode(node *scene.Node, transform graphics.Mat4) {
	if node.Mesh == nil {
		return
	}
	material := materialOrDefault(node.Mesh.Material)
	planner.commands = append(planner.commands, Command{
		Node: node, World: transform, Material: material, Transparent: IsTransparent(material),
		SquaredCameraDistance: squaredHorizontalDistance(transform.Apply(graphics.Vec4{0, 0, 0, 1}), planner.walkCamera.Position),
	})
}

// sortOpaqueBeforeTransparent keeps opaque commands in order and draws transparent ones last,
// farthest first, so blending composes correctly.
func sortOpaqueBeforeTransparent(commands []Command) {
	slices.SortStableFunc(commands, compareForDrawing) // no reflection, no closure: no garbage
}

func compareForDrawing(first, second Command) int {
	switch {
	case first.Transparent != second.Transparent:
		if first.Transparent {
			return 1
		}
		return -1
	case !first.Transparent:
		return 0
	case first.SquaredCameraDistance > second.SquaredCameraDistance:
		return -1
	case first.SquaredCameraDistance < second.SquaredCameraDistance:
		return 1
	}
	return 0
}

// splitOpaqueAndTransparent cuts the sorted commands where the transparent ones start.
func splitOpaqueAndTransparent(commands []Command) (opaque, transparent []Command) {
	for index, command := range commands {
		if command.Transparent {
			return commands[:index], commands[index:]
		}
	}
	return commands, nil
}

// waterCommand returns a large water plane that follows the camera horizontally.
func (planner *Planner) waterCommand(water *scene.Water, camera models.Camera) Command {
	if planner.waterPlane == nil {
		planner.waterPlane = newWaterGrid(water.Size, planner.WaterGridDivisions)
	}
	deepColor := water.Deep
	if deepColor == (graphics.Vec4{}) {
		deepColor = graphics.Vec4{0.05, 0.22, 0.28, 0.85}
	}
	planner.waterPlane.Material.BaseColor = deepColor
	world := graphics.Translate(graphics.Vec4{camera.Position[0], water.Level, camera.Position[2], 0})
	node := &scene.Node{Name: "water", Mesh: planner.waterPlane}
	return Command{Node: node, World: world, Material: planner.waterPlane.Material, SquaredCameraDistance: -1, Transparent: true, SeaSurface: true}
}

// waterFloorCommand returns an opaque dark floor under the water, so that nothing behind the
// scene shows through the translucent surface where the geometry ends.
func (planner *Planner) waterFloorCommand(water *scene.Water, camera models.Camera) Command {
	if planner.waterFloor == nil {
		planner.waterFloor = newWaterPlane(water.Size)
		planner.waterFloor.Material = &scene.Material{Name: "water floor", Kind: scene.KindUnlit, DoubleSided: true}
	}
	deep := water.Deep
	planner.waterFloor.Material.BaseColor = graphics.Vec4{deep[0] * 0.6, deep[1] * 0.6, deep[2] * 0.6, 1}
	depth := water.Level - water.EffectiveFloorDepth()
	world := graphics.Translate(graphics.Vec4{camera.Position[0], depth, camera.Position[2], 0})
	node := &scene.Node{Name: "water floor", Mesh: planner.waterFloor}
	return Command{Node: node, World: world, Material: planner.waterFloor.Material}
}

// defaultWaterGridDivisions is the number of cells per side of the water surface.
const defaultWaterGridDivisions = 256

// newWaterGrid returns the water surface as a grid whose cells are small near its centre (the
// camera) and large far away, so a varying surface (tides) meets the coast accurately nearby.
func newWaterGrid(size float32, divisions int) *scene.Mesh {
	if size <= 0 {
		size = 1000
	}
	if divisions <= 0 {
		divisions = defaultWaterGridDivisions
	}
	half := size / 2
	vertices := divisions + 1
	mesh := &scene.Mesh{Name: "water",
		Material: &scene.Material{Name: "water", Kind: scene.KindWater, Transparent: true, DoubleSided: true}}
	for row := 0; row < vertices; row++ {
		for column := 0; column < vertices; column++ {
			mesh.Positions = append(mesh.Positions, half*denseNearCentre(column, vertices), 0, half*denseNearCentre(row, vertices))
			mesh.Normals = append(mesh.Normals, 0, 1, 0)
		}
	}
	for row := 0; row < divisions; row++ {
		for column := 0; column < divisions; column++ {
			corner := uint32(row*vertices + column)
			next := corner + uint32(vertices)
			mesh.Indices = append(mesh.Indices, corner, next, corner+1, corner+1, next, next+1)
		}
	}
	return mesh
}

// denseNearCentre maps a grid index to -1..1 with a quadratic spacing: fine in the middle.
func denseNearCentre(index, count int) float32 {
	linear := 2*float32(index)/float32(count-1) - 1
	if linear < 0 {
		return -linear * linear
	}
	return linear * linear
}

func newWaterPlane(size float32) *scene.Mesh {
	if size <= 0 {
		size = 1000
	}
	half := size / 2
	return &scene.Mesh{
		Name:      "water",
		Positions: []float32{-half, 0, -half, half, 0, -half, -half, 0, half, half, 0, half},
		Normals:   []float32{0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0},
		Indices:   []uint32{0, 2, 1, 1, 2, 3},
		Material:  &scene.Material{Name: "water", Kind: scene.KindWater, Transparent: true, DoubleSided: true},
	}
}

// WaveLengthFor is the length of the sea's main wave, 1 without water.
func WaveLengthFor(water *scene.Water) float32 {
	if water == nil || water.EffectiveWaveLength() <= 0 {
		return 1
	}
	return water.EffectiveWaveLength()
}

// ShallowColorOf is the colour of shallow water, with a default when it is unset.
func ShallowColorOf(water *scene.Water) graphics.Vec4 {
	if water.Shallow == (graphics.Vec4{}) {
		return graphics.Vec4{0.3, 0.62, 0.6, 0.5}
	}
	return water.Shallow
}

// WindDirection is the horizontal direction of the wind, normalised, as x and z.
func WindDirection(wind scene.Wind) (float32, float32) {
	direction := graphics.Vec4{wind.Direction[0], 0, wind.Direction[2], 0}.Normalize()
	return direction[0], direction[2]
}
