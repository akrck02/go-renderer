//go:build nintendo3ds

package nintendo3ds

/*
#cgo CFLAGS: -Wno-ignored-attributes
#cgo LDFLAGS: -lcitro3d
#include <3ds.h>
#include <citro3d.h>
#include <malloc.h>
#include <stdlib.h>
#include <string.h>

// How the finished frame is copied to the screen: RGBA8 rendered, RGB8 shown, no scaling.
#define DISPLAY_TRANSFER_FLAGS \
	(GX_TRANSFER_FLIP_VERT(0) | GX_TRANSFER_OUT_TILED(0) | GX_TRANSFER_RAW_COPY(0) | \
	GX_TRANSFER_IN_FORMAT(GX_TRANSFER_FMT_RGBA8) | GX_TRANSFER_OUT_FORMAT(GX_TRANSFER_FMT_RGB8) | \
	GX_TRANSFER_SCALING(GX_TRANSFER_SCALE_NO))

static C3D_RenderTarget* openTopScreen(void) {
	// On a New 3DS, the processor at 804 MHz with its second-level cache instead of 268 MHz
	// without: libctru leaves it slow unless asked. Nothing changes on an older 3DS.
	osSetSpeedupEnable(true);
	gfxInitDefault();
	if (!C3D_Init(C3D_DEFAULT_CMDBUF_SIZE)) return NULL;
	// The top screen is a 240×400 portrait panel on its side, hence height and width swapped.
	// C3D_DEPTHTYPE is a transparent union, which GCC (devkitARM) reads as an int and clang does
	// not, so the depth format is set through the union; it is passed in a register either way.
	C3D_DEPTHTYPE depth;
	depth.__i = GPU_RB_DEPTH24_STENCIL8;
	C3D_RenderTarget* target = C3D_RenderTargetCreate(240, 400, GPU_RB_RGBA8, depth);
	if (target == NULL) return NULL;
	C3D_RenderTargetSetOutput(target, GFX_TOP, GFX_LEFT, DISPLAY_TRANSFER_FLAGS);

	// The first colour stage passes the vertex colour through; the PICA200 has no fragment
	// shaders, only these fixed stages.
	C3D_TexEnv* environment = C3D_GetTexEnv(0);
	C3D_TexEnvInit(environment);
	C3D_TexEnvSrc(environment, C3D_Both, GPU_PRIMARY_COLOR, 0, 0);
	C3D_TexEnvFunc(environment, C3D_Both, GPU_REPLACE);
	return target;
}

// openRightEye makes the top screen's second picture, for the right eye in 3D.
static C3D_RenderTarget* openRightEye(void) {
	C3D_DEPTHTYPE depth;
	depth.__i = GPU_RB_DEPTH24_STENCIL8;
	C3D_RenderTarget* target = C3D_RenderTargetCreate(240, 400, GPU_RB_RGBA8, depth);
	if (target != NULL) C3D_RenderTargetSetOutput(target, GFX_TOP, GFX_RIGHT, DISPLAY_TRANSFER_FLAGS);
	return target;
}

static void set3D(int on) {
	gfxSet3D(on != 0);
}

static float slider3D(void) {
	return osGet3DSliderState();
}

static void drawOn(C3D_RenderTarget* target) {
	C3D_FrameDrawOn(target);
}

static void closeScreen(C3D_RenderTarget* target) {
	C3D_RenderTargetDelete(target);
	C3D_Fini();
	gfxExit();
}

static void frameBegin(void) {
	C3D_FrameBegin(C3D_FRAME_SYNCDRAW);
}

// clearOn clears a picture and makes the next draws go to it.
static void clearOn(C3D_RenderTarget* target, u32 clearColour) {
	C3D_RenderTargetClear(target, C3D_CLEAR_ALL, clearColour, 0);
	C3D_FrameDrawOn(target);
}

static void endFrame(void) {
	C3D_FrameEnd(0);
}

// loadShader keeps the shader binary and the program in C memory: citro3d holds pointers to
// both while they are in use, which Go memory must not be lent for.
static shaderProgram_s* loadShader(const void* binary, u32 size, DVLB_s** parsed, void** copy) {
	*copy = memalign(4, size);
	if (*copy == NULL) return NULL;
	memcpy(*copy, binary, size);
	*parsed = DVLB_ParseFile((u32*)*copy, size);
	if (*parsed == NULL) return NULL;
	shaderProgram_s* program = malloc(sizeof(shaderProgram_s));
	if (program == NULL) return NULL;
	shaderProgramInit(program);
	shaderProgramSetVsh(program, &(*parsed)->DVLE[0]);
	return program;
}

static void freeShader(shaderProgram_s* program, DVLB_s* parsed, void* copy) {
	shaderProgramFree(program);
	free(program);
	DVLB_Free(parsed);
	free(copy);
}

static int uniformLocation(shaderProgram_s* program, const char* name) {
	return shaderInstanceGetUniformLocation(program->vertexShader, name);
}

// setMatrix takes the 16 floats already laid out the PICA200's way (nintendo3ds.GPUMatrix).
static void setMatrix(int location, const float* values) {
	C3D_Mtx matrix;
	memcpy(matrix.m, values, sizeof matrix.m);
	C3D_FVUnifMtx4x4(GPU_VERTEX_SHADER, location, &matrix);
}

// drawTriangles describes one buffer of interleaved float attributes (sizes, one per input
// register v0, v1, …) and draws it as triangles.
static void drawTriangles(void* vertices, int stride, const u8* sizes, int attributes, int count) {
	C3D_AttrInfo* attributeInfo = C3D_GetAttrInfo();
	AttrInfo_Init(attributeInfo);
	u64 permutation = 0;
	for (int index = 0; index < attributes; index++) {
		AttrInfo_AddLoader(attributeInfo, index, GPU_FLOAT, sizes[index]);
		permutation |= (u64)index << (index * 4);
	}
	C3D_BufInfo* bufferInfo = C3D_GetBufInfo();
	BufInfo_Init(bufferInfo);
	BufInfo_Add(bufferInfo, vertices, stride, attributes, permutation);
	C3D_DrawArrays(GPU_TRIANGLES, 0, count);
}

static void* copyToLinearMemory(const void* data, u32 size) {
	void* linear = linearAlloc(size);
	if (linear == NULL) return NULL;
	memcpy(linear, data, size);
	GSPGPU_FlushDataCache(linear, size);
	return linear;
}

static void openBottomConsole(void) {
	consoleInit(GFX_BOTTOM, NULL);
}

// orthographicTiltFromCitro3D is citro3d's own Mtx_OrthoTilt, right-handed like go-renderer, to
// check nintendo3ds.OrthographicTilt.
static void orthographicTiltFromCitro3D(float* out, float left, float right, float bottom, float top, float near, float far) {
	C3D_Mtx matrix;
	Mtx_OrthoTilt(&matrix, left, right, bottom, top, near, far, false);
	memcpy(out, matrix.m, sizeof matrix.m);
}
*/
import "C"

