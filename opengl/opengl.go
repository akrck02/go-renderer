package opengl

import (
	"fmt"
	"log"
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

func (opengl *OpenGL) Start(app *models.Application) {

	opengl.window = InitWindow(app)
	defer glfw.Terminate()
	opengl.init()

	opengl.frames = 0
	last_frame_time := time.Now()
	now := last_frame_time
	for !opengl.window.ShouldClose() {
		opengl.Draw(app)
		opengl.frames++

		if time.Since(last_frame_time).Milliseconds() >= 1000 {
			now = time.Now()
			opengl.window.SetTitle(fmt.Sprintf("%s %s   |   %d FPS", app.Title, app.Version, opengl.frames))
			last_frame_time = now
			opengl.frames = 0
		}
	}
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
func (opengl *OpenGL) Draw(app *models.Application) {

	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(opengl.program)

	app.Draw(app)

	glfw.PollEvents()
	opengl.window.SwapBuffers()
}
