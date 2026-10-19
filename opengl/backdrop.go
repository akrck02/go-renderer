package opengl

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/go-gl/gl/v4.1-core/gl"
)

// backdropBlur takes the snapshots the overlay's backdrops read (see overlay.Snapshot): it copies
// the frame drawn so far, halves it a few times and blurs the smallest copy in two passes. The
// halving is what keeps it cheap: a 24-pixel blur runs on a picture a sixteenth of the screen's
// side, so the blur itself touches under half a percent of the pixels a full-size one
// would. Its textures are made once and remade only when the framebuffer changes size.
type backdropBlur struct {
	program    *shaderProgram
	emptyArray uint32   // the blur's triangle comes from gl_VertexID: no buffer, but a vertex array must be bound
	copy       target   // the frame, full size
	levels     []target // the halvings: each half the one before
	spare      target   // the smallest level's size: where the first blur pass writes
	width      int
	height     int
}

// target is a texture with a framebuffer to draw into it.
type target struct {
	framebuffer, texture uint32
	width, height        int32
}

func newBackdropBlur() (*backdropBlur, error) {
	program, err := newShaderProgram(graphics.BackdropVertexShader, graphics.BackdropBlurShader)
	if err != nil {
		return nil, err
	}
	blur := &backdropBlur{program: program}
	gl.GenVertexArrays(1, &blur.emptyArray)
	return blur, nil
}

// snapshot copies what the bound framebuffer shows (width × height pixels), blurs it by deviation
// screen pixels, and leaves the result bound on unit textureUnit. The framebuffer, viewport and
// blending are as they were when it returns.
func (blur *backdropBlur) snapshot(width, height int, deviation float32, textureUnit uint32) {
	var drawn int32
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &drawn)
	halvings := graphics.BackdropHalvings(deviation)
	blur.prepare(width, height, halvings)

	// A multisampled framebuffer can only be copied at its own size, which resolves it; any other
	// is halved straight from the frame, sparing a full-size copy: most of the snapshot's cost.
	source, sourceWidth, sourceHeight := uint32(drawn), int32(width), int32(height)
	if halvings == 0 || isMultisampled(source) {
		if blur.copy.texture == 0 {
			blur.copy = newTarget(int32(width), int32(height)) // made only when needed: it is the largest
		}
		copyInto(source, sourceWidth, sourceHeight, blur.copy, gl.NEAREST)
		source = blur.copy.framebuffer
	}
	smallest := blur.copy
	for _, level := range blur.levels[:halvings] {
		copyInto(source, sourceWidth, sourceHeight, level, gl.LINEAR) // four texels into one
		source, sourceWidth, sourceHeight = level.framebuffer, level.width, level.height
		smallest = level
	}
	blur.blurInPlace(smallest, deviation/float32(int(1)<<halvings))

	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(drawn))
	gl.Viewport(0, 0, int32(width), int32(height))
	gl.ActiveTexture(gl.TEXTURE0 + textureUnit)
	gl.BindTexture(gl.TEXTURE_2D, smallest.texture)
}

// isMultisampled tells whether a framebuffer keeps several samples a pixel.
func isMultisampled(framebuffer uint32) bool {
	var samples int32
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, framebuffer)
	gl.GetIntegerv(gl.SAMPLES, &samples)
	return samples > 1
}

// copyInto copies a framebuffer's whole picture into a target, scaled to its size.
func copyInto(source uint32, width, height int32, into target, filter uint32) {
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, source)
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, into.framebuffer)
	gl.BlitFramebuffer(0, 0, width, height, 0, 0, into.width, into.height, gl.COLOR_BUFFER_BIT, filter)
}

// blurInPlace blurs a target across, into the spare, then down, back into the target.
func (blur *backdropBlur) blurInPlace(level target, deviation float32) {
	weights := graphics.BackdropWeights(deviation)
	program := blur.program
	gl.UseProgram(program.handle)
	gl.Uniform1fv(program.location("weights"), int32(len(weights)), &weights[0])
	program.setInteger("source", 0)
	gl.Disable(gl.BLEND)
	gl.Disable(gl.DEPTH_TEST)
	gl.BindVertexArray(blur.emptyArray)
	gl.Viewport(0, 0, level.width, level.height)
	gl.ActiveTexture(gl.TEXTURE0)
	blur.pass(level, blur.spare, 1/float32(level.width), 0)
	blur.pass(blur.spare, level, 0, 1/float32(level.height))
	gl.BindVertexArray(0)
}

func (blur *backdropBlur) pass(from, into target, stepX, stepY float32) {
	gl.BindFramebuffer(gl.FRAMEBUFFER, into.framebuffer)
	gl.BindTexture(gl.TEXTURE_2D, from.texture)
	gl.Uniform2f(blur.program.location("step"), stepX, stepY)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)
}

// prepare makes the textures for a framebuffer of this size, when it changed, and enough halvings.
func (blur *backdropBlur) prepare(width, height, halvings int) {
	if width != blur.width || height != blur.height {
		blur.release()
		blur.width, blur.height = width, height
	}
	for len(blur.levels) < halvings {
		beforeWidth, beforeHeight := int32(width), int32(height)
		if len(blur.levels) > 0 {
			before := blur.levels[len(blur.levels)-1]
			beforeWidth, beforeHeight = before.width, before.height
		}
		blur.levels = append(blur.levels, newTarget(max(beforeWidth/2, 1), max(beforeHeight/2, 1)))
	}
	smallest := target{width: int32(width), height: int32(height)}
	if halvings > 0 {
		smallest = blur.levels[halvings-1]
	}
	if blur.spare.width != smallest.width || blur.spare.height != smallest.height {
		deleteTarget(blur.spare)
		blur.spare = newTarget(smallest.width, smallest.height)
	}
}

func (blur *backdropBlur) release() {
	deleteTarget(blur.copy)
	for _, level := range blur.levels {
		deleteTarget(level)
	}
	deleteTarget(blur.spare)
	blur.copy, blur.levels, blur.spare = target{}, nil, target{}
}

// newTarget makes an RGBA texture that blends smoothly when read between texels, and its framebuffer.
func newTarget(width, height int32) target {
	made := target{width: width, height: height}
	gl.GenTextures(1, &made.texture)
	gl.BindTexture(gl.TEXTURE_2D, made.texture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.GenFramebuffers(1, &made.framebuffer)
	gl.BindFramebuffer(gl.FRAMEBUFFER, made.framebuffer)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, made.texture, 0)
	return made
}

func deleteTarget(made target) {
	if made.texture == 0 {
		return
	}
	gl.DeleteFramebuffers(1, &made.framebuffer)
	gl.DeleteTextures(1, &made.texture)
}
