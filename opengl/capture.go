package opengl

import (
	"github.com/akrck02/go-renderer/failure"
	"image"
	"image/png"
	"os"

	"github.com/akrck02/go-renderer/frameloop"
	"github.com/akrck02/go-renderer/window"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// bytesPerPixel is the size of one red, green, blue and alpha pixel read back from OpenGL.
const bytesPerPixel = 4

// capture draws a Headless application's frame once it has settled and saves it.
func (opengl *Window) capture(app *window.Application, clock *frameloop.Clock) error {
	for range settlingFrames {
		glfw.PollEvents()
		if err := frameloop.RunUpdateStep(app, clock.Step()); err != nil {
			return err
		}
		if err := opengl.Draw(app); err != nil {
			return err
		}
	}
	opengl.capturing = true
	if err := opengl.Draw(app); err != nil {
		return err
	}
	return opengl.SaveFrame(app, app.CapturePathOrDefault())
}

// SaveFrame saves the frame just drawn (and not yet shown) as a PNG at path.
func (opengl *Window) SaveFrame(app *window.Application, path string) error {
	width, height := opengl.framebufferSize(app)
	picture := flipRows(readPixels(width, height), width, height)
	file, err := os.Create(path)
	if err != nil {
		return failure.Wrap(FrameNotSaved, path, err)
	}
	if err := png.Encode(file, picture); err != nil {
		file.Close() // the encoding error is the one to tell; the file is incomplete either way
		return failure.Wrap(FrameNotSaved, path, err)
	}
	if err := file.Close(); err != nil {
		return failure.Wrap(FrameNotSaved, path, err)
	}
	return nil
}

// readPixels reads the back buffer of the window, where the frame drawn without swapping still is,
// as tightly packed rows from the bottom up.
func readPixels(width, height int) []byte {
	pixels := make([]byte, width*height*bytesPerPixel)
	gl.Flush() // every command finished before the read
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, 0)
	gl.ReadBuffer(gl.BACK)
	gl.PixelStorei(gl.PACK_ALIGNMENT, 1) // no padding at the end of rows
	gl.ReadPixels(0, 0, int32(width), int32(height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
	return pixels
}

// flipRows turns OpenGL's rows, bottom first, into a picture's, top first, and makes it opaque:
// the window's alpha is not the picture's.
func flipRows(pixels []byte, width, height int) *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	stride := width * bytesPerPixel
	for row := range height {
		from := row * stride
		to := (height - 1 - row) * stride
		copy(picture.Pix[to:to+stride], pixels[from:from+stride])
	}
	for alpha := bytesPerPixel - 1; alpha < len(picture.Pix); alpha += bytesPerPixel {
		picture.Pix[alpha] = 255
	}
	return picture
}
