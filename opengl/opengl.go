package opengl

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"time"

	"github.com/akrck02/go-renderer/frameloop"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/windowloop"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.4/glfw"
)

type OpenGL struct {
	window  *glfw.Window
	program uint32
	// capturing keeps the last frame in the back buffer (no swap) so it can be read: after a swap
	// the front buffer is undefined on some platforms (Wayland), and reading it gives an empty image
	capturing bool
	// unchangedFrames counts frames skipped because they would repeat the last one shown.
	unchangedFrames int
}

// StartLoop opens the window and runs the application until it closes. It must be called on the
// program's main thread (macOS only delivers window events there): lock it with
// runtime.LockOSThread in an init function of package main.
func (opengl *OpenGL) StartLoop(app *models.Application) error {
	if !windowloop.OnMainThread() {
		return windowloop.NotOnMainThread
	}

	opengl.window = InitWindow(app)
	defer glfw.Terminate()
	opengl.init(app)

	// Initialize state
	if app.Camera.Up == (graphics.Vec4{}) {
		app.Camera.Up = graphics.Vec4{0, 1, 0, 0}
	}
	if app.ModelMatrix == (graphics.Mat4{}) {
		app.ModelMatrix = graphics.Identity()
	}

	if app.Input != nil {
		windowloop.ForwardInput(opengl.window, app.Input)
	} else {
		setInputs(opengl, app)
	}
	clock := frameloop.NewClock(app.FixedStep)
	started := time.Now()

	switch app.Type {
	case models.WindowApplication:
		return windowloop.Run(opengl.window, app, clock, func() error { return opengl.Draw(app) })
	case models.HeadlessApplication:

		println("OpenGL does not support headless mode by default for now. Rendering frame with window spawn (slow).")

		if err := opengl.renderSettledFrame(app, clock); err != nil {
			return err
		}
		opengl.capturing = true
		if err := opengl.Draw(app); err != nil {
			return err
		}
		if err := opengl.DrawOnDisk(app, capturePath(app)); err != nil {
			return err
		}

		duration := time.Since(started)
		fmt.Printf("Frame generated in %d ms.\n", duration.Milliseconds())
	}

	return nil
}

func setInputs(opengl *OpenGL, app *models.Application) {
	opengl.window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Press || action == glfw.Repeat {
			speed := float32(0.1)
			rotSpeed := float32(0.05)

			// Get camera axes for relative movement/rotation
			view := graphics.LookAt(app.Camera.Position, app.Camera.Target, app.Camera.Up)
			right := graphics.Vec4{view[0], view[4], view[8], 0} // First row of View is Right in World Space
			up := graphics.Vec4{view[1], view[5], view[9], 0}    // Second row of View is Up in World Space

			switch key {
			// Movement
			case glfw.KeyUp:
				app.Camera.Position[1] += speed
				app.Camera.Target[1] += speed
			case glfw.KeyDown:
				app.Camera.Position[1] -= speed
				app.Camera.Target[1] -= speed
			case glfw.KeyLeft:
				app.Camera.Position[0] -= speed
				app.Camera.Target[0] -= speed
			case glfw.KeyRight:
				app.Camera.Position[0] += speed
				app.Camera.Target[0] += speed

			// Rotation (Model Axis) - Now relative to camera view
			case glfw.KeyW:
				rot := graphics.Rotate(right, float64(-rotSpeed))
				app.ModelMatrix = rot.Multiply(app.ModelMatrix)
			case glfw.KeyS:
				rot := graphics.Rotate(right, float64(rotSpeed))
				app.ModelMatrix = rot.Multiply(app.ModelMatrix)
			case glfw.KeyA:
				rot := graphics.Rotate(up, float64(-rotSpeed))
				app.ModelMatrix = rot.Multiply(app.ModelMatrix)
			case glfw.KeyD:
				rot := graphics.Rotate(up, float64(rotSpeed))
				app.ModelMatrix = rot.Multiply(app.ModelMatrix)
			}
		}
	})
}

