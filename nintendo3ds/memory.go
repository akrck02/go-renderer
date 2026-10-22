//go:build nintendo3ds

package nintendo3ds

/*
#include <3ds.h>

static void readMemory(u32* linearFree, u32* videoFree, s64* applicationFree, s64* applicationSize) {
	*linearFree = linearSpaceFree();
	*videoFree = vramSpaceFree();
	*applicationFree = osGetMemRegionFree(MEMREGION_APPLICATION);
	*applicationSize = osGetMemRegionSize(MEMREGION_APPLICATION);
}
*/
import "C"

// Memory is how much of the console's memory is free, in bytes: the linear memory the GPU reads
// (vertex buffers, textures), the video memory (screens, depth buffers) and the application's
// region, which holds the Go heap.
type Memory struct {
	LinearFree, VideoFree            uint64
	ApplicationFree, ApplicationSize uint64
}

// ReadMemory reads how much memory is free now. It is cheap: a few calls into the system.
func ReadMemory() Memory {
	var linearFree, videoFree C.u32
	var applicationFree, applicationSize C.s64
	C.readMemory(&linearFree, &videoFree, &applicationFree, &applicationSize)
	return Memory{LinearFree: uint64(linearFree), VideoFree: uint64(videoFree),
		ApplicationFree: uint64(applicationFree), ApplicationSize: uint64(applicationSize)}
}
