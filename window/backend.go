package window

// Backend opens a window (or a hidden one, for a Headless application) and drives an Application
// in it: opengl.Window, metal.Window and the software, Vulkan and WebGPU stubs.
type Backend interface {
	// Run opens the window and runs the application until it closes or sets Stop.
	Run(app *Application) error
	// Draw draws one frame of the application.
	Draw(app *Application) error
}
