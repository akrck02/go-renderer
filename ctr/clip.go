package ctr

import (
	"github.com/akrck02/go-renderer/camera"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/internal/sceneplan"
)

// openGLToPICA turns OpenGL's clip space into the PICA200's: a quarter turn, because the screens
// are mounted on their side (x from y, y from -x), and depth from [-w, w] to [-w, 0]. sceneplan
// plans frames with OpenGL's matrices, so this goes in front of its view-projection.
var openGLToPICA = graphics.Matrix{
	0, -1, 0, 0, // column 0: x goes to -y
	1, 0, 0, 0, // column 1: y goes to x
	0, 0, 0.5, 0, // column 2: z is halved…
	0, 0, -0.5, 1, // column 3: …and moved down by half of w
}

// PICAViewProjection is an OpenGL view-projection matrix (sceneplan's) made ready for the PICA200.
func PICAViewProjection(openGL graphics.Matrix) graphics.Matrix {
	return openGLToPICA.Multiply(openGL)
}

// Eye is one of the two pictures of the top screen in 3D.
type Eye int

// The two eyes of the top screen in 3D.
const (
	LeftEye Eye = iota
	RightEye
)

// side is -1 for the left eye and 1 for the right one.
func (eye Eye) side() float32 {
	if eye == LeftEye {
		return -1
	}
	return 1
}

// EyeCamera is the camera for one eye: moved sideways by half the separation, looking the same
// way (the two views stay parallel; EyeViewProjection makes them meet at the focal distance).
func EyeCamera(camera camera.Camera, eye Eye, separation float32) camera.Camera {
	right := cameraRight(camera)
	shift := eye.side() * separation / 2
	for axis := range 3 {
		camera.Position[axis] += right[axis] * shift
		camera.Target[axis] += right[axis] * shift
	}
	return camera
}

// EyeViewProjection is an eye's OpenGL view-projection: its camera's, with the picture moved
// sideways so that what lies at the focal distance falls on the same place for both eyes, on the
// plane of the screen; nearer comes out of it, farther goes in.
func EyeViewProjection(camera camera.Camera, eye Eye, separation, focalDistance float32, width, height int) graphics.Matrix {
	matrices := sceneplan.ComputeCameraMatrices(EyeCamera(camera, eye, separation), width, height)
	// A point straight ahead at the focal distance is seen by this eye half the separation to
	// the other side; moving clip x by this share of w puts it back in the middle.
	shift := eye.side() * separation / 2 * matrices.Projection[0] / focalDistance
	moveSideways := graphics.Identity()
	moveSideways[12] = shift // x += shift × w: column 3, row 0
	return moveSideways.Multiply(matrices.Projection).Multiply(matrices.View)
}

// cameraRight is the camera's right, of unit length.
func cameraRight(camera camera.Camera) graphics.Vector {
	forward := graphics.Vector{camera.Target[0] - camera.Position[0], camera.Target[1] - camera.Position[1], camera.Target[2] - camera.Position[2], 0}
	return forward.Cross(camera.Up).Normalize()
}
