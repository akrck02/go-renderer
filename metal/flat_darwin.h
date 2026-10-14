// The flat renderer's Objective-C side: a CAMetalLayer on a GLFW window, one pipeline, one atlas
// texture and instanced quads. Go calls these through cgo (flat_darwin.go).

#include <stdint.h>

// flatInstance is one quad: where it goes on screen, where it reads the atlas and the pictures
// (all in pixels, left, top, right, bottom; a picture place of -1 reads none) and its straight
// RGBA colour. 52 bytes; the shader reads the same layout (three packed_float4, uchar4).
typedef struct {
	float screen[4];
	float atlas[4];
	float picture[4];
	uint8_t color[4];
} flatInstance;

// flatRendererCreate puts a Metal layer in the window's content view and prepares the pipeline.
// It returns NULL and a message (to free) when Metal cannot start.
void *flatRendererCreate(void *cocoaWindow, int vsync, char **failure);
void flatRendererDestroy(void *renderer);

// flatRendererUploadAtlas replaces the atlas texture's pixels (one byte of coverage each), making a
// new texture when the size changed.
void flatRendererUploadAtlas(void *renderer, const uint8_t *pixels, int size);

// flatRendererUploadPictures sends the images' colours (straight RGBA, size × size pixels): all of
// them in a new texture when whole, else the area x, y, width × height of the current one.
void flatRendererUploadPictures(void *renderer, const uint8_t *pixels, int size, int whole, int x, int y, int width,
                                int height);

// flatRendererDraw clears to the background, draws the instances and shows the frame. It returns
// 0 when there was no drawable to draw into (a hidden or zero-sized window).
int flatRendererDraw(void *renderer, const flatInstance *instances, int count, const float background[4],
                     int width, int height);

// flatRendererCapture draws the same frame into an off-screen texture and copies its pixels, BGRA,
// width × height × 4 bytes, into pixels.
void flatRendererCapture(void *renderer, const flatInstance *instances, int count, const float background[4],
                         int width, int height, uint8_t *pixels);
