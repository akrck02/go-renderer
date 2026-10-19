//go:build nintendo3ds

package ctr

/*
#cgo CFLAGS: -Wno-ignored-attributes
#include <3ds.h>
#include <citro3d.h>
#include <stdlib.h>

// newLightTexture makes the picture of light: a texture in video memory the GPU draws into,
// read smoothly between texels when it is laid over the screen.
static C3D_Tex* newLightTexture(int width, int height) {
	C3D_Tex* texture = malloc(sizeof(C3D_Tex));
	if (texture == NULL) return NULL;
	if (!C3D_TexInitVRAM(texture, width, height, GPU_RGBA8)) {
		free(texture);
		return NULL;
	}
	C3D_TexSetFilter(texture, GPU_LINEAR, GPU_LINEAR);
	C3D_TexSetWrap(texture, GPU_CLAMP_TO_EDGE, GPU_CLAMP_TO_EDGE);
	return texture;
}

// newLightTarget lets the GPU draw into the picture of light, with no depth.
static C3D_RenderTarget* newLightTarget(C3D_Tex* texture) {
	C3D_DEPTHTYPE none;
	none.__i = -1;
	return C3D_RenderTargetCreateFromTex(texture, GPU_TEXFACE_2D, 0, none);
}

// newDiscTexture makes the discs' texture, one byte of brightness a pixel, from tiled pixels.
static C3D_Tex* newDiscTexture(int size, const void* tiled) {
	C3D_Tex* texture = malloc(sizeof(C3D_Tex));
	if (texture == NULL) return NULL;
	if (!C3D_TexInit(texture, size, size, GPU_L8)) {
		free(texture);
		return NULL;
	}
	C3D_TexSetFilter(texture, GPU_LINEAR, GPU_LINEAR);
	C3D_TexSetWrap(texture, GPU_CLAMP_TO_EDGE, GPU_CLAMP_TO_EDGE);
	C3D_TexUpload(texture, tiled);
	C3D_TexFlush(texture);
	return texture;
}

// beginLights clears the picture of light to the ambient, makes the next draws go to it, and
// adds what they draw to what is there.
static void beginLights(C3D_RenderTarget* target, u32 ambient, C3D_Tex* discs, C3D_Tex* pictures) {
	C3D_RenderTargetClear(target, C3D_CLEAR_COLOR, ambient, 0);
	C3D_FrameDrawOn(target);
	C3D_TexBind(0, discs);
	if (pictures != NULL) C3D_TexBind(1, pictures);
	C3D_CullFace(GPU_CULL_NONE);
	C3D_DepthTest(false, GPU_ALWAYS, GPU_WRITE_COLOR);
	C3D_AlphaBlend(GPU_BLEND_ADD, GPU_BLEND_ADD, GPU_ONE, GPU_ONE, GPU_ONE, GPU_ONE);
}

// describeLights tells where the lights' corners are, laid out as the overlay's (flat.go).
static void describeLights(void* vertices) {
	C3D_AttrInfo* attributeInfo = C3D_GetAttrInfo();
	AttrInfo_Init(attributeInfo);
	AttrInfo_AddLoader(attributeInfo, 0, GPU_FLOAT, 2);
	AttrInfo_AddLoader(attributeInfo, 1, GPU_FLOAT, 2);
	AttrInfo_AddLoader(attributeInfo, 2, GPU_FLOAT, 4);
	AttrInfo_AddLoader(attributeInfo, 3, GPU_FLOAT, 2);
	C3D_BufInfo* bufferInfo = C3D_GetBufInfo();
	BufInfo_Init(bufferInfo);
	BufInfo_Add(bufferInfo, vertices, 10 * sizeof(float), 4, 0x3210);
}

// setDiscStages: the light's colour times its disc's brightness.
static void setDiscStages(void) {
	C3D_TexEnv* first = C3D_GetTexEnv(0);
	C3D_TexEnvInit(first);
	C3D_TexEnvSrc(first, C3D_RGB, GPU_PRIMARY_COLOR, GPU_TEXTURE0, 0);
	C3D_TexEnvFunc(first, C3D_RGB, GPU_MODULATE);
	C3D_TexEnvSrc(first, C3D_Alpha, GPU_PRIMARY_COLOR, 0, 0);
	C3D_TexEnvFunc(first, C3D_Alpha, GPU_REPLACE);
	C3D_TexEnvInit(C3D_GetTexEnv(1)); // passes the first stage on
}

// setSpriteStages: the light's colour times its picture's colours, times the picture's alpha, so
// that where the picture is clear it adds no light.
static void setSpriteStages(void) {
	C3D_TexEnv* first = C3D_GetTexEnv(0);
	C3D_TexEnvInit(first);
	C3D_TexEnvSrc(first, C3D_RGB, GPU_PRIMARY_COLOR, GPU_TEXTURE1, 0);
	C3D_TexEnvFunc(first, C3D_RGB, GPU_MODULATE);
	C3D_TexEnvSrc(first, C3D_Alpha, GPU_PRIMARY_COLOR, 0, 0);
	C3D_TexEnvFunc(first, C3D_Alpha, GPU_REPLACE);
	C3D_TexEnv* second = C3D_GetTexEnv(1);
	C3D_TexEnvInit(second);
	C3D_TexEnvSrc(second, C3D_RGB, GPU_PREVIOUS, GPU_TEXTURE1, 0);
	C3D_TexEnvOpRgb(second, GPU_TEVOP_RGB_SRC_COLOR, GPU_TEVOP_RGB_SRC_ALPHA, 0);
	C3D_TexEnvFunc(second, C3D_RGB, GPU_MODULATE);
	C3D_TexEnvSrc(second, C3D_Alpha, GPU_PREVIOUS, 0, 0);
	C3D_TexEnvFunc(second, C3D_Alpha, GPU_REPLACE);
}

static void drawLights(int count, int sprites, u16* indices) {
	if (sprites) setSpriteStages(); else setDiscStages();
	C3D_DrawElements(GPU_TRIANGLES, count * 6, C3D_UNSIGNED_SHORT, indices);
}

// multiplyByLight lays the picture of light over the screen with a blend that multiplies by twice
// the light (screen × light + light × screen), keeping the screen's alpha: half is neutral.
static void multiplyByLight(C3D_Tex* light, u16* indices) {
	C3D_TexBind(0, light);
	C3D_TexEnv* first = C3D_GetTexEnv(0);
	C3D_TexEnvInit(first);
	C3D_TexEnvSrc(first, C3D_RGB, GPU_TEXTURE0, 0, 0);
	C3D_TexEnvFunc(first, C3D_RGB, GPU_REPLACE);
	C3D_TexEnvSrc(first, C3D_Alpha, GPU_PRIMARY_COLOR, 0, 0);
	C3D_TexEnvFunc(first, C3D_Alpha, GPU_REPLACE);
	C3D_TexEnvInit(C3D_GetTexEnv(1));
	C3D_AlphaBlend(GPU_BLEND_ADD, GPU_BLEND_ADD, GPU_DST_COLOR, GPU_SRC_COLOR, GPU_ZERO, GPU_ONE);
	C3D_DrawElements(GPU_TRIANGLES, 6, C3D_UNSIGNED_SHORT, indices);
}
*/
import "C"

