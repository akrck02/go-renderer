//go:build android

package android

/*
#cgo LDFLAGS: -lmediandk
#include <stdlib.h>
#include "video.h"
*/
import "C"

import (
	"errors"
	"image"
	"time"
	"unsafe"
)

// Video decodes a film's pictures with the phone's own decoders (the NDK's MediaCodec), a step at
// a time and never waiting, from the goroutine that draws. Its sound is not played.
type Video struct {
	handle        *C.video
	width, height int
	duration      time.Duration
}

// OpenVideo reads a film (MP4 or WebM, as the phone's decoders read it: H.264, H.265, VP8, VP9,
// AV1) from a file and starts decoding its pictures.
func OpenVideo(path string) (*Video, error) {
	name := C.CString(path)
	defer C.free(unsafe.Pointer(name))
	var failure *C.char
	handle := C.video_open(name, &failure)
	if handle == nil {
		defer C.free(unsafe.Pointer(failure))
		return nil, errors.New(C.GoString(failure))
	}
	var width, height C.int
	C.video_size(handle, &width, &height)
	duration := time.Duration(C.video_duration(handle)) * time.Microsecond
	return &Video{handle: handle, width: int(width), height: int(height), duration: duration}, nil
}

// Size is the film's pictures' width and height, in pixels.
func (video *Video) Size() (width, height int) { return video.width, video.height }

// Duration is how long the film lasts; 0 when its file does not say.
func (video *Video) Duration() time.Duration { return video.duration }

// Advance decodes the film up to a moment and, when a picture due by then has arrived, copies it
// into pixels (as much of it as fits). It tells whether pixels changed and whether the film ended.
func (video *Video) Advance(until time.Duration, pixels *image.NRGBA) (changed, ended bool) {
	var finished C.int
	ready := C.video_advance(video.handle, C.int64_t(until.Microseconds()), &finished) != 0
	if ready {
		changed = video.copyPicture(pixels)
	}
	return changed, finished != 0
}

// copyPicture turns the picture the reader holds into colours.
func (video *Video) copyPicture(pixels *image.NRGBA) bool {
	var planes C.videoPlanes
	if C.video_take(video.handle, &planes) == 0 {
		return false
	}
	defer C.video_release(video.handle)
	YUVPlanes{
		Luma:      unsafe.Slice((*byte)(planes.luma), int(planes.lumaLength)),
		Blue:      unsafe.Slice((*byte)(planes.blue), int(planes.blueLength)),
		Red:       unsafe.Slice((*byte)(planes.red), int(planes.redLength)),
		LumaRow:   int(planes.lumaRow),
		BlueRow:   int(planes.blueRow),
		RedRow:    int(planes.redRow),
		BluePixel: int(planes.bluePixel),
		RedPixel:  int(planes.redPixel),
		Crop:      image.Rect(int(planes.left), int(planes.top), int(planes.right), int(planes.bottom)),
	}.ToRGBA(pixels)
	return true
}

// Rewind goes back to the film's start.
func (video *Video) Rewind() { C.video_rewind(video.handle) }

// Close lets go of the decoder and the file.
func (video *Video) Close() {
	if video.handle != nil {
		C.video_close(video.handle)
		video.handle = nil
	}
}
