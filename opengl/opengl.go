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
}

// Draw a frame into the display
func (opengl *OpenGL) Draw(app *models.Application) error {

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

	// 1. Allocate a tightly packed byte slice for RGBA data
	// Width * Height * 4 channels (R, G, B, A)
	pixels := make([]byte, app.Width*app.Height*4)

	// Ensure OpenGL isn't assuming any row padding
	gl.PixelStorei(gl.PACK_ALIGNMENT, 1)

	// 2. Read the pixels from the frame buffer
	gl.ReadPixels(
		0, 0, // Start at the bottom-left corner (0,0)
		int32(app.Width), int32(app.Height),
		gl.RGBA,          // Format
		gl.UNSIGNED_BYTE, // Data type
		gl.Ptr(pixels),   // Destination pointer
	)

	// 3. Create a Go image.NRGBA object
	img := image.NewNRGBA(image.Rect(0, 0, app.Width, app.Height))

	// OpenGL's origin (0,0) is at the BOTTOM-left, but Go's image origin is TOP-left.
	// We need to flip the rows vertically while copying to the image buffer.
	stride := app.Width * 4
	for y := 0; y < app.Height; y++ {
		// OpenGL row index (bottom to top)
		glRow := y * stride
		// Go image row index (top to bottom)
		goRow := (app.Height - 1 - y) * stride

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
