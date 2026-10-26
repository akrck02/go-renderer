package opengl

import (
	"github.com/go-gl/gl/v4.1-core/gl"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/akrck02/go-renderer/scene"
)

// surfaceTextureUnit is the texture unit of the texture a draw lays over its surface.
const surfaceTextureUnit = 12

// textureStore keeps the scene's textures on the GPU, by picture: one upload for any number of
// meshes, and again only after the picture changed. It binds a texture only when the draw before
// had another.
type textureStore struct {
	byPicture   map[uint64]*uploadedTexture
	placeholder uint32 // a white pixel, bound when no texture is: drivers check every sampler
	bound       uint64 // the picture bound now, 0 = the placeholder
	uploads     int    // pictures uploaded so far, which tests count
}

// uploadedTexture is a picture on the GPU: which version of it, and how it is sampled.
type uploadedTexture struct {
	handle  uint32
	filled  bool // its pixels were uploaded once
	version uint64
	wrap    scene.Wrap
	sharp   bool
}

func newTextureStore() *textureStore {
	store := &textureStore{byPicture: map[uint64]*uploadedTexture{}}
	gl.GenTextures(1, &store.placeholder)
	white := [4]uint8{255, 255, 255, 255}
	gl.BindTexture(gl.TEXTURE_2D, store.placeholder)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, 1, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&white[0]))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	return store
}

// beginFrame binds the placeholder: other passes may have used the unit since the last frame.
func (store *textureStore) beginFrame(program *shaderProgram) {
	program.setInteger("surfaceTexture", surfaceTextureUnit)
	gl.ActiveTexture(gl.TEXTURE0 + surfaceTextureUnit)
	gl.BindTexture(gl.TEXTURE_2D, store.placeholder)
	gl.ActiveTexture(gl.TEXTURE0)
	store.bound = 0
}

// bind makes a material's texture the one draws read, uploading it first when new or changed.
func (store *textureStore) bind(material *scene.Material) {
	texture := material.Texture
	uploaded := store.byPicture[texture.Picture.ID()]
	current := uploaded != nil && uploaded.version == texture.Picture.Version() && uploaded.wrap == texture.Wrap && uploaded.sharp == texture.Sharp
	if current && store.bound == texture.Picture.ID() {
		return
	}
	gl.ActiveTexture(gl.TEXTURE0 + surfaceTextureUnit)
	defer gl.ActiveTexture(gl.TEXTURE0)
	if !current {
		uploaded = store.upload(texture, uploaded)
	}
	gl.BindTexture(gl.TEXTURE_2D, uploaded.handle)
	store.bound = texture.Picture.ID()
}

// upload sends a texture's pixels (when new or changed) and its sampling to the GPU, into the
// texture it had if any. It runs only when something changed, never in a frame that only draws.
func (store *textureStore) upload(texture *scene.Texture, uploaded *uploadedTexture) *uploadedTexture {
	picture := texture.Picture
	if uploaded == nil {
		uploaded = &uploadedTexture{}
		gl.GenTextures(1, &uploaded.handle)
		store.byPicture[picture.ID()] = uploaded
	}
	gl.BindTexture(gl.TEXTURE_2D, uploaded.handle)
	if !uploaded.filled || uploaded.version != picture.Version() {
		uploadPixels(picture)
		store.uploads++
	}
	setSampling(texture)
	uploaded.filled, uploaded.version, uploaded.wrap, uploaded.sharp = true, picture.Version(), texture.Wrap, texture.Sharp
	return uploaded
}

// uploadPixels fills the bound texture with a picture, and its smaller copies for smooth sampling
// far away. A picture's colours are sRGB, as a PNG's are; the GPU turns them into linear light as
// it reads them, so they multiply the linear colours of the scene and come out as they went in.
func uploadPixels(picture *draw2d.Image) {
	width, height := picture.Size()
	pixels := picture.Pixels
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(pixels.Stride/bytesPerPixel))
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.SRGB8_ALPHA8, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels.Pix))
	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
	gl.GenerateMipmap(gl.TEXTURE_2D)
}

// openglWraps are the wrapping modes of scene.Wrap, in its order.
var openglWraps = [...]int32{scene.Repeat: gl.REPEAT, scene.Clamp: gl.CLAMP_TO_EDGE, scene.Mirror: gl.MIRRORED_REPEAT}

// setSampling sets how the bound texture is read: wrapped as the texture says, and smooth (blended
// between pixels and between its smaller copies) or sharp (the nearest pixel).
func setSampling(texture *scene.Texture) {
	wrap := openglWraps[scene.Repeat]
	if int(texture.Wrap) < len(openglWraps) {
		wrap = openglWraps[texture.Wrap]
	}
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, wrap)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, wrap)
	minifying, magnifying := int32(gl.LINEAR_MIPMAP_LINEAR), int32(gl.LINEAR)
	if texture.Sharp {
		minifying, magnifying = gl.NEAREST_MIPMAP_NEAREST, gl.NEAREST
	}
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, minifying)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, magnifying)
}

// setTexture tells the scene program whether a draw is textured, and how; it binds the texture.
func (store *textureStore) setTexture(program *shaderProgram, material *scene.Material) {
	if !sceneplan.IsTextured(material) {
		program.setInteger("textured", 0)
		return
	}
	store.bind(material)
	program.setInteger("textured", 1)
	program.setVector4("texturePlacement", sceneplan.TexturePlacement(material))
}
