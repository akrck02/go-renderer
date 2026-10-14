//go:build android

// The NDK's media decoders, driven from the drawing thread a step at a time: video_advance never
// waits. Pictures come out in the decoder's own buffers, whose layout its output format tells: the
// planes one after another (I420, as software decoders give them) or the luma followed by the two
// chroma interleaved (NV12, as most hardware decoders give them), with rows padded to the decoder's
// liking. Drawing into an image reader instead looked simpler, but some phones' decoders then
// write their pictures compressed in their maker's own way, which no processor can read.

#include <fcntl.h>
#include <media/NdkMediaCodec.h>
#include <media/NdkMediaExtractor.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include "video.h"

// The decoders' colour layouts this reads (MediaCodecInfo.CodecCapabilities' numbers).
enum {
	layoutPlanar = 19,                  // COLOR_FormatYUV420Planar: I420
	layoutSemiPlanar = 21,              // COLOR_FormatYUV420SemiPlanar: NV12
	layoutFlexible = 0x7F420888,        // COLOR_FormatYUV420Flexible: I420 from the decoders that give it
	layoutQualcommAligned = 0x7FA30C04, // NV12 with rows to 128 bytes and planes to 32 rows
};

struct video {
	int file;
	AMediaExtractor *extractor;
	AMediaCodec *codec;
	int width, height;
	int64_t duration;
	int inputEnded, outputEnded;
	ssize_t waiting; // a decoded picture not due yet, -1 when none
	int64_t waitingTime;
	int32_t waitingOffset;
	ssize_t ready; // the latest picture due, for video_take; -1 when none
	int32_t readyOffset;
	int taken; // video_take handed out ready's planes
	// the output's layout
	int32_t layout, stride, sliceHeight, cropLeft, cropTop, cropRight, cropBottom;
};

static video *failed(video *film, char **failure, const char *message) {
	if (failure != NULL) {
		*failure = strdup(message);
	}
	video_close(film);
	return NULL;
}

// chooseTrack selects the first video track and returns its format (to delete) and its type.
static AMediaFormat *chooseTrack(AMediaExtractor *extractor, const char **type) {
	size_t count = AMediaExtractor_getTrackCount(extractor);
	for (size_t index = 0; index < count; index++) {
		AMediaFormat *format = AMediaExtractor_getTrackFormat(extractor, index);
		const char *mime = NULL;
		if (AMediaFormat_getString(format, AMEDIAFORMAT_KEY_MIME, &mime) && strncmp(mime, "video/", 6) == 0) {
			AMediaExtractor_selectTrack(extractor, index);
			*type = mime;
			return format;
		}
		AMediaFormat_delete(format);
	}
	return NULL;
}

video *video_open(const char *path, char **failure) {
	video *film = calloc(1, sizeof(video));
	film->file = -1;
	film->waiting = -1;
	film->ready = -1;
	film->file = open(path, O_RDONLY);
	if (film->file < 0) {
		return failed(film, failure, "video: the film's file does not open");
	}
	off_t length = lseek(film->file, 0, SEEK_END);
	film->extractor = AMediaExtractor_new();
	if (AMediaExtractor_setDataSourceFd(film->extractor, film->file, 0, length) != AMEDIA_OK) {
		return failed(film, failure, "video: the file is not a film the phone reads");
	}
	const char *type = NULL;
	AMediaFormat *format = chooseTrack(film->extractor, &type);
	if (format == NULL) {
		return failed(film, failure, "video: the film has no pictures");
	}
	int32_t width = 0, height = 0;
	AMediaFormat_getInt32(format, AMEDIAFORMAT_KEY_WIDTH, &width);
	AMediaFormat_getInt32(format, AMEDIAFORMAT_KEY_HEIGHT, &height);
	AMediaFormat_getInt64(format, AMEDIAFORMAT_KEY_DURATION, &film->duration);
	film->width = width;
	film->height = height;
	film->codec = AMediaCodec_createDecoderByType(type);
	if (width <= 0 || height <= 0 || film->codec == NULL || AMediaCodec_configure(film->codec, format, NULL, NULL, 0) != AMEDIA_OK ||
	    AMediaCodec_start(film->codec) != AMEDIA_OK) {
		AMediaFormat_delete(format);
		return failed(film, failure, "video: the phone has no decoder for this film");
	}
	AMediaFormat_delete(format);
	return film;
}

void video_size(video *film, int *width, int *height) {
	*width = film->width;
	*height = film->height;
}

int64_t video_duration(video *film) { return film->duration; }

// feed hands the decoder as many of the film's samples as it has room for.
static void feed(video *film) {
	while (!film->inputEnded) {
		ssize_t index = AMediaCodec_dequeueInputBuffer(film->codec, 0);
		if (index < 0) {
			return;
		}
		size_t capacity = 0;
		uint8_t *buffer = AMediaCodec_getInputBuffer(film->codec, index, &capacity);
		ssize_t size = AMediaExtractor_readSampleData(film->extractor, buffer, capacity);
		if (size < 0) {
			AMediaCodec_queueInputBuffer(film->codec, index, 0, 0, 0, AMEDIACODEC_BUFFER_FLAG_END_OF_STREAM);
			film->inputEnded = 1;
			return;
		}
		AMediaCodec_queueInputBuffer(film->codec, index, 0, size, AMediaExtractor_getSampleTime(film->extractor), 0);
		AMediaExtractor_advance(film->extractor);
	}
}

static int32_t number(AMediaFormat *format, const char *key, int32_t otherwise) {
	int32_t value = 0;
	return AMediaFormat_getInt32(format, key, &value) ? value : otherwise;
}

static int32_t roundedUp(int32_t value, int32_t step) { return (value + step - 1) / step * step; }