func (opengl *OpenGL) init(app *models.Application) {

	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)
	if app.FlatOnly {
		return // the flat renderer brings its own shaders and needs no depth test
	}

	vertexShader, err := CompileShader(graphics.VertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragmentShader, err := CompileShader(graphics.FragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	opengl.program, err = LinkProgram(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
}

var firstFrame = true

// Draw a frame into the display
func (opengl *OpenGL) Draw(app *models.Application) error {

	if firstFrame {
		fmt.Println("OpenGL: Rendering first frame...")
		firstFrame = false
	}

	presenting := nil != opengl.window && !opengl.capturing
	if presenting && app.Unchanged != nil && app.Unchanged(app) {
		opengl.unchangedFrames++
		return nil // the window already shows this picture
	}
	if app.FlatOnly {
		return opengl.drawFlat(app, presenting)
	}

	w, h := app.Width, app.Height
	if nil != opengl.window {
		w, h = opengl.window.GetFramebufferSize()
	}

	// Update matrices
	app.ProjectionMatrix = app.Camera.ProjectionMatrix(float64(w) / float64(h))
	app.ViewMatrix = graphics.LookAt(app.Camera.Position, app.Camera.Target, app.Camera.Up)
	app.Renderer.SetMatrices(app.ProjectionMatrix, app.ViewMatrix, app.ModelMatrix)

	gl.Viewport(0, 0, int32(w), int32(h))

	gl.ClearColor(
		app.Background[0],
		app.Background[1],
		app.Background[2],
		app.Background[3],
	)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(opengl.program)

	err := app.Draw(app)
	if nil != err {
		return err
	}

	if nil != opengl.window && !opengl.capturing {
		opengl.window.SwapBuffers()
	}

	return nil
}

// drawFlat draws a frame of a flat-only application: its Draw callback clears and paints the whole
// window (see FlatRenderer), so the 3D renderer's matrices, clear and program are left out.
func (opengl *OpenGL) drawFlat(app *models.Application, presenting bool) error {
	if err := app.Draw(app); err != nil {
		return err
	}
	if presenting {
		opengl.window.SwapBuffers()
	}
	return nil
}

// SaveFrameToDisk captures the current OpenGL framebuffer and saves it as a PNG.
func (opengl *OpenGL) DrawOnDisk(app *models.Application, filePath string) error {

	w, h := app.Width, app.Height
	if nil != opengl.window {
		w, h = opengl.window.GetFramebufferSize()
	}

	// 1. Allocate a tightly packed byte slice for RGBA data
	// Width * Height * 4 channels (R, G, B, A)
	pixels := make([]byte, w*h*4)

	// Ensure all OpenGL commands are finished before reading pixels
	gl.Flush()

	// The capture frame was drawn without swapping, so it is still in the back buffer of the
	// default framebuffer.
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, 0)
	gl.ReadBuffer(gl.BACK)

	// Ensure OpenGL isn't assuming any row padding
	gl.PixelStorei(gl.PACK_ALIGNMENT, 1)

	// 2. Read the pixels from the frame buffer
	gl.ReadPixels(
		0, 0, // Start at the bottom-left corner (0,0)
		int32(w), int32(h),
		gl.RGBA,          // Format
		gl.UNSIGNED_BYTE, // Data type
		gl.Ptr(pixels),   // Destination pointer
	)

	// Restore default read buffer
	gl.ReadBuffer(gl.BACK)

	// 3. Create a Go image.NRGBA object
	img := image.NewNRGBA(image.Rect(0, 0, w, h))

	// OpenGL's origin (0,0) is at the BOTTOM-left, but Go's image origin is TOP-left.
	// We need to flip the rows vertically while copying to the image buffer.
	stride := w * 4
	for y := 0; y < h; y++ {
		// OpenGL row index (bottom to top)
		glRow := y * stride
		// Go image row index (top to bottom)
		goRow := (h - 1 - y) * stride

		copy(img.Pix[goRow:goRow+stride], pixels[glRow:glRow+stride])
	}
	// the window's alpha is not the picture's: the frame is opaque
	for index := 3; index < len(img.Pix); index += 4 {
		img.Pix[index] = 255
	}

	// 4. Encode and save to disk
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

// renderSettledFrame draws a few frames so that every buffer is uploaded and the camera settled.
func (opengl *OpenGL) renderSettledFrame(app *models.Application, clock *frameloop.Clock) error {
	for frame := 0; frame < 3; frame++ {
		glfw.PollEvents()
		if err := frameloop.RunUpdateStep(app, clock.Step()); err != nil {
			return err
		}
		if err := opengl.Draw(app); err != nil {
			return err
		}
	}
	return nil
}

func capturePath(app *models.Application) string {
	if app.CapturePath == "" {
		return "frame.png"
	}
	return app.CapturePath
}

// Wake makes a window that draws on demand draw a frame now: for a program whose data changes
// on another goroutine (a download finishing, a message arriving). Safe from any goroutine.
func Wake() { windowloop.Wake() }
