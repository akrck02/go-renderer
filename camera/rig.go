package camera

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
)

// Rig places a headset's tracked space in the world. A headset reports where the eyes, the head
// and the hands are in its own space, in metres, from a point on the floor of the room; the rig
// says where that point stands in the scene, which way the room is turned, and how many scene
// units a metre is. Walking with a stick moves the rig; walking in the room moves the poses.
type Rig struct {
	// Position is where the room's origin (on its floor) is in the world.
	Position graphics.Vector
	// YawRadians turns the room around the world's up (y) axis, counter-clockwise seen from above.
	YawRadians float32
	// Scale is scene units per metre (0 = 1). Above 1 the player is a giant: a step crosses more
	// of the scene and the eyes are wider apart in it, so the world looks smaller.
	Scale float32
}

// Matrix takes the headset's space (metres) to the world: scale, turn, then move.
func (rig Rig) Matrix() graphics.Matrix {
	scale := scaleOrOne(rig.Scale)
	return graphics.Translate(rig.Position).
		Multiply(graphics.RotateY(float64(rig.YawRadians))).
		Multiply(graphics.Scale(graphics.Vector{scale, scale, scale, 1}))
}

// EyeView is the view matrix of an eye (a pose in the headset's space): from the world to the
// eye. It has no scale, so the near and far planes stay in scene units, like a Lens's.
func (rig Rig) EyeView(eye graphics.Pose) graphics.Matrix {
	return rig.worldPose(eye).ViewMatrix()
}

// EyePosition is where an eye is in the world: what lighting, fog and water reflections are
// worked out from.
func (rig Rig) EyePosition(eye graphics.Pose) graphics.Vector {
	return rig.worldPose(eye).Position
}

// Place is the model matrix of something tracked (a hand, a controller) in the world. It takes
// the rig's scale, so a model made in metres keeps the size it has in the room next to a giant's
// eyes as well as a person's.
func (rig Rig) Place(pose graphics.Pose) graphics.Matrix {
	return rig.Matrix().Multiply(pose.Matrix())
}

// worldPose is a pose of the headset's space moved into the world, without the scale: turned by
// the rig's yaw, its position moved (and scaled) by the rig.
func (rig Rig) worldPose(pose graphics.Pose) graphics.Pose {
	position := pose.Position
	position[3] = 1
	return graphics.Pose{
		Orientation: graphics.QuaternionAroundY(float64(rig.YawRadians)).Multiply(pose.Orientation),
		Position:    rig.Matrix().Apply(position),
	}
}

// coveringMargin widens the covering camera a little, so rounding never leaves out the edge of an
// eye's picture.
const coveringMargin = 1.001

// CoveringLens is one symmetric camera that sees everything the eyes see, for planning a frame
// (culling, levels of detail, the order of translucent surfaces) once for all of them; aspect is
// the width over the height of its picture. eyes and fields go in pairs, as the headset reports
// them; near and far are the eyes' clipping planes in scene units.
//
// It stands on the line through the middle of the eyes, looking the head's way. The eyes are
// apart, so a camera exactly between them would need a much wider angle to take in what each sees
// right in front of it; instead it steps back just enough (a few centimetres for a person) that
// both eyes' views fit in the angles they themselves have.
func CoveringLens(rig Rig, eyes []graphics.Pose, fields []graphics.FieldOfView, near, far float32) (lens Lens, aspect float32) {
	count := min(len(eyes), len(fields))
	if count == 0 {
		return rigLens(rig, near, far), 1
	}
	worldEyes := make([]graphics.Pose, count)
	for index := range count {
		worldEyes[index] = rig.worldPose(eyes[index])
	}
	middle, forward, up := averageEye(worldEyes)
	view := graphics.LookAt(middle, middle.Add(forward), up)
	corners := eyeCornersSeenFrom(view, worldEyes, fields[:count], near, far)
	return coveringFrom(corners, middle, forward, up, near, far)
}

// rigLens is the camera of a headset that reports no eyes: at the rig, looking the rig's way.
func rigLens(rig Rig, near, far float32) Lens {
	forward := graphics.RotateY(float64(rig.YawRadians)).Apply(graphics.Vector{0, 0, -1, 0})
	return Lens{Position: rig.Position, Target: rig.Position.Add(forward), Up: graphics.Vector{0, 1, 0, 0}, Near: near, Far: far}
}