import (
	"unsafe"

	"github.com/akrck02/go-renderer/failure"
	"github.com/akrck02/go-renderer/graphics"
)

// Running reports whether the program should go on; it is false when the system asks it to
// close (the HOME menu, the power button).
func Running() bool {
	return bool(C.aptMainLoop())
}

// OpenSecondScreenConsole turns the bottom screen into a text console for println, after Open.
func OpenSecondScreenConsole() {
	C.openBottomConsole()
}

// Screen is the top screen, drawn by citro3d: one picture, or one for each eye in 3D.
type Screen struct {
	target *C.C3D_RenderTarget
	right  *C.C3D_RenderTarget // made the first time a frame is drawn in 3D
	stereo bool                // the screen shows two pictures
	// drawing is the picture the draws go to now: the one picture, or an eye's; a pass that drew
	// elsewhere (the 2D lights) comes back to it
	drawing *C.C3D_RenderTarget
}

// OpenScreen starts the graphics system and prepares the main (top) screen for drawing.
func OpenScreen() (*Screen, error) {
	target := C.openTopScreen()
	if target == nil {
		return nil, failure.Wrap(ScreenNotOpened, "the top screen", nil)
	}
	return &Screen{target: target}, nil
}

// Close stops the graphics system.
func (screen *Screen) Close() {
	if screen.right != nil {
		C.C3D_RenderTargetDelete(screen.right)
	}
	C.closeScreen(screen.target)
}

// BeginFrame waits for the previous frame and starts one: the top screen and the bottom one are
// drawn between it and EndFrame.
func BeginFrame() {
	C.frameBegin()
	frameNumber++
}

// frameNumber counts the frames begun: what is kept for one frame knows when the next starts.
var frameNumber uint32

// EndFrame sends what was drawn this frame to the GPU and shows it.
func EndFrame() {
	C.endFrame()
}

// Clear clears the screen for one picture, in a frame: the screen leaves 3D, and the next draws go
// to it.
func (screen *Screen) Clear(clearColour uint32) {
	screen.setStereo(false)
	C.clearOn(screen.target, C.u32(clearColour))
	screen.drawing = screen.target
}

// ClearStereo clears both eyes' pictures in a frame; the next draws go to the left one.
func (screen *Screen) ClearStereo(clearColour uint32) bool {
	if screen.right == nil {
		screen.right = C.openRightEye()
	}
	if screen.right == nil {
		screen.Clear(clearColour)
		return false
	}
	screen.setStereo(true)
	C.clearOn(screen.right, C.u32(clearColour))
	C.clearOn(screen.target, C.u32(clearColour))
	screen.drawing = screen.target
	return true
}

// DrawOn makes the next draws go to one eye's picture, in a stereo frame.
func (screen *Screen) DrawOn(eye Eye) {
	screen.drawing = screen.target
	if eye == RightEye && screen.right != nil {
		screen.drawing = screen.right
	}
	C.drawOn(screen.drawing)
}

