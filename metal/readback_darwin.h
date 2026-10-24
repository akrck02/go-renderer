// Reading a frame back from the GPU, for captures, on every Mac.
#import <Metal/Metal.h>

// readableStorage is where a texture the processor reads back lives. On Apple silicon the
// processor and the GPU share memory, so Shared storage is read directly (Managed storage does
// nothing there, and macOS 27 deprecates it). An Intel Mac's GPU keeps its own copy, which only
// Managed storage brings back.
static inline MTLStorageMode readableStorage(id<MTLDevice> device) {
	if (device.hasUnifiedMemory) {
		return MTLStorageModeShared;
	}
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
	return MTLStorageModeManaged; // the only way back on an Intel Mac's own GPU memory
#pragma clang diagnostic pop
}

// bringBack asks the GPU to copy a texture back for the processor, when its storage needs it.
static inline void bringBack(id<MTLCommandBuffer> commands, id<MTLTexture> texture) {
	if (texture.storageMode == MTLStorageModeShared) {
		return; // already where the processor reads it
	}
	id<MTLBlitCommandEncoder> blit = [commands blitCommandEncoder];
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
	[blit synchronizeResource:texture]; // Managed storage, on an Intel Mac
#pragma clang diagnostic pop
	[blit endEncoding];
}
