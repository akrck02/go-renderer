// The scene renderer's thin layer over Metal: see scene_darwin.h for what each function does.

#import <Cocoa/Cocoa.h>
#import <Metal/Metal.h>
#import <QuartzCore/CAMetalLayer.h>
#include <stdlib.h>
#include <string.h>
#include "scene_darwin.h"

@interface SceneDevice : NSObject
@property(nonatomic, strong) NSWindow *window;
@property(nonatomic, strong) CAMetalLayer *layer;
@property(nonatomic, strong) id<MTLDevice> device;
@property(nonatomic, strong) id<MTLCommandQueue> queue;
@property(nonatomic, strong) id<MTLTexture> depth; // the frame's depth buffer, made again when the size changes
@property(nonatomic, strong) dispatch_semaphore_t previousFrame;
@end

@implementation SceneDevice
@end

@interface SceneFrame : NSObject
@property(nonatomic, strong) SceneDevice *owner;
@property(nonatomic, strong) id<MTLCommandBuffer> commands;
@property(nonatomic, strong) id<CAMetalDrawable> drawable; // nil for an off-screen frame
@property(nonatomic, strong) id<MTLTexture> target;        // where the picture goes
@property(nonatomic) BOOL resizing;
@end

@implementation SceneFrame
@end

static void setFailure(char **failure, NSString *message) {
	if (failure != NULL) {
		*failure = strdup(message.UTF8String);
	}
}

static void *retained(id object) { return object == nil ? NULL : (__bridge_retained void *)object; }

void sceneRelease(void *object) {
	if (object != NULL) {
		CFRelease(object);
	}
}

// chooseDevice prefers a low-power GPU when a Mac has two.
static id<MTLDevice> chooseDevice(void) {
	for (id<MTLDevice> device in MTLCopyAllDevices()) {
		if (device.isLowPower) {
			return device;
		}
	}
	return MTLCreateSystemDefaultDevice();
}

void *sceneDeviceCreate(void *cocoaWindow, int vsync, char **failure) {
	@autoreleasepool {
		id<MTLDevice> device = chooseDevice();
		if (device == nil) {
			setFailure(failure, @"metal: this computer has no Metal device");
			return NULL;
		}
		SceneDevice *scene = [SceneDevice new];
		scene.device = device;
		scene.queue = [device newCommandQueue];
		scene.window = (__bridge NSWindow *)cocoaWindow;
		scene.previousFrame = dispatch_semaphore_create(1);
		CAMetalLayer *layer = [CAMetalLayer layer];
		layer.device = device;
		layer.pixelFormat = MTLPixelFormatBGRA8Unorm;
		layer.framebufferOnly = YES;
		layer.opaque = YES;
		layer.displaySyncEnabled = vsync != 0;
		layer.maximumDrawableCount = 2;
		layer.contentsScale = scene.window.backingScaleFactor;
		NSView *view = scene.window.contentView;
		view.layer = layer;
		view.wantsLayer = YES;
		scene.layer = layer;
		return retained(scene);
	}
}

void *sceneLibraryCreate(void *handle, const char *source, char **failure) {
	@autoreleasepool {
		SceneDevice *scene = (__bridge SceneDevice *)handle;
		// precise maths: the noise behind patterns, stars and clouds takes sines of large numbers, which
		// fast maths would round differently from OpenGL
		MTLCompileOptions *options = [MTLCompileOptions new];
		if (@available(macOS 15.0, *)) {
			options.mathMode = MTLMathModeSafe;
		} else {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
			options.fastMathEnabled = NO; // the same, before macOS 15
#pragma clang diagnostic pop
		}
		NSError *error = nil;
		id<MTLLibrary> library = [scene.device newLibraryWithSource:[NSString stringWithUTF8String:source] options:options error:&error];
		if (library == nil) {
			setFailure(failure, [NSString stringWithFormat:@"metal: compiling the scene shaders: %@", error.localizedDescription]);
		}
		return retained(library);
	}
}

