//go:build nintendo3ds

package ctr

/*
#cgo CFLAGS: -Wno-ignored-attributes
#include <3ds.h>
#include <citro3d.h>
#include <stdlib.h>

// The same transfer to the screen as the top screen's (ctr.go).
#define BOTTOM_TRANSFER_FLAGS \
	(GX_TRANSFER_FLIP_VERT(0) | GX_TRANSFER_OUT_TILED(0) | GX_TRANSFER_RAW_COPY(0) | \
	GX_TRANSFER_IN_FORMAT(GX_TRANSFER_FMT_RGBA8) | GX_TRANSFER_OUT_FORMAT(GX_TRANSFER_FMT_RGB8) | \
	GX_TRANSFER_SCALING(GX_TRANSFER_SCALE_NO))

// openBottomScreen makes a picture for the bottom screen, a 240 × 320 portrait panel on its side
// like the top one, with depth, so it can show a scene as well as an interface.
static C3D_RenderTarget* openBottomScreen(void) {
	C3D_DEPTHTYPE depth;
	depth.__i = GPU_RB_DEPTH24_STENCIL8;
	C3D_RenderTarget* target = C3D_RenderTargetCreate(240, 320, GPU_RB_RGBA8, depth);
	if (target != NULL) C3D_RenderTargetSetOutput(target, GFX_BOTTOM, GFX_LEFT, BOTTOM_TRANSFER_FLAGS);
	return target;
}

static void clearBottom(C3D_RenderTarget* target, u32 clearColour) {
	C3D_RenderTargetClear(target, C3D_CLEAR_ALL, clearColour, 0);
	C3D_FrameDrawOn(target);
}

// newCoverageTexture makes a square texture of one byte per pixel (alpha), filtered linearly.
static C3D_Tex* newCoverageTexture(int size) {
	C3D_Tex* texture = malloc(sizeof(C3D_Tex));
	if (texture == NULL) return NULL;
	if (!C3D_TexInit(texture, size, size, GPU_A8)) {
		free(texture);
		return NULL;
	}
	C3D_TexSetFilter(texture, GPU_LINEAR, GPU_LINEAR);
	C3D_TexSetWrap(texture, GPU_CLAMP_TO_EDGE, GPU_CLAMP_TO_EDGE);
	return texture;
}

static void freeTexture(C3D_Tex* texture) {
	C3D_TexDelete(texture);
	free(texture);
}

// uploadTexture copies tiled pixels into the texture and flushes the processor's cache, so the
// GPU reads the new pixels and not what linear memory held before.
static void uploadTexture(C3D_Tex* texture, const void* tiled) {
	C3D_TexUpload(texture, tiled);
	C3D_TexFlush(texture);
}

// newPicturesTexture makes a square texture of straight RGBA pixels, filtered linearly.
static C3D_Tex* newPicturesTexture(int size) {
	C3D_Tex* texture = malloc(sizeof(C3D_Tex));
	if (texture == NULL) return NULL;
	if (!C3D_TexInit(texture, size, size, GPU_RGBA8)) {
		free(texture);
		return NULL;
	}
	C3D_TexSetFilter(texture, GPU_LINEAR, GPU_LINEAR);
	C3D_TexSetWrap(texture, GPU_CLAMP_TO_EDGE, GPU_CLAMP_TO_EDGE);
	return texture;
}

// prepareOverlay sets what the overlay's rectangles share: the atlas on texture unit 0, the
// pictures (when there are any) on unit 1, blended over what is behind, with no depth test.
static void prepareOverlay(C3D_Tex* atlas, C3D_Tex* pictures) {
	C3D_TexBind(0, atlas);
	if (pictures != NULL) C3D_TexBind(1, pictures);
	C3D_CullFace(GPU_CULL_NONE);
	C3D_DepthTest(false, GPU_ALWAYS, GPU_WRITE_COLOR);
	C3D_AlphaBlend(GPU_BLEND_ADD, GPU_BLEND_ADD, GPU_SRC_ALPHA, GPU_ONE_MINUS_SRC_ALPHA, GPU_ONE, GPU_ONE_MINUS_SRC_ALPHA);
}

// setPlainStages: the colour as it comes, its alpha times the atlas's coverage.
static void setPlainStages(void) {
	C3D_TexEnv* first = C3D_GetTexEnv(0);
	C3D_TexEnvInit(first);
	C3D_TexEnvSrc(first, C3D_RGB, GPU_PRIMARY_COLOR, 0, 0);
	C3D_TexEnvFunc(first, C3D_RGB, GPU_REPLACE);
	C3D_TexEnvSrc(first, C3D_Alpha, GPU_PRIMARY_COLOR, GPU_TEXTURE0, 0);
	C3D_TexEnvFunc(first, C3D_Alpha, GPU_MODULATE);
	C3D_TexEnvInit(C3D_GetTexEnv(1)); // passes the first stage on
}

// setPictureStages: the picture's colours times the colour (a tint), then the alpha times the
// atlas's coverage too, which rounds the picture's corners.
static void setPictureStages(void) {
	C3D_TexEnv* first = C3D_GetTexEnv(0);
	C3D_TexEnvInit(first);
	C3D_TexEnvSrc(first, C3D_Both, GPU_PRIMARY_COLOR, GPU_TEXTURE1, 0);
	C3D_TexEnvFunc(first, C3D_Both, GPU_MODULATE);
	C3D_TexEnv* second = C3D_GetTexEnv(1);
	C3D_TexEnvInit(second);
	C3D_TexEnvSrc(second, C3D_RGB, GPU_PREVIOUS, 0, 0);
	C3D_TexEnvFunc(second, C3D_RGB, GPU_REPLACE);
	C3D_TexEnvSrc(second, C3D_Alpha, GPU_PREVIOUS, GPU_TEXTURE0, 0);
	C3D_TexEnvFunc(second, C3D_Alpha, GPU_MODULATE);
}

// describeQuads tells where overlay corners are: position (2 floats), atlas position (2), colour
// (4) and picture position (2).
static void describeQuads(void* vertices) {
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

static void drawRun(int first, int count, int pictured) {
	if (pictured) setPictureStages(); else setPlainStages();
	C3D_DrawArrays(GPU_TRIANGLES, first, count);
}

static void* newLinearMemory(u32 size) {
	return linearAlloc(size);
}

static void flush(void* data, u32 size) {
	GSPGPU_FlushDataCache(data, size);
}
*/
import "C"

