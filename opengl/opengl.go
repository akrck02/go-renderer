package opengl

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"time"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.4/glfw"
)

type OpenGL struct {
	window  *glfw.Window
	frames  int
	program uint32
}

func (opengl *OpenGL) StartLoop(app *models.Application) error {

	opengl.window = InitWindow(app)
	defer glfw.Terminate()
	opengl.init()

	// Initialize camera if at zero
	if app.Camera.Up == (graphics.Vec4{}) {
		app.Camera.Up = graphics.Vec4{0, 1, 0, 0}
	}

	setInputs(opengl, app)

	last_frame_time := time.Now()
	now := last_frame_time

	var err error
	switch app.Type {
	case models.WindowApplication:

		for !opengl.window.ShouldClose() {

			err = opengl.Draw(app)
			if nil != err {
				return err
			}

			opengl.frames++

			if time.Since(last_frame_time).Milliseconds() >= 1000 {
				now = time.Now()
				opengl.window.SetTitle(fmt.Sprintf("%s %s   |   %d FPS", app.Title, app.Version, opengl.frames))
				last_frame_time = now
				opengl.frames = 0
			}
		}
	case models.HeadlessApplication:

		println("OpenGL does not support headless mode by default for now. Rendering frame with window spawn (slow).")

		err = opengl.Draw(app)
		if nil != err {
			return err
		}

		opengl.DrawOnDisk(app, "frame.png")

		duration := time.Since(last_frame_time)
		fmt.Printf("Frame generated in %d ms.\n", duration.Milliseconds())
	}

	return nil
}

func setInputs(opengl *OpenGL, app *models.Application) {
	opengl.window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Press || action == glfw.Repeat {
			speed := float32(0.1)
			rotSpeed := float32(0.05)
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

			// Rotation (Model Axis)
			case glfw.KeyW:
				app.Rotation[0] -= rotSpeed // Rotate around X
			case glfw.KeyS:
				app.Rotation[0] += rotSpeed
			case glfw.KeyA:
				app.Rotation[1] -= rotSpeed // Rotate around Y
			case glfw.KeyD:
				app.Rotation[1] += rotSpeed
			}
		}
	})
}

// init initializes OpenGL and links an initialized program.
func (opengl *OpenGL) init() {

	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	vertexShader, err := CompileShader(graphics.VertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragmentShader, err := CompileShader(graphics.FragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	opengl.program = gl.CreateProgram()
	gl.AttachShader(opengl.program, vertexShader)
	gl.AttachShader(opengl.program, fragmentShader)
	gl.LinkProgram(opengl.program)

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
}

// Draw a frame into the display
func (opengl *OpenGL) Draw(app *models.Application) error {

	w, h := app.Width, app.Height
	if nil != opengl.window {
		w, h = opengl.window.GetFramebufferSize()
	}

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

	if nil != opengl.window {
		glfw.PollEvents()
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

	// By default glReadPixels reads from the BACK buffer.
	// Since Draw() calls SwapBuffers(), the content is now in the FRONT buffer.
	gl.ReadBuffer(gl.FRONT)

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

	// 4. Encode and save to disk
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}