static void applyBlending(MTLRenderPipelineColorAttachmentDescriptor *colour, int blending) {
	if (blending == sceneBlendNone) {
		return;
	}
	colour.blendingEnabled = YES;
	colour.sourceRGBBlendFactor = MTLBlendFactorSourceAlpha;
	colour.destinationRGBBlendFactor = MTLBlendFactorOneMinusSourceAlpha;
	colour.destinationAlphaBlendFactor = MTLBlendFactorOneMinusSourceAlpha;
	colour.sourceAlphaBlendFactor = blending == sceneBlendOverlay ? MTLBlendFactorOne : MTLBlendFactorSourceAlpha;
}

void *scenePipelineCreate(void *handle, void *libraryHandle, const char *vertexName, const char *fragmentName, int blending,
                          char **failure) {
	@autoreleasepool {
		SceneDevice *scene = (__bridge SceneDevice *)handle;
		id<MTLLibrary> library = (__bridge id<MTLLibrary>)libraryHandle;
		MTLRenderPipelineDescriptor *descriptor = [MTLRenderPipelineDescriptor new];
		descriptor.vertexFunction = [library newFunctionWithName:[NSString stringWithUTF8String:vertexName]];
		descriptor.depthAttachmentPixelFormat = MTLPixelFormatDepth32Float;
		if (fragmentName != NULL) {
			descriptor.fragmentFunction = [library newFunctionWithName:[NSString stringWithUTF8String:fragmentName]];
			descriptor.colorAttachments[0].pixelFormat = MTLPixelFormatBGRA8Unorm;
			applyBlending(descriptor.colorAttachments[0], blending);
		}
		NSError *error = nil;
		id<MTLRenderPipelineState> pipeline = [scene.device newRenderPipelineStateWithDescriptor:descriptor error:&error];
		if (pipeline == nil) {
			setFailure(failure, [NSString stringWithFormat:@"metal: making the %s pipeline: %@", vertexName, error.localizedDescription]);
		}
		return retained(pipeline);
	}
}

void *sceneDepthStateCreate(void *handle, int compare, int write) {
	@autoreleasepool {
		SceneDevice *scene = (__bridge SceneDevice *)handle;
		MTLDepthStencilDescriptor *descriptor = [MTLDepthStencilDescriptor new];
		switch (compare) {
		case sceneCompareLess:
			descriptor.depthCompareFunction = MTLCompareFunctionLess;
			break;
		case sceneCompareLessEqual:
			descriptor.depthCompareFunction = MTLCompareFunctionLessEqual;
			break;
		default:
			descriptor.depthCompareFunction = MTLCompareFunctionAlways;
		}
		descriptor.depthWriteEnabled = write != 0;
		return retained([scene.device newDepthStencilStateWithDescriptor:descriptor]);
	}
}

void *sceneBufferCreate(void *handle, const void *bytes, long length) {
	@autoreleasepool {
		SceneDevice *scene = (__bridge SceneDevice *)handle;
		NSUInteger size = MAX((NSUInteger)length, 16);
		id<MTLBuffer> buffer = [scene.device newBufferWithLength:size options:MTLResourceStorageModeShared];
		if (bytes != NULL && length > 0) {
			memcpy(buffer.contents, bytes, length);
		}
		return retained(buffer);
	}
}

void sceneBufferWrite(void *handle, long offset, const void *bytes, long length) {
	id<MTLBuffer> buffer = (__bridge id<MTLBuffer>)handle;
	memcpy((uint8_t *)buffer.contents + offset, bytes, length);
}

static MTLPixelFormat pixelFormat(int format) {
	switch (format) {
	case sceneFormatR8:
		return MTLPixelFormatR8Unorm;
	case sceneFormatRG32Float:
		return MTLPixelFormatRG32Float;
	case sceneFormatRGBA8sRGB:
		return MTLPixelFormatRGBA8Unorm_sRGB;
	case sceneFormatDepth32:
		return MTLPixelFormatDepth32Float;
	default:
		return MTLPixelFormatBGRA8Unorm;
	}
}

