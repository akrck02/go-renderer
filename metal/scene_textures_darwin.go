package metal

/*
#include "scene_darwin.h"
*/
import "C"

import (
	"unsafe"

	"github.com/akrck02/go-renderer/internal/sceneplan"
	"github.com/akrck02/go-renderer/scene"
)

// surfaceTextureIndex is where the scene's fragment function reads a draw's texture.
const surfaceTextureIndex = 2

// uvBufferIndex is where the scene's vertex function reads a mesh's UVs.
const uvBufferIndex = 4

// textureStore keeps the scene's textures on the GPU, by picture: one upload for any number of
// meshes, and again only after the picture changed. How a texture is read (wrap, sharp) is the
// shader's choice of sampler, so it needs no upload.
type textureStore struct {
	byPicture   map[uint64]*uploadedTexture
	placeholder unsafe.Pointer // a white pixel, bound when a draw has no texture: Metal wants one bound
}

// uploadedTexture is a picture on the GPU and the version of it that was uploaded.
type uploadedTexture struct {
	handle  unsafe.Pointer
	version uint64
}

func (renderer *SceneRenderer) prepareSurfaceTextures() {
	white := []byte{255, 255, 255, 255}
	renderer.textures = &textureStore{byPicture: map[uint64]*uploadedTexture{}}
	renderer.textures.placeholder = renderer.newTexture(C.sceneFormatRGBA8sRGB, 1, 1, false, false)
	C.sceneTextureWrite(renderer.textures.placeholder, 0, 1, 1, unsafe.Pointer(&white[0]), 4)
	renderer.placeholders.uvs = renderer.newBuffer([]float32{0, 0})
}

// surfaceTexture returns the texture a draw reads: its material's, uploaded when new or changed,
// or the placeholder.
func (renderer *SceneRenderer) surfaceTexture(material *scene.Material) unsafe.Pointer {
	if !sceneplan.IsTextured(material) {
		return renderer.textures.placeholder
	}
	picture := material.Texture.Picture
	uploaded := renderer.textures.byPicture[picture.ID()]
	if uploaded != nil && uploaded.version == picture.Version() {
		return uploaded.handle
	}
	return renderer.uploadTexture(material.Texture, uploaded)
}

// uploadTexture sends a texture's pixels to the GPU, in sRGB (read as linear light, as on the
// desktop), with its smaller copies for smooth reading far away. A changed picture of another size
// gets a new texture. It runs only when a picture is new or changed.
func (renderer *SceneRenderer) uploadTexture(texture *scene.Texture, uploaded *uploadedTexture) unsafe.Pointer {
	picture := texture.Picture
	width, height := picture.Size()
	if uploaded != nil {
		C.sceneRelease(uploaded.handle)
	}
	uploaded = &uploadedTexture{handle: renderer.newTexture(C.sceneFormatRGBA8sRGB, width, height, false, true), version: picture.Version()}
	pixels := picture.Pixels
	C.sceneTextureWrite(uploaded.handle, 0, C.int(width), C.int(height), unsafe.Pointer(&pixels.Pix[0]), C.long(pixels.Stride))
	C.sceneTextureGenerateMipmaps(renderer.device, uploaded.handle)
	renderer.textures.byPicture[picture.ID()] = uploaded
	return uploaded.handle
}

// meshUVs returns the buffer of a mesh's UVs for a draw with a material: made the first time the
// mesh is drawn textured (or uploaded with UVs), and the placeholder for an untextured draw.
func (renderer *SceneRenderer) meshUVs(mesh *scene.Mesh, buffers *metalMesh, material *scene.Material) unsafe.Pointer {
	if !sceneplan.IsTextured(material) {
		return renderer.placeholders.uvs
	}
	if buffers.uvs == nil {
		buffers.uvs = renderer.newBuffer(sceneplan.UVsToDraw(mesh, material))
	}
	return buffers.uvs
}
