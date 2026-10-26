//go:build nintendo3ds

package nintendo3ds

/*
#cgo CFLAGS: -Wno-ignored-attributes
#include <3ds.h>
#include <citro3d.h>
#include <stdlib.h>

// newSurfaceTexture makes a texture of a size and format (GPU_RGBA8, GPU_RGB565 or GPU_ETC1), or
// returns NULL when linear memory is full.
static C3D_Tex* newSurfaceTexture(int width, int height, int format) {
	C3D_Tex* texture = malloc(sizeof(C3D_Tex));
	if (texture == NULL) return NULL;
	if (!C3D_TexInit(texture, width, height, (GPU_TEXCOLOR)format)) {
		free(texture);
		return NULL;
	}
	return texture;
}

// fillSurfaceTexture uploads already tiled pixels and sets how the texture is read.
static void fillSurfaceTexture(C3D_Tex* texture, const void* tiled, int wrap, int sharp) {
	C3D_TexUpload(texture, tiled);
	C3D_TexFlush(texture);
	GPU_TEXTURE_FILTER_PARAM filter = sharp ? GPU_NEAREST : GPU_LINEAR;
	C3D_TexSetFilter(texture, filter, filter);
	GPU_TEXTURE_WRAP_PARAM wraps[3] = {GPU_REPEAT, GPU_CLAMP_TO_EDGE, GPU_MIRRORED_REPEAT};
	C3D_TexSetWrap(texture, wraps[wrap], wraps[wrap]);
}

static void freeSurfaceTexture(C3D_Tex* texture) {
	C3D_TexDelete(texture);
	free(texture);
}

// setTextureStage makes the first colour stage pass the vertex colour on (0), or multiply it by
// the texture (1): the light is baked into the colour, so a texture costs one read a pixel.
static void setTextureStage(C3D_Tex* texture) {
	C3D_TexEnv* environment = C3D_GetTexEnv(0);
	C3D_TexEnvInit(environment);
	if (texture == NULL) {
		C3D_TexEnvSrc(environment, C3D_Both, GPU_PRIMARY_COLOR, 0, 0);
		C3D_TexEnvFunc(environment, C3D_Both, GPU_REPLACE);
		return;
	}
	C3D_TexBind(0, texture);
	C3D_TexEnvSrc(environment, C3D_Both, GPU_PRIMARY_COLOR, GPU_TEXTURE0, 0);
	C3D_TexEnvFunc(environment, C3D_Both, GPU_MODULATE);
}
*/
import "C"

import (
	"unsafe"

	"github.com/akrck02/go-renderer/scene"
)

// citro3dFormats are the formats of scene.TextureFormat, in its order.
var citro3dFormats = [...]C.int{scene.RGBA8: C.GPU_RGBA8, scene.RGB565: C.GPU_RGB565, scene.ETC1: C.GPU_ETC1}

// textureStore keeps the scene's textures in linear memory, by picture: one upload for any number
// of meshes, and again only after the picture changed. Pictures are scaled to the sides the
// PICA200 takes and encoded in Go, once, as they are uploaded.
type textureStore struct {
	byPicture map[uint64]*surfaceTexture
	bound     *C.C3D_Tex // what the first colour stage multiplies by; nil passes the colour on
	stageSet  bool       // the stage was set since the eye began (the overlay changes it)
	encoded   []byte     // reused by every upload: it only matters until it is copied
}

// surfaceTexture is a picture in linear memory: which version of it, and how it was kept.
type surfaceTexture struct {
	texture *C.C3D_Tex
	version uint64
	asked   scene.TextureFormat // the format the texture asked for (kept in RGBA8 if it would lose its transparency)
	wrap    scene.Wrap
	sharp   bool
}

func newTextureStore() *textureStore {
	return &textureStore{byPicture: map[uint64]*surfaceTexture{}}
}

// beginEye forgets the colour stage: prepareScene and the overlay set it between eyes.
func (store *textureStore) beginEye() { store.stageSet, store.bound = false, nil }

// useNone makes the next draws untextured.
func (store *textureStore) useNone() { store.setStage(nil) }

// use makes the next draws multiply their colour by a texture, uploading it first when new or
// changed. A texture linear memory has no room for draws untextured.
func (store *textureStore) use(texture *scene.Texture) {
	picture := texture.Picture
	kept := store.byPicture[picture.ID()]
	if kept == nil || kept.version != picture.Version() || kept.asked != texture.Format || kept.wrap != texture.Wrap || kept.sharp != texture.Sharp {
		kept = store.upload(texture, kept)
	}
	if kept == nil {
		store.setStage(nil)
		return
	}
	store.setStage(kept.texture)
}

func (store *textureStore) setStage(texture *C.C3D_Tex) {
	if store.stageSet && store.bound == texture {
		return
	}
	C.setTextureStage(texture)
	store.stageSet, store.bound = true, texture
}

// upload scales, encodes and copies a texture to linear memory, in place of the one it had.
func (store *textureStore) upload(texture *scene.Texture, old *surfaceTexture) *surfaceTexture {
	if old != nil {
		if store.bound == old.texture {
			store.stageSet = false // the stage must not keep reading memory let go of
		}
		C.freeSurfaceTexture(old.texture)
		delete(store.byPicture, texture.Picture.ID())
	}
	format := formatToUse(texture)
	pixels := fittedPixels(texture.Picture.Pixels)
	width, height := pixels.Rect.Dx(), pixels.Rect.Dy()
	made := C.newSurfaceTexture(C.int(width), C.int(height), citro3dFormats[format])
	if made == nil {
		return nil
	}
	store.encoded = encodeTexture(store.encoded, pixels, format)
	sharp := 0
	if texture.Sharp {
		sharp = 1
	}
	wrap := texture.Wrap
	if wrap < scene.Repeat || wrap > scene.Mirror {
		wrap = scene.Repeat
	}
	C.fillSurfaceTexture(made, unsafe.Pointer(&store.encoded[0]), C.int(wrap), C.int(sharp))
	kept := &surfaceTexture{texture: made, version: texture.Picture.Version(), asked: texture.Format, wrap: texture.Wrap, sharp: texture.Sharp}
	store.byPicture[texture.Picture.ID()] = kept
	return kept
}