void *sceneTextureCreate(void *handle, int format, int width, int height, int cube, int mipmapped, int renderTarget) {
	@autoreleasepool {
		SceneDevice *scene = (__bridge SceneDevice *)handle;
		MTLTextureDescriptor *descriptor;
		if (cube) {
			descriptor = [MTLTextureDescriptor textureCubeDescriptorWithPixelFormat:pixelFormat(format) size:width mipmapped:mipmapped != 0];
		} else {
			descriptor = [MTLTextureDescriptor texture2DDescriptorWithPixelFormat:pixelFormat(format)
			                                                                width:width
			                                                               height:height
			                                                            mipmapped:mipmapped != 0];
		}
		descriptor.usage = MTLTextureUsageShaderRead;
		if (renderTarget) {
			descriptor.usage |= MTLTextureUsageRenderTarget;
			descriptor.storageMode = MTLStorageModePrivate;
		}
		return retained([scene.device newTextureWithDescriptor:descriptor]);
	}
}

void sceneTextureWrite(void *handle, int slice, int width, int height, const void *bytes, long bytesPerRow) {
	id<MTLTexture> texture = (__bridge id<MTLTexture>)handle;
	[texture replaceRegion:MTLRegionMake2D(0, 0, width, height)
	           mipmapLevel:0
	                 slice:slice
	             withBytes:bytes
	           bytesPerRow:bytesPerRow
	         bytesPerImage:0];
}

void sceneTextureGenerateMipmaps(void *handle, void *textureHandle) {
	@autoreleasepool {
		SceneDevice *scene = (__bridge SceneDevice *)handle;
		id<MTLCommandBuffer> commands = [scene.queue commandBuffer];
		id<MTLBlitCommandEncoder> blit = [commands blitCommandEncoder];
		[blit generateMipmapsForTexture:(__bridge id<MTLTexture>)textureHandle];
		[blit endEncoding];
		[commands commit];
		[commands waitUntilCompleted];
	}
}

// matchDepth makes the frame's depth buffer the frame's size. It is memoryless where the GPU can
// keep it in tile memory: nothing reads it after the frame, so it then takes no memory at all.
static void matchDepth(SceneDevice *scene, int width, int height) {
	if (scene.depth != nil && (int)scene.depth.width == width && (int)scene.depth.height == height) {
		return;
	}
	MTLTextureDescriptor *descriptor = [MTLTextureDescriptor texture2DDescriptorWithPixelFormat:MTLPixelFormatDepth32Float
	                                                                                      width:width
	                                                                                     height:height
	                                                                                  mipmapped:NO];
	descriptor.usage = MTLTextureUsageRenderTarget;
	descriptor.storageMode = [scene.device supportsFamily:MTLGPUFamilyApple1] ? MTLStorageModeMemoryless : MTLStorageModePrivate;
	scene.depth = [scene.device newTextureWithDescriptor:descriptor];
}

