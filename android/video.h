// A film's pictures, decoded by the phone's own decoders (the NDK's MediaCodec) into YUV planes
// that Go turns into colours (video.go). The sound is not played.

#ifndef GO_RENDERER_ANDROID_VIDEO_H
#define GO_RENDERER_ANDROID_VIDEO_H

#include <stdint.h>

typedef struct video video;

// videoPlanes is a decoded picture: its luma and its two chroma planes, each with how far apart its
// rows and its pixels are, and the part of it that shows.
typedef struct {
	const uint8_t *luma, *blue, *red;
	int lumaLength, blueLength, redLength;
	int lumaRow, blueRow, redRow;
	int bluePixel, redPixel;
	int left, top, right, bottom;
} videoPlanes;

// video_open reads a film from a file and starts decoding its first video track. It returns NULL
// and a message (to free) when it cannot.
video *video_open(const char *path, char **failure);
void video_size(video *film, int *width, int *height);
int64_t video_duration(video *film); // microseconds; 0 when the file does not say

// video_advance decodes up to a moment (microseconds into the film) and makes the latest picture
// due by then ready to take. It returns 1 when one is ready, and sets ended at the film's end.
int video_advance(video *film, int64_t until, int *ended);
// video_take gives the ready picture's planes, valid until the next video_advance or video_rewind;
// 0 when there is none, or its layout is one this does not read.
int video_take(video *film, videoPlanes *planes);
void video_release(video *film);

void video_rewind(video *film);
void video_close(video *film);

#endif
