package openxr

import (
	"github.com/akrck02/go-renderer/gles"
	"github.com/akrck02/go-renderer/graphics"
)

// Options are what a session is opened with.
type Options struct {
	ApplicationName string // what the runtime shows of the program; "Go Renderer" when empty
	// Passthrough asks to see the room behind the scene, on headsets whose runtime has
	// XR_FB_passthrough (a Meta Quest): the room shows wherever the eyes' picture is transparent,
	// so draw it with gles' SceneRenderer.SeeThrough. Session.ShowsRoom tells whether it is shown;
	// without the extension the session opens as usual, with nothing behind the scene.
	Passthrough bool
}

// Eye is one eye of a frame: where it is, what it sees, and where to draw it.
type Eye struct {
	Pose          graphics.Pose        // in the session's space: metres, the floor at y 0 when the runtime has one
	Field         graphics.FieldOfView // see graphics.PerspectiveFromFieldOfView
	Target        gles.Framebuffer     // colour and depth, Width × Height; 0 when the frame is not to be drawn
	Width, Height int
}

// Frame is one frame of a headset's: whether to draw it, when it will be shown, and the eyes.
type Frame struct {
	ShouldRender bool  // false: draw nothing, still give the frame back with EndFrame
	DisplayTime  int64 // when the runtime will show it, in its own nanoseconds (XrTime)
	Eyes         [2]Eye
	// Head is between the eyes, in the same space. It is never a zero rotation: the identity pose
	// (at the origin, looking down -Z) until the runtime knows where the head is.
	Head graphics.Pose
	// HeadTracked tells whether the runtime knows both where the head is and which way it looks
	// this frame. Before then (the headset just put on, tracking lost) Head is a guess or the
	// identity: do not place anything in the room by it.
	HeadTracked bool
}

// defaultApplicationName is what the runtime is told when the program does not say.
const defaultApplicationName = "Go Renderer"

// applicationName is the name the runtime is told.
func (options Options) applicationName() string {
	if options.ApplicationName == "" {
		return defaultApplicationName
	}
	return options.ApplicationName
}

// emptyFrame is a frame with nothing to draw, shown at a display time (0 when no frame was begun),
// its head at the identity pose.
func emptyFrame(displayTime int64) Frame {
	return Frame{DisplayTime: displayTime, Head: identityPose()}
}

// identityPose is at the origin and turned nowhere: a position is a point, so its w is 1.
func identityPose() graphics.Pose {
	return graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0, 0, 0, 1}}
}

// poseFrom turns OpenXR's pose (a rotation as x, y, z, w and a position in metres) into a
// graphics.Pose: the position is a point, so its w is 1.
func poseFrom(orientation [4]float32, position [3]float32) graphics.Pose {
	return graphics.Pose{
		Orientation: graphics.Quaternion{X: orientation[0], Y: orientation[1], Z: orientation[2], W: orientation[3]},
		Position:    graphics.Vector{position[0], position[1], position[2], 1},
	}
}

// fieldFrom turns OpenXR's field of view (angles from the eye's forward, left and down negative)
// into a graphics.FieldOfView, which keeps the same signs.
func fieldFrom(left, right, up, down float32) graphics.FieldOfView {
	return graphics.FieldOfView{LeftRadians: left, RightRadians: right, UpRadians: up, DownRadians: down}
}