void *sceneFrameBegin(void *handle, int width, int height, int offscreen) {
	@autoreleasepool {
		SceneDevice *scene = (__bridge SceneDevice *)handle;
		dispatch_semaphore_wait(scene.previousFrame, DISPATCH_TIME_FOREVER);
		matchDepth(scene, width, height);
		SceneFrame *frame = [SceneFrame new];
		frame.owner = scene;
		if (offscreen) {
			MTLTextureDescriptor *descriptor = [MTLTextureDescriptor texture2DDescriptorWithPixelFormat:MTLPixelFormatBGRA8Unorm
			                                                                                      width:width
			                                                                                     height:height
			                                                                                  mipmapped:NO];
			descriptor.usage = MTLTextureUsageRenderTarget;
			descriptor.storageMode = MTLStorageModeManaged;
			frame.target = [scene.device newTextureWithDescriptor:descriptor];
		} else {
			CAMetalLayer *layer = scene.layer;
			CGFloat scale = scene.window.backingScaleFactor;
			if (layer.contentsScale != scale) {
				layer.contentsScale = scale;
			}
			CGSize size = CGSizeMake(width, height);
			if (!CGSizeEqualToSize(layer.drawableSize, size)) {
				layer.drawableSize = size;
			}
			frame.resizing = scene.window.inLiveResize;
			layer.presentsWithTransaction = frame.resizing;
			frame.drawable = [layer nextDrawable];
			if (frame.drawable == nil) {
				dispatch_semaphore_signal(scene.previousFrame);
				return NULL;
			}
			frame.target = frame.drawable.texture;
		}
		frame.commands = [scene.queue commandBuffer];
		dispatch_semaphore_t previousFrame = scene.previousFrame;
		[frame.commands addCompletedHandler:^(id<MTLCommandBuffer> finished) {
			dispatch_semaphore_signal(previousFrame);
		}];
		return retained(frame);
	}
}

void *scenePassBeginDepth(void *frameHandle, void *depthHandle) {
	@autoreleasepool {
		SceneFrame *frame = (__bridge SceneFrame *)frameHandle;
		MTLRenderPassDescriptor *pass = [MTLRenderPassDescriptor renderPassDescriptor];
		pass.depthAttachment.texture = (__bridge id<MTLTexture>)depthHandle;
		pass.depthAttachment.loadAction = MTLLoadActionClear;
		pass.depthAttachment.storeAction = MTLStoreActionStore;
		pass.depthAttachment.clearDepth = 1;
		id<MTLRenderCommandEncoder> encoder = [frame.commands renderCommandEncoderWithDescriptor:pass];
		[encoder setFrontFacingWinding:MTLWindingCounterClockwise];
		[encoder setCullMode:MTLCullModeNone];
		return retained(encoder);
	}
}

void *scenePassBeginMain(void *frameHandle, const float clear[4]) {
	@autoreleasepool {
		SceneFrame *frame = (__bridge SceneFrame *)frameHandle;
		MTLRenderPassDescriptor *pass = [MTLRenderPassDescriptor renderPassDescriptor];
		pass.colorAttachments[0].texture = frame.target;
		pass.colorAttachments[0].loadAction = MTLLoadActionClear;
		pass.colorAttachments[0].storeAction = MTLStoreActionStore;
		pass.colorAttachments[0].clearColor = MTLClearColorMake(clear[0], clear[1], clear[2], clear[3]);
		pass.depthAttachment.texture = frame.owner.depth;
		pass.depthAttachment.loadAction = MTLLoadActionClear;
		pass.depthAttachment.storeAction = MTLStoreActionDontCare;
		pass.depthAttachment.clearDepth = 1;
		id<MTLRenderCommandEncoder> encoder = [frame.commands renderCommandEncoderWithDescriptor:pass];
		[encoder setFrontFacingWinding:MTLWindingCounterClockwise];
		[encoder setCullMode:MTLCullModeNone];
		return retained(encoder);
	}
}

void sceneEncoderSetPipeline(void *encoder, void *pipeline) {
	[(__bridge id<MTLRenderCommandEncoder>)encoder setRenderPipelineState:(__bridge id<MTLRenderPipelineState>)pipeline];
}

void sceneEncoderSetDepthState(void *encoder, void *state) {
	[(__bridge id<MTLRenderCommandEncoder>)encoder setDepthStencilState:(__bridge id<MTLDepthStencilState>)state];
}

void sceneEncoderSetDepthBias(void *encoder, float bias, float slope) {
	[(__bridge id<MTLRenderCommandEncoder>)encoder setDepthBias:bias slopeScale:slope clamp:0];
}