// readLayout learns how the decoder lays its pictures out, when it says it changed.
static void readLayout(video *film) {
	AMediaFormat *format = AMediaCodec_getOutputFormat(film->codec);
	int32_t width = number(format, AMEDIAFORMAT_KEY_WIDTH, film->width);
	int32_t height = number(format, AMEDIAFORMAT_KEY_HEIGHT, film->height);
	film->layout = number(format, AMEDIAFORMAT_KEY_COLOR_FORMAT, layoutPlanar);
	int32_t rowStep = film->layout == layoutQualcommAligned ? 128 : 1;
	int32_t planeStep = film->layout == layoutQualcommAligned ? 32 : 1;
	film->stride = roundedUp(number(format, AMEDIAFORMAT_KEY_STRIDE, width), rowStep);
	film->sliceHeight = roundedUp(number(format, "slice-height", height), planeStep);
	if (film->sliceHeight < height) {
		film->sliceHeight = height;
	}
	film->cropLeft = number(format, "crop-left", 0);
	film->cropTop = number(format, "crop-top", 0);
	film->cropRight = number(format, "crop-right", film->width - 1) + 1;
	film->cropBottom = number(format, "crop-bottom", film->height - 1) + 1;
	AMediaFormat_delete(format);
}

// keepReady makes a picture the one video_take hands out, letting the one before go.
static void keepReady(video *film, ssize_t index, int32_t offset) {
	video_release(film);
	if (film->ready >= 0) {
		AMediaCodec_releaseOutputBuffer(film->codec, film->ready, false);
	}
	film->ready = index;
	film->readyOffset = offset;
}

// drain takes the decoded pictures due by until, keeping the latest, and holds the first one not
// yet due.
static void drain(video *film, int64_t until, int *fresh) {
	for (int attempts = 0; attempts < 16; attempts++) {
		if (film->waiting < 0 && !film->outputEnded) {
			AMediaCodecBufferInfo info;
			ssize_t index = AMediaCodec_dequeueOutputBuffer(film->codec, &info, 0);
			if (index == AMEDIACODEC_INFO_TRY_AGAIN_LATER) {
				return;
			}
			if (index == AMEDIACODEC_INFO_OUTPUT_FORMAT_CHANGED) {
				readLayout(film);
				continue;
			}
			if (index < 0) {
				continue; // its buffers changed: ask again
			}
			if (info.flags & AMEDIACODEC_BUFFER_FLAG_END_OF_STREAM) {
				film->outputEnded = 1;
			}
			if (info.size <= 0) {
				AMediaCodec_releaseOutputBuffer(film->codec, index, false);
				continue;
			}
			film->waiting = index;
			film->waitingTime = info.presentationTimeUs;
			film->waitingOffset = info.offset;
		}
		if (film->waiting < 0 || film->waitingTime > until) {
			return;
		}
		keepReady(film, film->waiting, film->waitingOffset);
		film->waiting = -1;
		*fresh = 1;
	}
}

int video_advance(video *film, int64_t until, int *ended) {
	int fresh = 0;
	feed(film);
	drain(film, until, &fresh);
	*ended = film->outputEnded && film->waiting < 0;
	return fresh;
}

int video_take(video *film, videoPlanes *planes) {
	if (film->ready < 0 || film->layout == 0) {
		return 0;
	}
	size_t size = 0;
	uint8_t *buffer = AMediaCodec_getOutputBuffer(film->codec, film->ready, &size);
	if (buffer == NULL) {
		return 0;
	}
	const uint8_t *luma = buffer + film->readyOffset;
	int remaining = (int)size - film->readyOffset;
	int lumaPlane = film->stride * film->sliceHeight;
	int semiPlanar = film->layout == layoutSemiPlanar || film->layout == layoutQualcommAligned;
	if (!semiPlanar && film->layout != layoutPlanar && film->layout != layoutFlexible) {
		return 0; // a layout this does not read
	}
	planes->luma = luma;
	planes->lumaLength = remaining;
	planes->lumaRow = film->stride;
	planes->blue = luma + lumaPlane;
	planes->blueLength = remaining - lumaPlane;
	if (semiPlanar) {
		planes->red = planes->blue + 1;
		planes->redLength = planes->blueLength - 1;
		planes->blueRow = film->stride;
		planes->redRow = film->stride;
		planes->bluePixel = 2;
		planes->redPixel = 2;
	} else {
		int chromaRow = film->stride / 2;
		int chromaPlane = chromaRow * (film->sliceHeight / 2);
		planes->red = planes->blue + chromaPlane;
		planes->redLength = planes->blueLength - chromaPlane;
		planes->blueRow = chromaRow;
		planes->redRow = chromaRow;
		planes->bluePixel = 1;
		planes->redPixel = 1;
	}
	planes->left = film->cropLeft;
	planes->top = film->cropTop;
	planes->right = film->cropRight;
	planes->bottom = film->cropBottom;
	film->taken = 1;
	return 1;
}

void video_release(video *film) { film->taken = 0; }

void video_rewind(video *film) {
	video_release(film);
	film->ready = -1; // flushing gives every buffer back to the decoder
	AMediaExtractor_seekTo(film->extractor, 0, AMEDIAEXTRACTOR_SEEK_PREVIOUS_SYNC);
	AMediaCodec_flush(film->codec);
	film->inputEnded = 0;
	film->outputEnded = 0;
	film->waiting = -1;
}

void video_close(video *film) {
	if (film == NULL) {
		return;
	}
	if (film->codec != NULL) {
		AMediaCodec_stop(film->codec);
		AMediaCodec_delete(film->codec);
	}
	if (film->extractor != NULL) {
		AMediaExtractor_delete(film->extractor);
	}
	if (film->file >= 0) {
		close(film->file);
	}
	free(film);
}