import (
	_ "embed"
	"errors"
	"unsafe"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/opengl/overlay"
	"github.com/akrck02/go-renderer/scene"
)

// The bottom screen's size, which interfaces are laid out for. It is also the touch screen.
const (
	BottomWidth  = 320
	BottomHeight = 240
)

//go:embed shaders/quad.shbin
var quadShaderBinary []byte

// BottomScreen is the bottom screen, drawn by citro3d (instead of the text console of
// OpenBottomConsole).
type BottomScreen struct {
	target *C.C3D_RenderTarget
}

// OpenBottomScreen prepares the bottom screen for drawing, after Open.
func OpenBottomScreen() (*BottomScreen, error) {
	target := C.openBottomScreen()
	if target == nil {
		return nil, errors.New("ctr: cannot draw on the bottom screen")
	}
	return &BottomScreen{target: target}, nil
}

// Surface is a screen the GPU draws a frame on: the top one or the bottom one.
type Surface interface {
	clear(colour uint32) // clears it for one picture and makes the next draws go to it
	size() (width, height int)
}

func (screen *BottomScreen) clear(colour uint32) { C.clearBottom(screen.target, C.u32(colour)) }
func (screen *BottomScreen) size() (int, int)    { return BottomWidth, BottomHeight }
func (screen *Screen) clear(colour uint32)       { screen.Clear(colour) }
func (screen *Screen) size() (int, int)          { return ScreenWidth, ScreenHeight }

