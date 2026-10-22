package opengl

import "github.com/akrck02/go-renderer/failure"

// Codes of the errors the OpenGL backend returns. Their details say what failed and, where
// OpenGL or GLFW said why, carry it.
var (
	// FramebufferIncomplete is a framebuffer OpenGL cannot draw into, such as the shadow map's on
	// a driver that lacks depth textures. Its details give OpenGL's status.
	FramebufferIncomplete = failure.Define("opengl.framebuffer-incomplete", "OpenGL cannot draw into the framebuffer")
	// WindowNotOpened is a window GLFW could not open: no display, or no OpenGL 4.1.
	WindowNotOpened = failure.Define("opengl.window-not-opened", "the window could not be opened")
	// ContextNotStarted is OpenGL's functions failing to load in the window's context.
	ContextNotStarted = failure.Define("opengl.context-not-started", "OpenGL could not start")
	// ShaderNotCompiled is a shader the driver would not compile. Its details give the driver's log.
	ShaderNotCompiled = failure.Define("opengl.shader-not-compiled", "OpenGL could not compile a shader")
	// ProgramNotLinked is two shaders the driver would not link. Its details give the driver's log.
	ProgramNotLinked = failure.Define("opengl.program-not-linked", "OpenGL could not link a shader program")
	// FrameNotSaved is a captured frame that could not be written to its file.
	FrameNotSaved = failure.Define("opengl.frame-not-saved", "the frame could not be saved")
	// NotImplemented is an immediate-mode shape the OpenGL renderer cannot draw yet.
	NotImplemented = failure.Define("opengl.not-implemented", "the OpenGL renderer cannot draw this shape yet")
)
