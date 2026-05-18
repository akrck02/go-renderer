package models

type GraphicsApiBackend int

const (
	Software GraphicsApiBackend = 0
	OpenGL   GraphicsApiBackend = 1
	Vulkan   GraphicsApiBackend = 2
	Metal    GraphicsApiBackend = 3
	WebGPU   GraphicsApiBackend = 4
)

type GraphicsApi interface {
	StartLoop(app *Application) error
	Draw(app *Application) error
}