// FlatRenderer paints a 2D draw list (an interface: go-ui's, or a program's own) on the bottom
// screen, as gles.FlatRenderer does: opengl/overlay lays it out as rectangles of one atlas of
// glyphs, icons and rounded corners, and they are drawn in one call. Pictures are not drawn yet.
type FlatRenderer struct {
	screen     *BottomScreen
	shader     *Shader
	projection int
	painter    *overlay.Painter
	layer      scene.Overlay

	texture      *C.C3D_Tex
	textureSize  int
	atlasVersion int
	tiled        []byte
	tooLarge     bool // the atlas outgrew the largest texture: told once

	pictures         *C.C3D_Tex
	picturesSize     int
	tiledPictures    []byte
	picturesTooLarge bool // told once

	vertices []float32
	runs     []quadRun
	buffers  [2]linearBuffer // one for the frame being drawn, one for the GPU's previous frame
	frame    int
}

// linearBuffer is linear memory the GPU reads vertices from, kept and grown from frame to frame.
type linearBuffer struct {
	data     unsafe.Pointer
	capacity int // in bytes
}

// NewFlatRenderer loads the overlay shader and the fonts, for painting on the bottom screen.
func NewFlatRenderer(screen *BottomScreen) (*FlatRenderer, error) {
	shader, err := LoadShader(quadShaderBinary)
	if err != nil {
		return nil, err
	}
	painter, err := overlay.NewPainter()
	if err != nil {
		shader.Free()
		return nil, err
	}
	return &FlatRenderer{screen: screen, shader: shader, projection: shader.Uniform("projection"),
		painter: painter, atlasVersion: -1}, nil
}

// Paint clears the bottom screen to the background and paints the list on it, in a frame begun
// with BeginFrame. Coordinates are the screen's pixels, from its top-left corner.
func (renderer *FlatRenderer) Paint(list *draw2d.List, background draw2d.Color) {
	renderer.PaintOn(renderer.screen, list, background)
}

// PaintOn is Paint on either screen: an interface can also be shown on the top one (in one picture,
// not in 3D).
func (renderer *FlatRenderer) PaintOn(surface Surface, list *draw2d.List, background draw2d.Color) {
	surface.clear(packedColour(background))
	renderer.paint(surface, list)
}

// PaintOver paints the list over what the screen shows already (a scene), without clearing it, on
// the picture drawn now (for 3D, call it after DrawOn for each eye).
func (renderer *FlatRenderer) PaintOver(surface Surface, list *draw2d.List) {
	renderer.paint(surface, list)
}

func (renderer *FlatRenderer) paint(surface Surface, list *draw2d.List) {
	width, height := surface.size()
	renderer.layer.Drawing = list
	if renderer.layer.IsEmpty() {
		return
	}
	quads := renderer.painter.Paint(&renderer.layer, nil, overlay.Frame{Width: width, Height: height, Scale: 1})
	if !renderer.uploadAtlas() {
		return
	}
	renderer.uploadPictures()
	renderer.vertices, renderer.runs = appendQuadVertices(renderer.vertices[:0], renderer.runs[:0], quads, renderer.textureSize, renderer.picturesSize)
	if len(renderer.vertices) == 0 {
		return
	}
	vertices := renderer.copyVertices()
	if vertices == nil {
		return
	}
	renderer.shader.Use()
	// Pixels from the top-left corner: y grows downwards, so top and bottom are swapped.
	SetMatrix(renderer.projection, OrthographicTilt(0, float32(width), float32(height), 0, 0, 1))
	C.prepareOverlay(renderer.texture, renderer.pictures)
	C.describeQuads(vertices)
	for _, run := range renderer.runs {
		if run.pictured && renderer.pictures == nil {
			continue // pictures that could not be uploaded
		}
		pictured := 0
		if run.pictured {
			pictured = 1
		}
		C.drawRun(C.int(run.first), C.int(run.count), C.int(pictured))
	}
}

