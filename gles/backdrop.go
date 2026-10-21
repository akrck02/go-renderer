package gles

import "github.com/akrck02/go-renderer/graphics"

// blurredTextureUnit is where the blurred snapshot is bound for the quads that read it.
const blurredTextureUnit = 2

// backdropBlur takes the snapshots the overlay's backdrops read (see overlay.Snapshot), with what
// OpenGL ES 2 has: it copies the frame into a texture, halves it a few times by drawing it into
// smaller ones (each texel read between four, which averages them), and blurs the smallest in two
// passes. The halving keeps it cheap: a 24-pixel blur runs on a picture a sixteenth of the
// screen's side. Its textures are made once and remade only when the screen changes size.
type backdropBlur struct {
	functions Functions
	program   *program
	corners   Buffer   // two triangles over the whole target
	copy      target   // the frame, full size: OpenGL ES 2 cannot copy and shrink at once
	levels    []target // the halvings: each half the one before
	spare     target   // the smallest level's size: where the first blur pass writes
	width     int
	height    int
}

// target is a texture with a framebuffer to draw into it.
type target struct {
	framebuffer   Framebuffer
	texture       Texture
	width, height int
}

// cornerFloats are the two triangles' corners, from 0 to 1.
var cornerFloats = []float32{0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1}

func newBackdropBlur(functions Functions) (*backdropBlur, error) {
	shaders, err := newProgram(functions, backdropVertexShader, backdropBlurShader, backdropAttributes)
	if err != nil {
		return nil, err
	}
	blur := &backdropBlur{functions: functions, program: shaders, corners: functions.CreateBuffer()}
	functions.BindBuffer(ArrayBuffer, blur.corners)
	functions.BufferData(ArrayBuffer, floatBytes(cornerFloats), StaticDraw)
	return blur, nil
}

// snapshot copies the target framebuffer (width × height pixels), blurs it by deviation pixels,
// and leaves the result bound on blurredTextureUnit. The target is bound again when it returns;
// the caller sets its own program, buffer and textures up again.
func (blur *backdropBlur) snapshot(target Framebuffer, width, height int, deviation float32) {
	functions := blur.functions
	halvings := graphics.BackdropHalvings(deviation)
	blur.prepare(width, height, halvings)

	functions.ActiveTexture(Texture0)
	functions.BindTexture(Texture2D, blur.copy.texture)
	functions.CopyTexSubImage2D(Texture2D, 0, 0, 0, 0, 0, width, height)

	blur.program.use()
	blur.program.setInteger("source", 0)
	functions.Disable(Blend)
	functions.Disable(DepthTest)
	functions.BindBuffer(ArrayBuffer, blur.corners)
	functions.VertexAttribPointer(0, 2, Float, false, 2*4, 0)
	smallest := blur.copy
	for _, level := range blur.levels[:halvings] {
		blur.pass(smallest, level, [graphics.BackdropTaps + 1]float32{1}, 0, 0)
		smallest = level
	}
	weights := graphics.BackdropWeights(deviation / float32(int(1)<<halvings))
	blur.pass(smallest, blur.spare, weights, 1/float32(smallest.width), 0)
	blur.pass(blur.spare, smallest, weights, 0, 1/float32(smallest.height))

	functions.BindFramebuffer(FramebufferTarget, target)
	functions.Viewport(0, 0, width, height)
	functions.ActiveTexture(Texture0 + blurredTextureUnit)
	functions.BindTexture(Texture2D, smallest.texture)
}

// pass draws a texture into a target through the blur shader, with these weights along step.
func (blur *backdropBlur) pass(from, into target, weights [graphics.BackdropTaps + 1]float32, stepX, stepY float32) {
	functions := blur.functions
	functions.BindFramebuffer(FramebufferTarget, into.framebuffer)
	functions.Viewport(0, 0, into.width, into.height)
	functions.ActiveTexture(Texture0)
	functions.BindTexture(Texture2D, from.texture)
	blur.program.setVector2("step", stepX, stepY)
	for tap, weight := range weights {
		blur.program.setFloat(weightNames[tap], weight)
	}
	functions.DrawArrays(Triangles, 0, len(cornerFloats)/2)
}

// weightNames are the blur's weights as OpenGL ES 2 names each element of the array.
var weightNames = [graphics.BackdropTaps + 1]string{"weights[0]", "weights[1]", "weights[2]", "weights[3]", "weights[4]"}

// prepare makes the textures for a screen of this size, when it changed, and enough halvings.
func (blur *backdropBlur) prepare(width, height, halvings int) {
	if width != blur.width || height != blur.height {
		blur.release()
		blur.width, blur.height = width, height
		blur.copy = newTarget(blur.functions, width, height, RGB) // a screen may have no alpha to copy, and none is needed
	}
	for len(blur.levels) < halvings {
		before := blur.copy
		if len(blur.levels) > 0 {
			before = blur.levels[len(blur.levels)-1]
		}
		blur.levels = append(blur.levels, newTarget(blur.functions, max(before.width/2, 1), max(before.height/2, 1), RGBA))
	}
	smallest := blur.copy
	if halvings > 0 {
		smallest = blur.levels[halvings-1]
	}
	if blur.spare.width != smallest.width || blur.spare.height != smallest.height {
		blur.deleteTarget(blur.spare)
		blur.spare = newTarget(blur.functions, smallest.width, smallest.height, RGBA)
	}
}

func (blur *backdropBlur) release() {
	blur.deleteTarget(blur.copy)
	for _, level := range blur.levels {
		blur.deleteTarget(level)
	}
	blur.deleteTarget(blur.spare)
	blur.copy, blur.levels, blur.spare = target{}, nil, target{}
}

// newTarget makes a texture read smoothly between texels, and its framebuffer. The ones drawn into
// are RGBA, which every OpenGL ES 2 can draw into. It binds the texture on the first unit.
func newTarget(functions Functions, width, height int, format Enum) target {
	made := target{texture: functions.CreateTexture(), framebuffer: functions.CreateFramebuffer(), width: width, height: height}
	functions.ActiveTexture(Texture0)
	functions.BindTexture(Texture2D, made.texture)
	functions.TexImage2D(Texture2D, 0, format, width, height, format, UnsignedByte, nil)
	functions.TexParameteri(Texture2D, TextureMinFilter, int32(Linear))
	functions.TexParameteri(Texture2D, TextureMagFilter, int32(Linear))
	functions.TexParameteri(Texture2D, TextureWrapS, int32(ClampToEdge))
	functions.TexParameteri(Texture2D, TextureWrapT, int32(ClampToEdge))
	functions.BindFramebuffer(FramebufferTarget, made.framebuffer)
	functions.FramebufferTexture2D(FramebufferTarget, ColorAttachment0, Texture2D, made.texture, 0)
	functions.BindFramebuffer(FramebufferTarget, 0)
	return made
}

func (blur *backdropBlur) deleteTarget(made target) {
	if made.texture == 0 {
		return
	}
	blur.functions.DeleteFramebuffer(made.framebuffer)
	blur.functions.DeleteTexture(made.texture)
}
