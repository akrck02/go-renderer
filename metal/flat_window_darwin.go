package metal

import (
	"errors"
	"fmt"
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
	window   *glfw.Window
	renderer *FlatRenderer
	// unchangedFrames counts frames skipped because they would repeat the last one shown.
	unchangedFrames int
}

// Renderer is the window's renderer, for the application's Draw callback.
func (flat *FlatWindow) Renderer() *FlatRenderer { return flat.renderer }

// StartLoop opens the window and runs the application until it closes; a HeadlessApplication
// draws one frame into its CapturePath instead. It must be called on the program's main thread.
func (flat *FlatWindow) StartLoop(app *models.Application) error {
	if !windowloop.OnMainThread() {
		return errors.New("metal: StartLoop must run on the main thread; call runtime.LockOSThread in an init function of package main")
	}
	if err := glfw.Init(); err != nil {
		return err
	}
	defer glfw.Terminate()
	window, err := openWindow(app)
	if err != nil {
		return err
	}
	flat.window = window
	defer window.Destroy()
	renderer, err := NewFlatRenderer(window, !app.DisableVsync)
	if err != nil {
		return err
	}
	flat.renderer = renderer
	defer renderer.Close()

	if app.Input != nil {
		windowloop.ForwardInput(window, app.Input)
	}
	clock := windowloop.NewClock(app.FixedStep)
	if app.Type == models.HeadlessApplication {
		return flat.capture(app, clock)
	}
	return windowloop.Run(window, app, clock, func() error { return flat.drawFrame(app) })
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
func (flat *FlatWindow) drawFrame(app *models.Application) error {
	if app.Unchanged != nil && app.Unchanged(app) {
		flat.unchangedFrames++
		return nil
	}
	return app.Draw(app)
}

// capture runs a few update steps so the interface settles, then draws one frame off screen and
// saves it as a PNG.
func (flat *FlatWindow) capture(app *models.Application, clock *windowloop.Clock) error {
	flat.renderer.capturing = true
	for frame := 0; frame < 3; frame++ {
		glfw.PollEvents()
		if app.Input != nil {
			windowloop.UpdateSizes(flat.window, app.Input)
		}
		if err := windowloop.RunUpdateStep(app, clock.Step()); err != nil {
			return err
		}
		if err := app.Draw(app); err != nil {
			return err
		}
	}
	if flat.renderer.captured == nil {
		return errors.New("metal: the capture drew nothing")
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
	return png.Encode(file, flat.renderer.captured)
}