void sceneEncoderSetVertexBuffer(void *encoder, void *buffer, int index) {
	[(__bridge id<MTLRenderCommandEncoder>)encoder setVertexBuffer:(__bridge id<MTLBuffer>)buffer offset:0 atIndex:index];
}

void sceneEncoderSetVertexBytes(void *encoder, const void *bytes, long length, int index) {
	[(__bridge id<MTLRenderCommandEncoder>)encoder setVertexBytes:bytes length:length atIndex:index];
}

void sceneEncoderSetFragmentBytes(void *encoder, const void *bytes, long length, int index) {
	[(__bridge id<MTLRenderCommandEncoder>)encoder setFragmentBytes:bytes length:length atIndex:index];
}

void sceneEncoderSetVertexTexture(void *encoder, void *texture, int index) {
	[(__bridge id<MTLRenderCommandEncoder>)encoder setVertexTexture:(__bridge id<MTLTexture>)texture atIndex:index];
}

void sceneEncoderSetFragmentTexture(void *encoder, void *texture, int index) {
	[(__bridge id<MTLRenderCommandEncoder>)encoder setFragmentTexture:(__bridge id<MTLTexture>)texture atIndex:index];
}

static MTLPrimitiveType primitiveType(int primitive) {
	switch (primitive) {
	case scenePrimitiveTriangleStrip:
		return MTLPrimitiveTypeTriangleStrip;
	case scenePrimitiveLines:
		return MTLPrimitiveTypeLine;
	case scenePrimitivePoints:
		return MTLPrimitiveTypePoint;
	default:
		return MTLPrimitiveTypeTriangle;
	}
}

void sceneEncoderDraw(void *encoder, int primitive, int start, int count, int instances) {
	[(__bridge id<MTLRenderCommandEncoder>)encoder drawPrimitives:primitiveType(primitive)
	                                                  vertexStart:start
	                                                  vertexCount:count
	                                                instanceCount:MAX(instances, 1)];
}

void sceneEncoderDrawIndexed(void *encoder, int count, void *indexBuffer, int instances) {
	[(__bridge id<MTLRenderCommandEncoder>)encoder drawIndexedPrimitives:MTLPrimitiveTypeTriangle
	                                                          indexCount:count
	                                                           indexType:MTLIndexTypeUInt32
	                                                         indexBuffer:(__bridge id<MTLBuffer>)indexBuffer
	                                                   indexBufferOffset:0
	                                                       instanceCount:MAX(instances, 1)];
}

void sceneEncoderEnd(void *encoder) {
	id<MTLRenderCommandEncoder> ended = (__bridge_transfer id<MTLRenderCommandEncoder>)encoder;
	[ended endEncoding];
}

void sceneFrameFinish(void *frameHandle, uint8_t *pixels, double *gpuSeconds) {
	@autoreleasepool {
		SceneFrame *frame = (__bridge_transfer SceneFrame *)frameHandle;
		if (pixels != NULL) {
			id<MTLBlitCommandEncoder> blit = [frame.commands blitCommandEncoder];
			[blit synchronizeResource:frame.target];
			[blit endEncoding];
		}
		if (frame.drawable != nil && !frame.resizing) {
			[frame.commands presentDrawable:frame.drawable];
		}
		[frame.commands commit];
		if (frame.drawable != nil && frame.resizing) {
			// shown in the same transaction as the window's new size, so it is never stretched
			[frame.commands waitUntilScheduled];
			[frame.drawable present];
		}
		if (pixels == NULL && gpuSeconds == NULL) {
			return;
		}
		[frame.commands waitUntilCompleted];
		if (gpuSeconds != NULL) {
			*gpuSeconds = frame.commands.GPUEndTime - frame.commands.GPUStartTime;
		}
		if (pixels != NULL) {
			NSUInteger width = frame.target.width, height = frame.target.height;
			[frame.target getBytes:pixels bytesPerRow:width * 4 fromRegion:MTLRegionMake2D(0, 0, width, height) mipmapLevel:0];
		}
	}
}
