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

static void drawOn(C3D_RenderTarget* target) {
	C3D_FrameDrawOn(target);
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

// setPicturesFilter reads the pictures at the nearest pixel (pixel art) or blended between pixels;
// the texture's binding carries it to the GPU.
static void setPicturesFilter(C3D_Tex* texture, int nearest) {
	GPU_TEXTURE_FILTER_PARAM filter = nearest ? GPU_NEAREST : GPU_LINEAR;
	C3D_TexSetFilter(texture, filter, filter);
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

// quadIndices make each rectangle's four corners two triangles: 0 1 2 and 0 2 3, for as many
// rectangles as 16-bit corner numbers reach. Made once, in linear memory.
static u16* quadIndices;

static int makeQuadIndices(int quads) {
	if (quadIndices != NULL) return 1;
	quadIndices = linearAlloc(quads * 6 * sizeof(u16));
	if (quadIndices == NULL) return 0;
	for (int quad = 0; quad < quads; quad++) {
		u16 corner = quad * 4;
		u16* indices = quadIndices + quad * 6;
		indices[0] = corner; indices[1] = corner + 1; indices[2] = corner + 2;
		indices[3] = corner; indices[4] = corner + 2; indices[5] = corner + 3;
	}
	GSPGPU_FlushDataCache(quadIndices, quads * 6 * sizeof(u16));
	return 1;
}

// indicesFrom is where the indices of rectangle first start, for a pass that draws from another
// buffer with the same indices (lights.go).
static u16* indicesFrom(int first) {
	return quadIndices + first * 6;
}

static void drawRun(int first, int count, int pictured) {
	if (pictured) setPictureStages(); else setPlainStages();
	C3D_DrawElements(GPU_TRIANGLES, count * 6, C3D_UNSIGNED_SHORT, quadIndices + first * 6);
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
	"math"
	"unsafe"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/failure"
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
		return nil, failure.Wrap(ScreenNotOpened, "the bottom screen", nil)
	}
	return &BottomScreen{target: target}, nil
}

// Surface is a screen the GPU draws a frame on: the top one or the bottom one.
type Surface interface {
	clear(colour uint32) // clears it for one picture and makes the next draws go to it
	resume()             // makes the next draws go to it again, after a pass drew elsewhere
	size() (width, height int)
}

func (screen *BottomScreen) clear(colour uint32) { C.clearBottom(screen.target, C.u32(colour)) }
func (screen *BottomScreen) resume()             { C.drawOn(screen.target) }
func (screen *BottomScreen) size() (int, int)    { return BottomWidth, BottomHeight }
func (screen *Screen) clear(colour uint32)       { screen.Clear(colour) }
func (screen *Screen) size() (int, int)          { return ScreenWidth, ScreenHeight }

// FlatRenderer paints a 2D draw list (an interface: go-ui's, or a program's own) on the bottom
// screen, as gles.FlatRenderer does: opengl/overlay lays it out as rectangles of one atlas of
// glyphs, icons and rounded corners, drawn in runs, and its 2D lights light them (lights.go).
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

	lights lightPass // made the first time a list has lights

	// lists are the lists painted in a frame, in the order they are painted (a frame may paint
	// several: the scene's drawing in each eye, then the interface), each with its corners kept:
	// a list that draws the same as the frame before is drawn again from them, without laying it
	// out, which is most of what an interface costs the console's processor
	lists      []paintedList
	listsFrame uint32 // the frame nextList counts in
	nextList   int
}

// paintedList is a list's corners in linear memory, and what they were made for.
type paintedList struct {
	fingerprint               uint64
	width, height             int
	textureSize, picturesSize int // texture positions are fractions of them
	memory                    linearBuffer
	vertices                  linearBuffer // memory, once it holds the corners; empty when it does not
	runs                      []quadRun
	quadCount                 int
	sharp                     bool // a picture of the list is read at the nearest pixel
	// the list's lights (lights.go): their corners, then a rectangle for each lighting's area, in
	// their own memory; how they run; and where the list is lit
	lightMemory   linearBuffer
	lightVertices linearBuffer // lightMemory, once it holds the corners
	lightRuns     []quadRun
	lightCount    int
	lightings     []overlay.Lighting
}

