// The scene renderer's Objective-C side: a thin layer over Metal (device, buffers, textures,
// pipelines, passes, draw calls). The Go side (scene_darwin.go) decides what to draw, in the same
// order as the OpenGL scene renderer. Every object returned is retained; sceneRelease lets go of it.

#include <stdint.h>

// Texture formats.
enum {
	sceneFormatBGRA8 = 1,     // a frame's colours
	sceneFormatR8 = 2,        // the overlay atlas (coverage)
	sceneFormatRG32Float = 3, // the water level variation (two harmonics)
	sceneFormatRGBA8sRGB = 4, // sky images, read as linear colours
	sceneFormatDepth32 = 5,   // shadow and seabed maps
};

// Blending of a pipeline.
enum {
	sceneBlendNone = 0,
	sceneBlendAlpha = 1,   // source alpha over the destination, for colour and alpha alike
	sceneBlendOverlay = 2, // colour by source alpha; alpha by one (the overlay's BlendFuncSeparate)
};

// Depth comparisons.
enum {
	sceneCompareAlways = 0,
	sceneCompareLess = 1,
	sceneCompareLessEqual = 2,
};

// Primitives.
enum {
	scenePrimitiveTriangles = 0,
	scenePrimitiveTriangleStrip = 1,
	scenePrimitiveLines = 2,
	scenePrimitivePoints = 3,
};

void sceneRelease(void *object);

// sceneDeviceCreate puts a Metal layer in the window (made with no OpenGL context). It returns NULL
// and a message to free when Metal cannot start.
void *sceneDeviceCreate(void *cocoaWindow, int vsync, char **failure);
void *sceneLibraryCreate(void *device, const char *source, char **failure);
// scenePipelineCreate makes a pipeline drawing into the frame (colour and depth), or into a depth
// map alone when fragmentName is NULL.
void *scenePipelineCreate(void *device, void *library, const char *vertexName, const char *fragmentName, int blending,
                          char **failure);
void *sceneDepthStateCreate(void *device, int compare, int write);

void *sceneBufferCreate(void *device, const void *bytes, long length);
void sceneBufferWrite(void *buffer, long offset, const void *bytes, long length);

void *sceneTextureCreate(void *device, int format, int width, int height, int cube, int mipmapped, int renderTarget);
// sceneTextureWrite fills one slice (a cube face, or 0) of the top level of a texture.
void sceneTextureWrite(void *texture, int slice, int width, int height, const void *bytes, long bytesPerRow);
void sceneTextureGenerateMipmaps(void *device, void *texture);

// sceneFrameBegin starts a frame of width × height pixels: into the window, or into an off-screen
// picture to read back when offscreen is 1. It waits for the previous frame to finish, so a frame
// may rewrite buffers the last one used. It returns NULL when the window has nothing to draw into.
void *sceneFrameBegin(void *device, int width, int height, int offscreen);
// scenePassBeginDepth starts drawing into a depth map, cleared to the farthest depth.
void *scenePassBeginDepth(void *frame, void *depthTexture);
// scenePassBeginMain starts drawing the picture: colours cleared to clear, and a depth buffer.
void *scenePassBeginMain(void *frame, const float clear[4]);

void sceneEncoderSetPipeline(void *encoder, void *pipeline);
void sceneEncoderSetDepthState(void *encoder, void *state);
void sceneEncoderSetDepthBias(void *encoder, float bias, float slope);
void sceneEncoderSetVertexBuffer(void *encoder, void *buffer, int index);
void sceneEncoderSetVertexBytes(void *encoder, const void *bytes, long length, int index);
void sceneEncoderSetFragmentBytes(void *encoder, const void *bytes, long length, int index);
void sceneEncoderSetVertexTexture(void *encoder, void *texture, int index);
void sceneEncoderSetFragmentTexture(void *encoder, void *texture, int index);
void sceneEncoderDraw(void *encoder, int primitive, int start, int count, int instances);
void sceneEncoderDrawIndexed(void *encoder, int count, void *indexBuffer, int instances);
void sceneEncoderEnd(void *encoder);

// sceneFrameFinish sends the frame to the GPU and shows it. With pixels (an off-screen frame) it
// waits and copies the picture, BGRA, width × height × 4 bytes; with gpuSeconds it waits and says
// how long the GPU took.
void sceneFrameFinish(void *frame, uint8_t *pixels, double *gpuSeconds);
