package opengl

import "github.com/akrck02/go-renderer/failure"

// FramebufferIncomplete is a framebuffer OpenGL cannot draw into, such as the shadow map's on a
// driver that lacks depth textures. Its details give OpenGL's status.
var FramebufferIncomplete = failure.Define("opengl.framebuffer-incomplete", "OpenGL cannot draw into the framebuffer")