// resume makes the next draws go back to the picture they went to before a pass drew elsewhere.
func (screen *Screen) resume() {
	if screen.drawing != nil {
		C.drawOn(screen.drawing)
	}
}

func (screen *Screen) setStereo(stereo bool) {
	if screen.stereo == stereo {
		return
	}
	screen.stereo = stereo
	on := 0
	if stereo {
		on = 1
	}
	C.set3D(C.int(on))
}

// DepthSlider is the position of the 3D slider, from 0 (off) to 1.
func DepthSlider() float32 {
	return float32(C.slider3D())
}

// Shader is a vertex shader for the PICA200, assembled by devkitPro's picasso (a .shbin file).
type Shader struct {
	program *C.shaderProgram_s
	parsed  *C.DVLB_s
	copy    unsafe.Pointer
}

// LoadShader reads a .shbin binary. The bytes are copied, so the slice can be dropped.
func LoadShader(binary []byte) (*Shader, error) {
	if len(binary) == 0 {
		return nil, failure.Wrap(InvalidShader, "empty", nil)
	}
	shader := &Shader{}
	shader.program = C.loadShader(unsafe.Pointer(&binary[0]), C.u32(len(binary)), &shader.parsed, &shader.copy)
	if shader.program == nil {
		return nil, InvalidShader
	}
	return shader, nil
}

// Use makes the shader the one later draws go through.
func (shader *Shader) Use() {
	C.C3D_BindProgram(shader.program)
}

// Uniform answers where the shader reads a uniform, or -1 if it has none of that name.
func (shader *Shader) Uniform(name string) int {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return int(C.uniformLocation(shader.program, cName))
}

// Free releases the shader.
func (shader *Shader) Free() {
	C.freeShader(shader.program, shader.parsed, shader.copy)
}

// SetMatrix sets a 4×4 matrix uniform of the vertex shader in use.
func SetMatrix(location int, matrix graphics.Matrix) {
	if location < 0 {
		return
	}
	laidOut := GPUMatrix(matrix)
	C.setMatrix(C.int(location), (*C.float)(unsafe.Pointer(&laidOut[0])))
}

// VertexBuffer is a list of vertices in the linear memory the GPU reads, outside the Go heap,
// so the garbage collector never scans it.
type VertexBuffer struct {
	data     unsafe.Pointer
	sizes    []uint8 // floats of each attribute, in input register order
	stride   int     // bytes per vertex
	vertices int
}

// NewVertexBuffer copies interleaved float attributes to linear memory. sizes gives how many
// floats each attribute has, for instance {3, 4} for a position and a colour.
func NewVertexBuffer(values []float32, sizes []uint8) (*VertexBuffer, error) {
	floatsPerVertex := 0
	for _, size := range sizes {
		floatsPerVertex += int(size)
	}
	if floatsPerVertex == 0 || len(values) == 0 || len(values)%floatsPerVertex != 0 {
		return nil, InvalidVertices
	}
	bytes := len(values) * 4
	data := C.copyToLinearMemory(unsafe.Pointer(&values[0]), C.u32(bytes))
	if data == nil {
		return nil, OutOfLinearMemory
	}
	return &VertexBuffer{data: data, sizes: append([]uint8(nil), sizes...), stride: floatsPerVertex * 4, vertices: len(values) / floatsPerVertex}, nil
}

// DrawTriangles draws the buffer's vertices, three by three, through the shader in use.
func (buffer *VertexBuffer) DrawTriangles() {
	C.drawTriangles(buffer.data, C.int(buffer.stride), (*C.u8)(unsafe.Pointer(&buffer.sizes[0])), C.int(len(buffer.sizes)), C.int(buffer.vertices))
}

// copyToLinearMemory copies size bytes to new linear memory, which the GPU reads; nil when it is
// full.
func copyToLinearMemory(data unsafe.Pointer, size int) unsafe.Pointer {
	return C.copyToLinearMemory(data, C.u32(size))
}

func freeLinearMemory(data unsafe.Pointer) {
	if data != nil {
		C.linearFree(data)
	}
}

// ProcessingTime is how long, in milliseconds, the last frame took to prepare on the processor,
// as citro3d measures it.
func ProcessingTime() float32 {
	return float32(C.C3D_GetProcessingTime())
}

// DrawingTime is how long, in milliseconds, the GPU took to draw the last frame, as citro3d
// measures it.
func DrawingTime() float32 {
	return float32(C.C3D_GetDrawingTime())
}

// Free releases the buffer's linear memory.
func (buffer *VertexBuffer) Free() {
	C.linearFree(buffer.data)
}

// ReferenceOrthographicTilt is citro3d's right-handed Mtx_OrthoTilt, laid out the PICA200's way, to check
// OrthographicTilt against on the console.
func ReferenceOrthographicTilt(left, right, bottom, top, near, far float32) [16]float32 {
	var laidOut [16]float32
	C.orthographicTiltFromCitro3D((*C.float)(unsafe.Pointer(&laidOut[0])), C.float(left), C.float(right), C.float(bottom), C.float(top), C.float(near), C.float(far))
	return laidOut
}
