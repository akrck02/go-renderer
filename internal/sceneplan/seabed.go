package sceneplan

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// SeabedRegion is the box the seabed map sees from straight above: its projection (OpenGL clip
// space) and the heights at depth 0 and 1.
type SeabedRegion struct {
	ViewProjection graphics.Matrix
	Top, Bottom    float32
}

// SeabedPlan says whether the seabed map, the depth of the ground seen from above over the whole
// scene, must be drawn again this frame. The water shader reads it to know how deep the water is.
type SeabedPlan struct {
	Redraw    bool // the scene is new or its ground changed
	Available bool // there is ground to draw
	Region    SeabedRegion
	Casters   []Command // what to draw into the map, whole and without culling
}

// planSeabed decides whether the seabed map must be drawn again, and around what. Without water
// there is no seabed to draw: measuring the casters, every lit mesh when none is ground, again
// whenever one changes, would cost a scene of moving meshes most of a frame on a slow processor.
func (planner *Planner) planSeabed(world *scene.Scene, commands []Command) SeabedPlan {
	if world.Environment.Water == nil {
		return SeabedPlan{}
	}
	casters := planner.seabedCasters(commands)
	if planner.seabedFor == world && !anyMeshDirty(casters) {
		return SeabedPlan{Available: planner.seabed.Available, Region: planner.seabed.Region}
	}
	planner.seabedFor = world
	planner.seabed = SeabedPlan{Available: len(casters) > 0}
	if !planner.seabed.Available {
		return planner.seabed
	}
	planner.seabed.Region = fitSeabedRegion(boundsOfCommands(casters))
	plan := planner.seabed
	plan.Redraw, plan.Casters = true, casters
	return plan
}

// seabedCasters returns the walkable ground, or every opaque lit mesh when nothing is marked
// as ground, in the planner's own lists: they hold until the next call.
func (planner *Planner) seabedCasters(commands []Command) []Command {
	ground, opaque := planner.ground[:0], planner.opaque[:0]
	for _, command := range commands {
		if command.Transparent || command.Material.Kind != scene.KindLit {
			continue
		}
		opaque = append(opaque, command)
		if command.Node.Mesh.Ground {
			ground = append(ground, command)
		}
	}
	planner.ground, planner.opaque = ground, opaque // kept for the next frame's lists
	if len(ground) > 0 {
		return ground
	}
	return opaque
}

func anyMeshDirty(commands []Command) bool {
	for _, command := range commands {
		if command.Node.Mesh.Dirty || (command.Node.Instances != nil && command.Node.Instances.Dirty) {
			return true
		}
	}
	return false
}

// boundsOfCommands returns the world bounding box of the commands' meshes.
func boundsOfCommands(commands []Command) (minimum, maximum graphics.Vector) {
	partial := &scene.Scene{}
	for _, command := range commands {
		partial.Nodes = append(partial.Nodes, &scene.Node{Transform: command.World, Mesh: command.Node.Mesh, Instances: command.Node.Instances})
	}
	return partial.Bounds()
}

// fitSeabedRegion looks straight down on the bounding box, with a small margin on every side.
func fitSeabedRegion(minimum, maximum graphics.Vector) SeabedRegion {
	margin := float32(math.Max(float64(maximum[0]-minimum[0]), float64(maximum[2]-minimum[2]))) * 0.01
	top, bottom := maximum[1]+margin, minimum[1]-margin
	centerX, centerZ := (minimum[0]+maximum[0])/2, (minimum[2]+maximum[2])/2
	halfWidth := (maximum[0]-minimum[0])/2 + margin
	halfDepth := (maximum[2]-minimum[2])/2 + margin
	view := graphics.LookAt(graphics.Vector{centerX, top, centerZ, 1}, graphics.Vector{centerX, bottom, centerZ, 1}, graphics.Vector{0, 0, -1, 0})
	projection := graphics.OrthographicOpenGL(-halfWidth, halfWidth, -halfDepth, halfDepth, 0, top-bottom)
	return SeabedRegion{ViewProjection: projection.Multiply(view), Top: top, Bottom: bottom}
}
