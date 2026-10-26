package openxr

import (
	"testing"

	"github.com/akrck02/go-renderer/graphics"
)

func TestAPoseKeepsOpenXRsOrderAndIsAPoint(test *testing.T) {
	pose := poseFrom([4]float32{0.5, -0.5, 0.25, 0.75}, [3]float32{1, 2, -3})
	if pose.Orientation != (graphics.Quaternion{X: 0.5, Y: -0.5, Z: 0.25, W: 0.75}) {
		test.Fatalf("orientation %v", pose.Orientation)
	}
	if pose.Position != (graphics.Vector{1, 2, -3, 1}) {
		test.Fatalf("position %v", pose.Position)
	}
}

func TestAFieldOfViewKeepsItsSigns(test *testing.T) {
	field := fieldFrom(-0.75, 0.5, 0.5, -0.75)
	if field.LeftRadians != -0.75 || field.RightRadians != 0.5 || field.UpRadians != 0.5 || field.DownRadians != -0.75 {
		test.Fatalf("field %+v", field)
	}
}

func TestTheApplicationIsNamedWhenTheProgramDoesNotSay(test *testing.T) {
	if (Options{}).applicationName() != defaultApplicationName {
		test.Fatal("no default name")
	}
	if (Options{ApplicationName: "Example"}).applicationName() != "Example" {
		test.Fatal("the program's name was not kept")
	}
}

func TestAnEmptyFrameLooksAheadFromTheOrigin(test *testing.T) {
	frame := emptyFrame(5)
	if frame.Head.Orientation != graphics.IdentityQuaternion() {
		test.Fatalf("an empty frame's head turns by %v, want the identity", frame.Head.Orientation)
	}
	if frame.Head.Position != (graphics.Vector{0, 0, 0, 1}) {
		test.Fatalf("an empty frame's head is at %v", frame.Head.Position)
	}
	if frame.HeadTracked || frame.ShouldRender || frame.DisplayTime != 5 {
		test.Fatalf("empty frame %+v", frame)
	}
}
