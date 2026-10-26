package camera

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
)

// headsetEyes are two eyes of a made-up headset: 64 mm apart at a standing height, each seeing
// more to its own side than towards the nose, as headsets do.
func headsetEyes(head graphics.Quaternion) ([]graphics.Pose, []graphics.FieldOfView) {
	headPosition := graphics.Vector{0.2, 1.6, -0.3, 1}
	eyes := make([]graphics.Pose, 2)
	for index, side := range []float32{-1, 1} {
		offset := head.Rotate(graphics.Vector{side * 0.032, 0, 0, 0})
		eyes[index] = graphics.Pose{Orientation: head, Position: headPosition.Add(offset)}
	}
	fields := []graphics.FieldOfView{
		{LeftRadians: -0.9, RightRadians: 0.75, UpRadians: 0.8, DownRadians: -0.85},
		{LeftRadians: -0.75, RightRadians: 0.9, UpRadians: 0.8, DownRadians: -0.85},
	}
	return eyes, fields
}

func testRig() Rig {
	return Rig{Position: graphics.Vector{10, 0, 5, 1}, YawRadians: 0.6, Scale: 2}
}

func sameSpot(first, second graphics.Vector) bool {
	for axis := range 3 {
		if math.Abs(float64(first[axis]-second[axis])) > 1e-4 {
			return false
		}
	}
	return true
}

func TestAnEyesViewSeesItselfAtTheOrigin(test *testing.T) {
	rig := testRig()
	eyes, _ := headsetEyes(graphics.QuaternionAroundY(0.3))
	for _, eye := range eyes {
		position := rig.EyePosition(eye)
		if seen := rig.EyeView(eye).Apply(position); !sameSpot(seen, graphics.Vector{}) {
			test.Fatalf("the eye seen from itself is at %v, want the origin", seen)
		}
		if placed := rig.Place(eye).Apply(graphics.Vector{0, 0, 0, 1}); !sameSpot(placed, position) {
			test.Fatalf("Place puts the eye at %v, EyePosition at %v", placed, position)
		}
	}
}

func TestTheRigScalesTurnsAndMovesTheRoom(test *testing.T) {
	rig := Rig{Position: graphics.Vector{10, 0, 5, 1}, YawRadians: math.Pi / 2, Scale: 3}
	// a metre ahead in the room (-z), turned a quarter left, is three units towards -x
	ahead := graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0, 0, -1, 1}}
	if position := rig.EyePosition(ahead); !sameSpot(position, graphics.Vector{7, 0, 5, 1}) {
		test.Fatalf("a metre ahead is at %v, want (7, 0, 5)", position)
	}
	// and it looks the room's way: a point further ahead in the world is straight in front
	inFront := rig.EyeView(ahead).Apply(graphics.Vector{4, 0, 5, 1})
	if !sameSpot(inFront, graphics.Vector{0, 0, -3, 1}) {
		test.Fatalf("a point ahead is seen at %v, want straight in front", inFront)
	}
}

func TestAZeroScaleIsOne(test *testing.T) {
	pose := graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{1, 2, 3, 1}}
	if position := (Rig{}).EyePosition(pose); !sameSpot(position, pose.Position) {
		test.Fatalf("an empty rig moved the pose to %v", position)
	}
}

func TestTheCoveringLensContainsEveryEyesView(test *testing.T) {
	heads := map[string]graphics.Quaternion{
		"level":       graphics.IdentityQuaternion(),
		"turned":      graphics.QuaternionAroundY(1.1),
		"looking up":  graphics.QuaternionAroundY(-0.4).Multiply(graphics.Quaternion{X: float32(math.Sin(0.35)), W: float32(math.Cos(0.35))}),
		"head tilted": graphics.Quaternion{Z: float32(math.Sin(0.2)), W: float32(math.Cos(0.2))},
	}
	near, far := float32(0.1), float32(500)
	for name, head := range heads {
		rig := testRig()
		eyes, fields := headsetEyes(head)
		covering, aspect := CoveringLens(rig, eyes, fields, near, far)
		coveringView := covering.ProjectionMatrix(float64(aspect)).
			Multiply(graphics.LookAt(covering.Position, covering.Target, covering.Up))
		for index, eye := range eyes {
			eyeToWorld := rig.worldPose(eye).Matrix()
			for _, distance := range []float32{near, 0.5, 10, far} {
				corners := frustumCorners(fields[index], distance, distance)
				for _, corner := range corners[:4] {
					clip := coveringView.Apply(eyeToWorld.Apply(corner))
					if !insideClip(clip) {
						test.Fatalf("%s: eye %d's corner at %v is outside the covering lens (clip %v)", name, index, distance, clip)
					}
				}
			}
		}
	}
}

func TestTheCoveringLensStaysCloseToTheEyes(test *testing.T) {
	rig := testRig()
	eyes, fields := headsetEyes(graphics.IdentityQuaternion())
	covering, aspect := CoveringLens(rig, eyes, fields, 0.1, 500)
	middle := rig.EyePosition(graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0.2, 1.6, -0.3, 1}})
	step := covering.Position.Add(graphics.Vector{-middle[0], -middle[1], -middle[2], 0})
	// a few centimetres, times the rig's scale of 2
	if length := math.Sqrt(float64(step.Dot(step))); length > 0.2 {
		test.Fatalf("the covering lens is %v units from the middle of the eyes, want a few centimetres", length)
	}
	widest := 2 * math.Atan(math.Tan(0.85)) // the eyes see 0.85 radians down
	if float64(covering.FieldOfViewRadians) > widest*1.01 {
		test.Fatalf("field of view = %v, want about the eyes' %v", covering.FieldOfViewRadians, widest)
	}
	if aspect < 1 || aspect > 2 {
		test.Fatalf("aspect = %v, want a little wider than tall", aspect)
	}
}

func TestACoveringLensWithoutEyesLooksTheRigsWay(test *testing.T) {
	rig := Rig{Position: graphics.Vector{1, 2, 3, 1}}
	lens, aspect := CoveringLens(rig, nil, nil, 0.1, 100)
	if aspect != 1 || !sameSpot(lens.Position, rig.Position) || !sameSpot(lens.Target, graphics.Vector{1, 2, 2, 1}) {
		test.Fatalf("lens = %+v, aspect %v", lens, aspect)
	}
}

// insideClip tells whether a clip-space point is in the picture and between the clipping planes,
// with a hair of slack for rounding.
func insideClip(clip graphics.Vector) bool {
	limit := clip[3] * 1.00001
	return clip[3] > 0 && abs32(clip[0]) <= limit && abs32(clip[1]) <= limit && abs32(clip[2]) <= limit
}
