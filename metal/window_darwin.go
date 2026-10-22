package metal

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/akrck02/go-renderer/frameloop"
	"github.com/akrck02/go-renderer/window"
	"github.com/akrck02/go-renderer/windowloop"
)

// FlatWindow runs an application that only draws flat shapes (an interface) through Metal, with
// the same loop as the OpenGL backend: drawing on demand, the fixed-step clock, the resize
// follower. The application's Draw callback paints with Renderer.
type FlatWindow struct {
	renderer *FlatRenderer
	window   metalWindow
}

// Renderer is the window's renderer, for the application's Draw callback.
func (flat *FlatWindow) Renderer() *FlatRenderer { return flat.renderer }

// Run opens the window and runs the application until it closes; a Headless application
// draws one frame into its CapturePath instead. It must be called on the program's main thread.
func (flat *FlatWindow) Run(app *window.Application) error {
	return flat.window.run(app, func(glfwWindow *glfw.Window, vsync bool) (capturer, error) {
		renderer, err := NewFlatRenderer(glfwWindow, vsync)
		flat.renderer = renderer
		return renderer, err
	})
}

// SceneWindow runs an application that draws a 3D scene through Metal (the viewer, for one),
// with the same loop as the OpenGL backend. The application's Draw callback draws with Renderer.
type SceneWindow struct {
	renderer *SceneRenderer
	window   metalWindow
}

// Renderer is the window's renderer, for the application's Draw callback.
func (scene *SceneWindow) Renderer() *SceneRenderer { return scene.renderer }

// Run opens the window and runs the application until it closes; a Headless application
// draws one frame into its CapturePath instead. It must be called on the program's main thread.
func (scene *SceneWindow) Run(app *window.Application) error {
	return scene.window.run(app, func(glfwWindow *glfw.Window, vsync bool) (capturer, error) {
		renderer, err := NewSceneRenderer(glfwWindow, vsync)
		scene.renderer = renderer
		return renderer, err
	})
}

// capturer is a renderer that can draw its frames off screen, for captures.
type capturer interface {
	startCapturing()
	capturedPicture() *image.NRGBA
	Close()
}

func (renderer *FlatRenderer) startCapturing()               { renderer.capturing = true }
func (renderer *FlatRenderer) capturedPicture() *image.NRGBA { return renderer.captured }

func (renderer *SceneRenderer) startCapturing()               { renderer.capturing = true }
func (renderer *SceneRenderer) capturedPicture() *image.NRGBA { return renderer.captured }

// metalWindow is a GLFW window with no OpenGL context, run by the shared window loop.
type metalWindow struct {
	window *glfw.Window
	// unchangedFrames counts frames skipped because they would repeat the last one shown.
	unchangedFrames int
}

// run opens the window, makes its renderer and runs the loop, or captures one frame.
func (metal *metalWindow) run(app *window.Application, makeRenderer func(*glfw.Window, bool) (capturer, error)) error {
	if !windowloop.OnMainThread() {
		return windowloop.NotOnMainThread
	}
	if err := glfw.Init(); err != nil {
		return err
	}
	defer glfw.Terminate()
	glfwWindow, err := openWindow(app)
	if err != nil {
		return err
	}
	metal.window = glfwWindow
	defer glfwWindow.Destroy()
	renderer, err := makeRenderer(glfwWindow, !app.DisableVsync)
	if err != nil {
		return err
	}
	defer renderer.Close()

	if app.Input != nil {
		windowloop.ForwardInput(glfwWindow, app.Input)
	}
	clock := frameloop.NewClock(app.FixedStepSeconds)
	if app.Kind == window.Headless {
		return metal.capture(app, clock, renderer)
	}
	return windowloop.Run(glfwWindow, app, clock, func() error { return metal.drawFrame(app) })
}

// openWindow makes a window with no OpenGL context, for a Metal layer.
func openWindow(app *window.Application) (*glfw.Window, error) {
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.CocoaRetinaFramebuffer, glfw.True)
	glfw.WindowHint(glfw.CocoaGraphicsSwitching, glfw.True)
	if app.Kind == window.Headless {
		glfw.WindowHint(glfw.Visible, glfw.False)
	}
	glfwWindow, err := glfw.CreateWindow(app.Width, app.Height, fmt.Sprintf("%s %s", app.Title, app.Version), nil, nil)
	if err != nil {
		return nil, err
	}
	if app.MinWidth > 0 || app.MinHeight > 0 {
		glfwWindow.SetSizeLimits(max(app.MinWidth, glfw.DontCare), max(app.MinHeight, glfw.DontCare), glfw.DontCare, glfw.DontCare)
	}
	return glfwWindow, nil
}

// drawFrame draws and shows a frame, unless the application says it would look like the last one.
func (metal *metalWindow) drawFrame(app *window.Application) error {
	if app.Unchanged != nil && app.Unchanged(app) {
		metal.unchangedFrames++
		return nil
	}
	return drawApplication(app)
}

// drawApplication calls the application's Draw callback, if it has one.
func drawApplication(app *window.Application) error {
	if app.Callbacks.Draw == nil {
		return nil
	}
	return app.Callbacks.Draw(app)
}

// capture runs a few update steps so the application settles, drawing each frame off screen, and
// saves the last as a PNG.
func (metal *metalWindow) capture(app *window.Application, clock *frameloop.Clock, renderer capturer) error {
	renderer.startCapturing()
	for frame := 0; frame < 3; frame++ {
		glfw.PollEvents()
		if app.Input != nil {
			windowloop.UpdateSizes(metal.window, app.Input)
		}
		if err := frameloop.RunUpdateStep(app, clock.Step()); err != nil {
			return err
		}
		if err := drawApplication(app); err != nil {
			return err
		}
	}
	picture := renderer.capturedPicture()
	if picture == nil {
		return NothingCaptured
	}
	path := app.CapturePathOrDefault()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, picture)
}
