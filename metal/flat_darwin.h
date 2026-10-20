// The flat renderer's Objective-C side: a CAMetalLayer on a GLFW window, one pipeline, one atlas
// texture and instanced quads. Go calls these through cgo (flat_darwin.go).

#include <stdint.h>

// flatInstance is one quad: where it goes on screen, where it reads the atlas and the pictures
// (all in pixels, left, top, right, bottom; a picture place of -1 reads none) and its straight
// RGBA colour. 52 bytes; the shader reads the same layout (three packed_float4, uchar4). A picture
// place of -2 marks a backdrop's quad, which shows the blurred snapshot.
typedef struct {
	float screen[4];
	float atlas[4];
	float picture[4];
	uint8_t color[4];
} flatInstance;

// flatSnapshot is one of the overlay's snapshots (see overlay.Snapshot): before the instance at,
// the frame drawn so far is halved halvings times and blurred with these weights (the centre's,
// then each tap's on one side, four taps). 28 bytes, as the Go side writes it.
typedef struct {
	int32_t at;
	int32_t halvings;
	float weights[5];
} flatSnapshot;

// flatLight is one of the overlay's 2D lights (see overlay.LightQuad): where it is on screen, its
// colour (alpha: its intensity), the picture it shines through (-1 for a point light, which is a
// disc full within inner, a share of its half side) and inner. 64 bytes, as lights.go writes it.
typedef struct {
	float screen[4];
	float color[4];
	float picture[4];
	float inner;
	float unused[3];
} flatLight;

// flatLighting is one of the overlay's lightings (see overlay.Lighting): before the instance at,
// the lights from first to last are added into a picture of light filled with ambient (at half,
// already), which multiplies the area (left, top, right, bottom on screen). 48 bytes.
typedef struct {
	int32_t at;
	int32_t first;
	int32_t last;
	int32_t unused;
	float ambient[4];
	float area[4];
} flatLighting;

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

// flatRendererDraw clears to the background, draws the instances, blurring the frame at each
// snapshot for the backdrops after it and lighting it at each lighting, and shows the frame. It
// returns 0 when there was no drawable to draw into (a hidden or zero-sized window).
int flatRendererDraw(void *renderer, const flatInstance *instances, int count, const flatSnapshot *snapshots,
                     int snapshotCount, const flatLight *lights, int lightCount, const flatLighting *lightings,
                     int lightingCount, const float background[4], int width, int height);

// flatRendererCapture draws the same frame into an off-screen texture and copies its pixels, BGRA,
// width × height × 4 bytes, into pixels.
void flatRendererCapture(void *renderer, const flatInstance *instances, int count, const flatSnapshot *snapshots,
                         int snapshotCount, const flatLight *lights, int lightCount, const flatLighting *lightings,
                         int lightingCount, const float background[4], int width, int height, uint8_t *pixels);