// uploadPictures sends the pictures' colours to the GPU when pictures were added or changed,
// tiled. Pictures larger than the chip's largest texture are not drawn.
func (renderer *FlatRenderer) uploadPictures() {
	pictures := renderer.painter.Pictures()
	size := pictures.Size()
	if size == 0 {
		return // no picture drawn yet
	}
	whole, areas := pictures.Pending()
	if !whole && len(areas) == 0 && size == renderer.picturesSize {
		return
	}
	if size > largestTexture {
		if !renderer.picturesTooLarge {
			println("ctr: the overlay's pictures outgrew the largest texture:", size)
			renderer.picturesTooLarge = true
		}
		pictures.Uploaded()
		return
	}
	if renderer.pictures == nil || renderer.picturesSize != size {
		if renderer.pictures != nil {
			C.freeTexture(renderer.pictures)
		}
		renderer.pictures = C.newPicturesTexture(C.int(size))
		renderer.picturesSize = size
		if renderer.pictures == nil {
			return
		}
	}
	// The whole atlas, even for a few changed areas: tiling scatters an area's rows over the
	// texture. Pictures change seldom; a video would want only its tiles.
	renderer.tiledPictures = tileRGBAForPICA(renderer.tiledPictures, pictures.Image.Pix, size)
	C.uploadTexture(renderer.pictures, unsafe.Pointer(&renderer.tiledPictures[0]))
	pictures.Uploaded()
}

// uploadAtlas sends the atlas to the GPU when glyphs, icons or corners were added to it, tiled the
// way the PICA200 reads textures. It answers whether there is an atlas to draw with.
func (renderer *FlatRenderer) uploadAtlas() bool {
	atlas := renderer.painter.Atlas()
	if atlas.Version == renderer.atlasVersion {
		return renderer.texture != nil
	}
	size := atlas.Size()
	if size > largestTexture {
		if !renderer.tooLarge {
			println("ctr: the overlay's atlas outgrew the largest texture:", size)
			renderer.tooLarge = true
		}
		return renderer.texture != nil // draw with what the texture still has
	}
	if renderer.texture == nil || renderer.textureSize != size {
		if renderer.texture != nil {
			C.freeTexture(renderer.texture)
		}
		renderer.texture = C.newCoverageTexture(C.int(size))
		renderer.textureSize = size
		if renderer.texture == nil {
			return false
		}
	}
	renderer.tiled = tileForPICA(renderer.tiled, atlas.Image.Pix, size)
	C.uploadTexture(renderer.texture, unsafe.Pointer(&renderer.tiled[0]))
	renderer.atlasVersion = atlas.Version
	return true
}

// copyVertices copies this frame's corners to linear memory, alternating between two buffers so
// the GPU never reads one being written.
func (renderer *FlatRenderer) copyVertices() unsafe.Pointer {
	buffer := &renderer.buffers[renderer.frame%2]
	renderer.frame++
	bytes := len(renderer.vertices) * 4
	if buffer.capacity < bytes {
		freeLinearMemory(buffer.data)
		buffer.capacity = bytes * 2 // room to grow before the next allocation
		buffer.data = C.newLinearMemory(C.u32(buffer.capacity))
		if buffer.data == nil {
			buffer.capacity = 0
			return nil
		}
	}
	copy(unsafe.Slice((*float32)(buffer.data), len(renderer.vertices)), renderer.vertices)
	C.flush(buffer.data, C.u32(bytes))
	return buffer.data
}

// packedColour packs a draw2d colour (in the screen's gamma already) as 0xRRGGBBAA.
func packedColour(colour draw2d.Color) uint32 {
	channel := func(value float32) uint32 { return uint32(min(max(value, 0), 1)*255 + 0.5) }
	return channel(colour[0])<<24 | channel(colour[1])<<16 | channel(colour[2])<<8 | channel(colour[3])
}
