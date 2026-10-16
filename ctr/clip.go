package ctr

import "github.com/akrck02/go-renderer/graphics"

// openGLToPICA turns OpenGL's clip space into the PICA200's: a quarter turn, because the screens
// are mounted on their side (x from y, y from -x), and depth from [-w, w] to [-w, 0]. sceneplan
// plans frames with OpenGL's matrices, so this goes in front of its view-projection.
var openGLToPICA = graphics.Mat4{
	0, -1, 0, 0, // column 0: x goes to -y
	1, 0, 0, 0, // column 1: y goes to x
	0, 0, 0.5, 0, // column 2: z is halved…
	0, 0, -0.5, 1, // column 3: …and moved down by half of w
}

// PICAViewProjection is an OpenGL view-projection matrix (sceneplan's) made ready for the PICA200.
func PICAViewProjection(openGL graphics.Mat4) graphics.Mat4 {
	return openGLToPICA.Multiply(openGL)
}