// anySharp tells whether a quad reads its picture at the nearest pixel.
func anySharp(quads []overlay.Quad) bool {
	for _, quad := range quads {
		if quad.Pictured && quad.Sharp {
			return true
		}
	}
	return false
}

// boolToInt is 1 for true and 0 for false, as C wants a flag.
func boolToInt(flag bool) int {
	if flag {
		return 1
	}
	return 0
}

// linearBuffer is linear memory the GPU reads vertices from, kept and grown from frame to frame.
type linearBuffer struct {
	data     unsafe.Pointer
	capacity int // in bytes
}

// firstVertexBuffer is the room a list's corners start with, a few thousand rectangles.
const firstVertexBuffer = 64 << 10

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
	if renderer.listsFrame != frameNumber {
		renderer.listsFrame, renderer.nextList = frameNumber, 0
	}
	if renderer.nextList == len(renderer.lists) {
		renderer.lists = append(renderer.lists, paintedList{})
	}
	painted := &renderer.lists[renderer.nextList]
	renderer.nextList++
	fingerprint := quickFingerprint(list)
	same := painted.vertices.data != nil && painted.fingerprint == fingerprint && painted.width == width && painted.height == height &&
		painted.textureSize == renderer.textureSize && painted.picturesSize == renderer.picturesSize
	if !same && !renderer.paintList(painted, width, height) {
		return
	}
	painted.fingerprint = fingerprint
	renderer.shader.Use()
	if renderer.pictures != nil {
		// one filter for the whole texture: a list with a sharp picture reads all of them sharp
		C.setPicturesFilter(renderer.pictures, C.int(boolToInt(painted.sharp)))
	}
	// without lights this is one pass over the runs; each lighting cuts it in two
	drawn := 0
	for index, lighting := range painted.lightings {
		renderer.drawQuads(painted, drawn, lighting.At, width, height)
		renderer.light(surface, painted, index, width, height)
		drawn = lighting.At
	}
	renderer.drawQuads(painted, drawn, painted.quadCount, width, height)
}

// drawQuads draws a painted list's rectangles from first up to (not including) last, run by run.
func (renderer *FlatRenderer) drawQuads(painted *paintedList, first, last, width, height int) {
	if first >= last {
		return
	}
	// Pixels from the top-left corner: y grows downwards, so top and bottom are swapped.
	SetMatrix(renderer.projection, OrthographicTilt(0, float32(width), float32(height), 0, 0, 1))
	C.prepareOverlay(renderer.texture, renderer.pictures)
	C.describeQuads(painted.vertices.data)
	for _, run := range painted.runs {
		from, to := max(run.first, first), min(run.first+run.count, last)
		if from >= to || run.pictured && renderer.pictures == nil {
			continue // outside the range, or pictures that could not be uploaded
		}
		C.drawRun(C.int(from), C.int(to-from), C.int(boolToInt(run.pictured)))
	}
}

// indicesFrom is where the indices of rectangle first start, for a pass that draws from another
// buffer with the same indices.
func indicesFrom(first int) unsafe.Pointer { return unsafe.Pointer(C.indicesFrom(C.int(first))) }

// flushLinear makes what the processor wrote into linear memory visible to the GPU.
func flushLinear(data unsafe.Pointer, size int) { C.flush(data, C.u32(size)) }

// growLinear makes sure a buffer of linear memory holds size bytes, with room to grow; false when
// there is no memory.
func growLinear(buffer *linearBuffer, size int) bool {
	if buffer.capacity >= size {
		return true
	}
	freeLinearMemory(buffer.data)
	buffer.capacity = max(2*size, firstVertexBuffer) // room to grow before the next allocation
	buffer.data = C.newLinearMemory(C.u32(buffer.capacity))
	if buffer.data == nil {
		buffer.capacity = 0
		return false
	}
	return true
}

