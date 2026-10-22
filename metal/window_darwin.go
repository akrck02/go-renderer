package metal

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/windowloop"
	"github.com/go-gl/glfw/v3.4/glfw"
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

// StartLoop opens the window and runs the application until it closes; a HeadlessApplication
// draws one frame into its CapturePath instead. It must be called on the program's main thread.
func (flat *FlatWindow) StartLoop(app *models.Application) error {
	return flat.window.run(app, func(window *glfw.Window, vsync bool) (capturer, error) {
		renderer, err := NewFlatRenderer(window, vsync)
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

// StartLoop opens the window and runs the application until it closes; a HeadlessApplication
// draws one frame into its CapturePath instead. It must be called on the program's main thread.
func (scene *SceneWindow) StartLoop(app *models.Application) error {
	return scene.window.run(app, func(window *glfw.Window, vsync bool) (capturer, error) {
		renderer, err := NewSceneRenderer(window, vsync)
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
func (metal *metalWindow) run(app *models.Application, makeRenderer func(*glfw.Window, bool) (capturer, error)) error {
	if !windowloop.OnMainThread() {
		return windowloop.NotOnMainThread
	}
	if err := glfw.Init(); err != nil {
		return err
	}
	defer glfw.Terminate()
	window, err := openWindow(app)
	if err != nil {
		return err
	}
	metal.window = window
	defer window.Destroy()
	renderer, err := makeRenderer(window, !app.DisableVsync)
	if err != nil {
		return err
	}
	defer renderer.Close()

	if app.Input != nil {
		windowloop.ForwardInput(window, app.Input)
	}
	clock := windowloop.NewClock(app.FixedStep)
	if app.Type == models.HeadlessApplication {
		return metal.capture(app, clock, renderer)
	}
	return windowloop.Run(window, app, clock, func() error { return metal.drawFrame(app) })
}

// openWindow makes a window with no OpenGL context, for a Metal layer.
func openWindow(app *models.Application) (*glfw.Window, error) {
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.CocoaRetinaFramebuffer, glfw.True)
	glfw.WindowHint(glfw.CocoaGraphicsSwitching, glfw.True)
	if app.Type == models.HeadlessApplication {
		glfw.WindowHint(glfw.Visible, glfw.False)
	}
	window, err := glfw.CreateWindow(app.Width, app.Height, fmt.Sprintf("%s %s", app.Title, app.Version), nil, nil)
	if err != nil {
		return nil, err
	}
	if app.MinWidth > 0 || app.MinHeight > 0 {
		window.SetSizeLimits(max(app.MinWidth, glfw.DontCare), max(app.MinHeight, glfw.DontCare), glfw.DontCare, glfw.DontCare)
	}
	return window, nil
}

// drawFrame draws and shows a frame, unless the application says it would look like the last one.
func (metal *metalWindow) drawFrame(app *models.Application) error {
	if app.Unchanged != nil && app.Unchanged(app) {
		metal.unchangedFrames++
		return nil
	}
	return app.Draw(app)
}

// capture runs a few update steps so the application settles, drawing each frame off screen, and
// saves the last as a PNG.
func (metal *metalWindow) capture(app *models.Application, clock *windowloop.Clock, renderer capturer) error {
	renderer.startCapturing()
	for frame := 0; frame < 3; frame++ {
		glfw.PollEvents()
		if app.Input != nil {
			windowloop.UpdateSizes(metal.window, app.Input)
		}
		if err := windowloop.RunUpdateStep(app, clock.Step()); err != nil {
			return err
		}
		if err := app.Draw(app); err != nil {
			return err
		}
	}
	picture := renderer.capturedPicture()
	if picture == nil {
		return NothingCaptured
	}
	path := app.CapturePath
	if path == "" {
		path = "frame.png"
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, picture)
}
