package opengl

import (
	"github.com/akrck02/go-renderer/failure"
	"github.com/akrck02/go-renderer/frameloop"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/window"
	"github.com/akrck02/go-renderer/windowloop"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// settlingFrames is how many frames a Headless application draws before the one it saves: enough
// for every buffer to be uploaded and the camera to settle.
const settlingFrames = 3

// Window runs a window.Application in an OpenGL 4.1 window. Its zero value is ready to use.
type Window struct {
	window *glfw.Window
	// program draws the immediate-mode shapes; 0 for a FlatOnly application.
	program uint32
	// capturing keeps the last frame in the back buffer (no swap) so it can be read: after a swap
	// the front buffer is undefined on some platforms (Wayland), and reading it gives an empty image
	capturing bool
	// unchangedFrames counts frames skipped because they would repeat the last one shown.
	unchangedFrames int
}

// Run opens the window and runs the application until it closes or sets Stop; a Headless
// application draws one frame into a hidden window and saves it. It must be called on the
// program's main thread (macOS only delivers window events there): lock it with
// runtime.LockOSThread in an init function of package main.
func (opengl *Window) Run(app *window.Application) error {
	if !windowloop.OnMainThread() {
		return windowloop.NotOnMainThread
	}
	glfwWindow, err := openWindow(app)
	if err != nil {
		return err
	}
	opengl.window = glfwWindow
	defer glfw.Terminate()
	if err := opengl.start(app); err != nil {
		return err
	}
	fillDefaults(app)
	if app.Input != nil {
		windowloop.ForwardInput(opengl.window, app.Input)
	}
	clock := frameloop.NewClock(app.FixedStepSeconds)
	if app.Kind == window.Headless {
		return opengl.capture(app, clock)
	}
	return windowloop.Run(opengl.window, app, clock, func() error { return opengl.Draw(app) })
}

// fillDefaults gives the application what it needs to draw and left unset.
func fillDefaults(app *window.Application) {
	if app.Renderer == nil {
		app.Renderer = &Renderer{}
	}
	if app.Camera.Up == (graphics.Vector{}) {
		app.Camera.Up = graphics.Vector{0, 1, 0, 0}
	}
	if app.ModelMatrix == (graphics.Matrix{}) {
		app.ModelMatrix = graphics.Identity()
	}
}

// start loads OpenGL's functions and, unless the application is flat only, the immediate-mode
// shaders.
func (opengl *Window) start(app *window.Application) error {
	if err := gl.Init(); err != nil {
		return failure.Wrap(ContextNotStarted, "loading OpenGL's functions", err)
	}
	if app.FlatOnly {
		return nil // the flat renderer brings its own shaders and needs no depth test
	}
	program, err := newProgram(immediateVertexShader, immediateFragmentShader)
	if err != nil {
		return err
	}
	opengl.program = program
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	return nil
}

// Draw draws a frame: the immediate-mode matrices from the application's camera, a clear to its
// background, then its Draw callback. A frame the application says is Unchanged is skipped.
func (opengl *Window) Draw(app *window.Application) error {
	presenting := opengl.window != nil && !opengl.capturing
	if presenting && app.Unchanged != nil && app.Unchanged(app) {
		opengl.unchangedFrames++
		return nil // the window already shows this picture
	}
	if app.FlatOnly {
		return opengl.drawFlat(app, presenting)
	}

	width, height := opengl.framebufferSize(app)
	app.ProjectionMatrix = app.Camera.ProjectionMatrix(float64(width) / float64(height))
	app.ViewMatrix = graphics.LookAt(app.Camera.Position, app.Camera.Target, app.Camera.Up)
	app.Renderer.SetMatrices(app.ProjectionMatrix, app.ViewMatrix, app.ModelMatrix)

	gl.Viewport(0, 0, int32(width), int32(height))
	gl.ClearColor(app.Background[0], app.Background[1], app.Background[2], app.Background[3])
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(opengl.program)

	if app.Callbacks.Draw != nil {
		if err := app.Callbacks.Draw(app); err != nil {
			return err
		}
	}
	if presenting {
		opengl.window.SwapBuffers()
	}
	return nil
}

// drawFlat draws a frame of a flat-only application: its Draw callback clears and paints the whole
// window (see FlatRenderer), so the 3D renderer's matrices, clear and program are left out.
func (opengl *Window) drawFlat(app *window.Application, presenting bool) error {
	if app.Callbacks.Draw != nil {
		if err := app.Callbacks.Draw(app); err != nil {
			return err
		}
	}
	if presenting {
		opengl.window.SwapBuffers()
	}
	return nil
}

// framebufferSize is the size in pixels of what is drawn: the window's framebuffer, or the
// application's size before a window exists.
func (opengl *Window) framebufferSize(app *window.Application) (width, height int) {
	if opengl.window == nil {
		return app.Width, app.Height
	}
	return opengl.window.GetFramebufferSize()
}

// Wake makes a window that draws on demand draw a frame now: for a program whose data changes
// on another goroutine (a download finishing, a message arriving). Safe from any goroutine.
func Wake() { windowloop.Wake() }