// paintList lays the list out as rectangles and writes their corners into the painted list's
// own linear memory, which the GPU has finished reading: a frame begins once it is done with the
// one before.
func (renderer *FlatRenderer) paintList(painted *paintedList, width, height int) bool {
	painted.vertices.data, painted.runs, painted.quadCount = nil, painted.runs[:0], 0
	quads := renderer.painter.Paint(&renderer.layer, nil, overlay.Frame{Width: width, Height: height, Scale: 1})
	if !renderer.uploadAtlas() {
		return false
	}
	renderer.uploadPictures()
	quads = quads[:min(len(quads), maxQuadsPerList)]
	if len(quads) == 0 && len(renderer.painter.Lightings()) == 0 || C.makeQuadIndices(maxQuadsPerList) == 0 {
		return false
	}
	floats := len(quads) * 4 * floatsPerQuadVertex
	if !growLinear(&painted.memory, max(floats, 1)*4) {
		return false
	}
	if len(quads) > 0 {
		painted.runs = writeQuadVertices(unsafe.Slice((*float32)(painted.memory.data), floats), painted.runs, quads, renderer.textureSize, renderer.picturesSize)
		flushLinear(painted.memory.data, floats*4)
	}
	painted.quadCount, painted.sharp = len(quads), anySharp(quads)
	if !renderer.writeLights(painted) {
		return false
	}
	painted.vertices = painted.memory
	painted.width, painted.height = width, height
	painted.textureSize, painted.picturesSize = renderer.textureSize, renderer.picturesSize
	return true
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

// packedColour packs a draw2d colour (in the screen's gamma already) as 0xRRGGBBAA.
func packedColour(colour draw2d.Color) uint32 {
	channel := func(value float32) uint32 { return uint32(min(max(value, 0), 1)*255 + 0.5) }
	return channel(colour[0])<<24 | channel(colour[1])<<16 | channel(colour[2])<<8 | channel(colour[3])
}

// quickFingerprint sums up what a list draws in one number, as draw2d.List.Fingerprint does but
// mixing the values straight in, without writing them out as bytes first: it runs every frame
// on the console, to tell whether the list drew the same as the frame before.
func quickFingerprint(list *draw2d.List) uint64 {
	hash := uint64(14695981039346656037)
	for index := range list.Commands {
		command := &list.Commands[index]
		hash = mixed(hash, uint32(command.Kind)|uint32(command.Layer)<<8|uint32(command.Weight)<<16|uint32(boolBit(command.HasClip))<<24)
		for _, value := range [...]float32{
			command.Area.X, command.Area.Y, command.Area.Width, command.Area.Height,
			command.Clip.X, command.Clip.Y, command.Clip.Width, command.Clip.Height,
			command.Radius, command.BorderWidth, command.Blur, command.Size, command.Baseline,
			command.Fill[0], command.Fill[1], command.Fill[2], command.Fill[3],
			command.Border[0], command.Border[1], command.Border[2], command.Border[3],
		} {
			hash = mixed(hash, math.Float32bits(value))
		}
		for position := 0; position < len(command.Text); position++ {
			hash = mixed(hash, uint32(command.Text[position]))
		}
		hash = mixed(hash, 0xff) // so "ab"+"c" and "a"+"bc" differ
		if command.Picture != nil {
			hash = mixed(hash, uint32(command.Picture.ID()))
			hash = mixed(hash, uint32(command.Picture.Version()))
		}
	}
	return hash
}

// mixed is a hash with a value mixed in, FNV-1a style a word at a time.
func mixed(hash uint64, value uint32) uint64 {
	return (hash ^ uint64(value)) * 1099511628211
}

func boolBit(value bool) int {
	if value {
		return 1
	}
	return 0
}