import (
	"unsafe"

	"github.com/akrck02/go-renderer/opengl/overlay"
)

// lightPass draws the 2D lights (see overlay.Lighting) on the PICA200: the lights are added into
// a picture of light, a texture the GPU draws into, which then multiplies the screen. Its
// textures are made the first time a list has lights; failed says they could not be, which is
// told once.
type lightPass struct {
	texture *C.C3D_Tex
	target  *C.C3D_RenderTarget
	discs   *C.C3D_Tex
	failed  bool
}

// prepare makes the pass's textures, the first time; false when they cannot be made.
func (pass *lightPass) prepare() bool {
	if pass.target != nil {
		return true
	}
	if pass.failed {
		return false
	}
	pass.texture = C.newLightTexture(lightTextureWidth, lightTextureHeight)
	if pass.texture != nil {
		pass.target = C.newLightTarget(pass.texture)
	}
	tiled := tileForPICA(nil, lightDiscs(), discTextureSide)
	pass.discs = C.newDiscTexture(discTextureSide, unsafe.Pointer(&tiled[0]))
	if pass.target == nil || pass.discs == nil {
		println("ctr: no room for the 2D lights' textures: lists are drawn unlit")
		pass.failed, pass.target = true, nil
		return false
	}
	return true
}

// light lights what the surface shows with one lighting of a painted list, and leaves the next
// draws going to the surface again.
func (renderer *FlatRenderer) light(surface Surface, painted *paintedList, lighting overlay.Lighting, width, height int) {
	pass := &renderer.lights
	C.beginLights(pass.target, C.u32(ambientColour(lighting.Ambient)), pass.discs, renderer.pictures)
	// y grows upwards here: a texture drawn into keeps its rows bottom first, as the tiling does
	SetMatrix(renderer.projection, Orthographic(0, lightCanvasWidth, 0, lightCanvasHeight, 0, 1))
	C.describeLights(painted.lightVertices.data)
	for _, run := range painted.lightRuns {
		first, last := max(run.first, lighting.First), min(run.first+run.count, lighting.Last)
		if first >= last || run.pictured && renderer.pictures == nil {
			continue
		}
		C.drawLights(C.int(last-first), C.int(boolToInt(run.pictured)), (*C.u16)(indicesFrom(first)))
	}
	surface.resume()
	SetMatrix(renderer.projection, OrthographicTilt(0, float32(width), float32(height), 0, 0, 1))
	C.multiplyByLight(pass.texture, (*C.u16)(indicesFrom(painted.lightCount)))
}

// writeLights writes a painted list's lights and lightings, after its quads; false when there is
// no memory for them, and the list is drawn unlit.
func (renderer *FlatRenderer) writeLights(painted *paintedList, width, height int) bool {
	painted.lightings, painted.lightRuns, painted.lightVertices.data = painted.lightings[:0], painted.lightRuns[:0], nil
	lightings := renderer.painter.Lightings()
	if len(lightings) == 0 || !renderer.lights.prepare() {
		return true
	}
	lights := renderer.painter.Lights()
	lights = lights[:min(len(lights), maxQuadsPerList-1)] // room for the screen's rectangle after them
	floats := (len(lights) + 1) * 4 * floatsPerQuadVertex
	if !growLinear(&painted.lightMemory, floats*4) {
		return false
	}
	vertices := unsafe.Slice((*float32)(painted.lightMemory.data), floats)
	painted.lightRuns = writeLightVertices(vertices, painted.lightRuns, lights, renderer.picturesSize, width, height)
	flushLinear(painted.lightMemory.data, floats*4)
	painted.lightVertices, painted.lightCount = painted.lightMemory, len(lights)
	for _, lighting := range lightings {
		lighting.First, lighting.Last = min(lighting.First, len(lights)), min(lighting.Last, len(lights))
		painted.lightings = append(painted.lightings, lighting)
	}
	return true
}