// averageEye returns the middle of the eyes and the directions they look and hold as up, on average.
func averageEye(eyes []graphics.Pose) (middle, forward, up graphics.Vector) {
	share := 1 / float32(len(eyes))
	for _, eye := range eyes {
		for axis := range 3 {
			middle[axis] += eye.Position[axis] * share
		}
		forward = forward.Add(eye.Orientation.Rotate(graphics.Vector{0, 0, -1, 0}))
		up = up.Add(eye.Orientation.Rotate(graphics.Vector{0, 1, 0, 0}))
	}
	middle[3] = 1
	return middle, forward.Normalize(), up.Normalize()
}

// eyeCornersSeenFrom returns the corners of every eye's frustum (on its near and far planes) in
// the space of view. A frustum is the convex hull of its corners, so a camera that contains them
// contains all the eye sees.
func eyeCornersSeenFrom(view graphics.Matrix, eyes []graphics.Pose, fields []graphics.FieldOfView, near, far float32) []graphics.Vector {
	corners := make([]graphics.Vector, 0, len(eyes)*8)
	for index, eye := range eyes {
		toView := view.Multiply(eye.Matrix())
		for _, corner := range frustumCorners(fields[index], near, far) {
			corners = append(corners, toView.Apply(corner))
		}
	}
	return corners
}

// frustumCorners returns the eight corners of the space an eye with this field of view sees
// between near and far, in the eye's own space (looking down -z): the near four, then the far four.
func frustumCorners(field graphics.FieldOfView, near, far float32) [8]graphics.Vector {
	horizontal := [2]float64{float64(field.LeftRadians), float64(field.RightRadians)}
	vertical := [2]float64{float64(field.DownRadians), float64(field.UpRadians)}
	var corners [8]graphics.Vector
	for plane, distance := range [2]float32{near, far} {
		for side := range 2 {
			for height := range 2 {
				x := float32(math.Tan(horizontal[side])) * distance
				y := float32(math.Tan(vertical[height])) * distance
				corners[plane*4+side*2+height] = graphics.Vector{x, y, -distance, 1}
			}
		}
	}
	return corners
}

// coveringFrom builds the covering camera from the eyes' corners, seen from the middle of the
// eyes. Its angles are the widest the eyes see; it then steps back along forward until the corners
// near the eyes fit in those angles too.
func coveringFrom(corners []graphics.Vector, middle, forward, up graphics.Vector, near, far float32) (Lens, float32) {
	tangentX, tangentY := widestTangents(corners)
	back := float32(0)
	for _, corner := range corners {
		depth := -corner[2]
		back = max(back, abs32(corner[0])/tangentX-depth, abs32(corner[1])/tangentY-depth)
	}
	closest, farthest := depthRange(corners)
	position := middle
	for axis := range 3 {
		position[axis] -= forward[axis] * back
	}
	lens := Lens{
		Position:           position,
		Target:             position.Add(forward),
		Up:                 up,
		FieldOfViewRadians: float32(2 * math.Atan(float64(tangentY))),
		Near:               min(near, (closest+back)/coveringMargin),
		Far:                max(far, (farthest+back)*coveringMargin),
	}
	return lens, tangentX / tangentY
}

// widestTangents returns the tangents of the widest half angles, sideways and up or down, at which
// the far corners are seen (the near ones are left to stepping back), with a small margin.
func widestTangents(corners []graphics.Vector) (tangentX, tangentY float32) {
	_, farthest := depthRange(corners)
	for _, corner := range corners {
		depth := -corner[2]
		if depth < farthest/2 {
			continue
		}
		tangentX = max(tangentX, abs32(corner[0])/depth)
		tangentY = max(tangentY, abs32(corner[1])/depth)
	}
	tangentX = max(tangentX, 1e-3) * coveringMargin
	tangentY = max(tangentY, 1e-3) * coveringMargin
	return tangentX, tangentY
}

// depthRange returns how near and how far in front of the camera the corners are.
func depthRange(corners []graphics.Vector) (closest, farthest float32) {
	closest = float32(math.Inf(1))
	for _, corner := range corners {
		depth := -corner[2]
		closest = min(closest, depth)
		farthest = max(farthest, depth)
	}
	return closest, farthest
}

func abs32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}
