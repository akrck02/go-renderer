// The flat renderer through Metal: see flat_darwin.h for what each function does, and
// docs/metal.md for the design.

#import <Cocoa/Cocoa.h>
#import <Metal/Metal.h>
#import <QuartzCore/CAMetalLayer.h>
#include <stdlib.h>
#include <string.h>
#include "flat_darwin.h"

// framesInFlight is how many frames the processor may prepare while the GPU draws earlier ones:
// each has its own instance buffer, so none is written while the GPU reads it.
#define framesInFlight 3

// The shaders, compiled when the renderer starts, so building needs no Xcode. They match the
// OpenGL flat path's (graphics.OverlayVertexShader and OverlayFragmentShader): a quad in screen
// pixels from the top-left corner, painted in its colour where the atlas covers; a backdrop's quad
// (picture place -2) shows the blurred snapshot instead. cover and blurPass are the backdrop's
// blur (graphics.BackdropBlurShader): one triangle over the target, and one pass of a separable
// Gaussian, which with weights 1, 0, 0, 0, 0 halves the picture instead.
static NSString *const shaderSource = @
	"#include <metal_stdlib>\n"
	"using namespace metal;\n"
	"struct Instance { packed_float4 screen; packed_float4 atlas; packed_float4 picture; uchar4 color; };\n"
	"struct Sizes { float2 screen; float2 atlas; float2 pictures; float2 unused; };\n"
	"struct Fragment { float4 position [[position]]; float2 atlasCoordinate; float2 pictureCoordinate; float4 color; float backdrop; };\n"
	"vertex Fragment placeQuad(uint corner [[vertex_id]], uint index [[instance_id]],\n"
	"                          const device Instance *instances [[buffer(0)]], constant Sizes &sizes [[buffer(1)]]) {\n"
	"    Instance quad = instances[index];\n"
	"    float2 along = float2(corner & 1, corner >> 1);\n"
	"    float2 position = mix(float2(quad.screen[0], quad.screen[1]), float2(quad.screen[2], quad.screen[3]), along);\n"
	"    float2 atlas = mix(float2(quad.atlas[0], quad.atlas[1]), float2(quad.atlas[2], quad.atlas[3]), along);\n"
	"    float2 normalized = position / sizes.screen;\n"
	"    Fragment out;\n"
	"    out.position = float4(normalized.x * 2.0 - 1.0, 1.0 - normalized.y * 2.0, 0.0, 1.0);\n"
	"    out.atlasCoordinate = atlas / sizes.atlas;\n"
	"    float2 picture = mix(float2(quad.picture[0], quad.picture[1]), float2(quad.picture[2], quad.picture[3]), along);\n"
	"    out.pictureCoordinate = quad.picture[0] < 0.0 ? float2(-1.0) : picture / sizes.pictures;\n"
	"    out.color = float4(quad.color) / 255.0;\n"
	"    out.backdrop = quad.picture[0] < -1.5 ? 1.0 : 0.0;\n"
	"    return out;\n"
	"}\n"
	"fragment float4 paintQuad(Fragment in [[stage_in]], texture2d<float> atlas [[texture(0)]],\n"
	"                          texture2d<float> pictures [[texture(1)]], texture2d<float> blurred [[texture(2)]],\n"
	"                          sampler smooth [[sampler(0)]], constant Sizes &sizes [[buffer(0)]]) {\n"
	"    float coverage = atlas.sample(smooth, in.atlasCoordinate).r;\n"
	"    float4 painted = in.color;\n"
	"    if (in.pictureCoordinate.x >= 0.0) {\n"
	"        painted *= pictures.sample(smooth, in.pictureCoordinate);\n"
	"    }\n"
	"    if (in.backdrop > 0.5) {\n"
	"        painted = float4(blurred.sample(smooth, in.position.xy / sizes.screen).rgb, 1.0);\n"
	"    }\n"
	"    return float4(painted.rgb, painted.a * coverage);\n"
	"}\n"
	"struct Covered { float4 position [[position]]; float2 coordinate; };\n"
	"vertex Covered cover(uint corner [[vertex_id]]) {\n"
	"    float2 along = float2(float((corner << 1) & 2), float(corner & 2));\n"
	"    Covered out;\n"
	"    out.position = float4(along * 2.0 - 1.0, 0.0, 1.0);\n"
	"    out.coordinate = float2(along.x, 1.0 - along.y);\n"
	"    return out;\n"
	"}\n"
	"struct BlurPass { float2 step; float2 unused; float4 near; float far; };\n"
	"fragment float4 blurPass(Covered in [[stage_in]], texture2d<float> source [[texture(0)]],\n"
	"                         sampler smooth [[sampler(0)]], constant BlurPass &pass [[buffer(0)]]) {\n"
	"    float weights[5] = {pass.near.x, pass.near.y, pass.near.z, pass.near.w, pass.far};\n"
	"    float4 sum = source.sample(smooth, in.coordinate) * weights[0];\n"
	"    for (int tap = 1; tap <= 4; tap++) {\n"
	"        float2 offset = pass.step * float(tap);\n"
	"        sum += source.sample(smooth, in.coordinate + offset) * weights[tap];\n"
	"        sum += source.sample(smooth, in.coordinate - offset) * weights[tap];\n"
	"    }\n"
	"    return sum;\n"
	"}\n";

// blurPassBytes is BlurPass in the shaders: the step between taps, then the weights. 48 bytes, the
// float4 aligned to 16 as Metal lays it out.
typedef struct {
	float step[2];
	float unused[2];
	float near[4];
	float far;
	float padding[3];
} blurPassBytes;

@interface FlatRenderer : NSObject
@property(nonatomic, strong) NSWindow *window;
@property(nonatomic, strong) CAMetalLayer *layer;
@property(nonatomic, strong) id<MTLDevice> device;
@property(nonatomic, strong) id<MTLCommandQueue> queue;
@property(nonatomic, strong) id<MTLRenderPipelineState> pipeline;
@property(nonatomic, strong) id<MTLRenderPipelineState> blurPipeline;
// the backdrop's textures, made once for a target size and kept: the halvings, each half the one
// before, the spare the first blur pass writes into, and a full-size copy for a blur not halved
@property(nonatomic, strong) NSMutableArray<id<MTLTexture>> *levels;
@property(nonatomic, strong) id<MTLTexture> spare;
@property(nonatomic, strong) id<MTLTexture> unhalved;
@property(nonatomic) NSUInteger levelsWidth;
@property(nonatomic) NSUInteger levelsHeight;
@property(nonatomic, strong) id<MTLSamplerState> sampler;
@property(nonatomic, strong) id<MTLTexture> atlas;
@property(nonatomic, strong) id<MTLTexture> pictures;
@property(nonatomic, strong) NSMutableArray<id<MTLBuffer>> *instanceBuffers;
@property(nonatomic, strong) dispatch_semaphore_t inFlight;
@property(nonatomic) NSUInteger nextBuffer;
@end

@implementation FlatRenderer
@end

static void setFailure(char **failure, NSString *message) {
	if (failure != NULL) {
		*failure = strdup(message.UTF8String);
	}
}

// chooseDevice prefers a low-power GPU when a Mac has two: drawing an interface is light work.
static id<MTLDevice> chooseDevice(void) {
	NSArray<id<MTLDevice>> *devices = MTLCopyAllDevices();
	for (id<MTLDevice> device in devices) {
		if (device.isLowPower) {
			return device;
		}
	}
	return MTLCreateSystemDefaultDevice();
}

static id<MTLLibrary> makeLibrary(id<MTLDevice> device, char **failure) {
	NSError *error = nil;
	id<MTLLibrary> library = [device newLibraryWithSource:shaderSource options:nil error:&error];
	if (library == nil) {
		setFailure(failure, [NSString stringWithFormat:@"metal: compiling the shaders: %@", error.localizedDescription]);
	}
	return library;
}

// makeBlurPipeline draws one pass of the backdrop's blur into a texture: no blending, the pass
// writes every texel.
static id<MTLRenderPipelineState> makeBlurPipeline(id<MTLDevice> device, id<MTLLibrary> library, char **failure) {
	MTLRenderPipelineDescriptor *descriptor = [MTLRenderPipelineDescriptor new];
	descriptor.vertexFunction = [library newFunctionWithName:@"cover"];
	descriptor.fragmentFunction = [library newFunctionWithName:@"blurPass"];
	descriptor.colorAttachments[0].pixelFormat = MTLPixelFormatBGRA8Unorm;
	NSError *error = nil;
	id<MTLRenderPipelineState> pipeline = [device newRenderPipelineStateWithDescriptor:descriptor error:&error];
	if (pipeline == nil) {
		setFailure(failure, [NSString stringWithFormat:@"metal: making the blur's pipeline: %@", error.localizedDescription]);
	}
	return pipeline;
}

static id<MTLRenderPipelineState> makePipeline(id<MTLDevice> device, id<MTLLibrary> library, char **failure) {
	NSError *error = nil;
	MTLRenderPipelineDescriptor *descriptor = [MTLRenderPipelineDescriptor new];
	descriptor.vertexFunction = [library newFunctionWithName:@"placeQuad"];
	descriptor.fragmentFunction = [library newFunctionWithName:@"paintQuad"];
	MTLRenderPipelineColorAttachmentDescriptor *colour = descriptor.colorAttachments[0];
	colour.pixelFormat = MTLPixelFormatBGRA8Unorm;
	// the same blending as the OpenGL path's BlendFuncSeparate(SRC_ALPHA, ONE_MINUS_SRC_ALPHA, ONE, ONE_MINUS_SRC_ALPHA)
	colour.blendingEnabled = YES;
	colour.sourceRGBBlendFactor = MTLBlendFactorSourceAlpha;
	colour.destinationRGBBlendFactor = MTLBlendFactorOneMinusSourceAlpha;
	colour.sourceAlphaBlendFactor = MTLBlendFactorOne;
	colour.destinationAlphaBlendFactor = MTLBlendFactorOneMinusSourceAlpha;
	id<MTLRenderPipelineState> pipeline = [device newRenderPipelineStateWithDescriptor:descriptor error:&error];
	if (pipeline == nil) {
		setFailure(failure, [NSString stringWithFormat:@"metal: making the pipeline: %@", error.localizedDescription]);
	}
	return pipeline;
}

void *flatRendererCreate(void *cocoaWindow, int vsync, char **failure) {
	@autoreleasepool {
		id<MTLDevice> device = chooseDevice();
		if (device == nil) {
			setFailure(failure, @"metal: this computer has no Metal device");
			return NULL;
		}
		id<MTLLibrary> library = makeLibrary(device, failure);
		if (library == nil) {
			return NULL;
		}
		id<MTLRenderPipelineState> pipeline = makePipeline(device, library, failure);
		if (pipeline == nil) {
			return NULL;
		}
		id<MTLRenderPipelineState> blurPipeline = makeBlurPipeline(device, library, failure);
		if (blurPipeline == nil) {
			return NULL;
		}
		FlatRenderer *renderer = [FlatRenderer new];
		renderer.window = (__bridge NSWindow *)cocoaWindow;
		renderer.device = device;
		renderer.queue = [device newCommandQueue];
		renderer.pipeline = pipeline;
		renderer.blurPipeline = blurPipeline;
		renderer.levels = [NSMutableArray array];
		renderer.instanceBuffers = [NSMutableArray arrayWithCapacity:framesInFlight];
		renderer.inFlight = dispatch_semaphore_create(framesInFlight);

		MTLSamplerDescriptor *sampling = [MTLSamplerDescriptor new];
		sampling.minFilter = MTLSamplerMinMagFilterLinear;
		sampling.magFilter = MTLSamplerMinMagFilterLinear;
		sampling.sAddressMode = MTLSamplerAddressModeClampToEdge;
		sampling.tAddressMode = MTLSamplerAddressModeClampToEdge;
		renderer.sampler = [device newSamplerStateWithDescriptor:sampling];

		CAMetalLayer *layer = [CAMetalLayer layer];
		layer.device = device;
		layer.pixelFormat = MTLPixelFormatBGRA8Unorm;
		// the backdrop's blur reads what was drawn: drawables that are only render targets cannot be
		// read. It costs Metal a little freedom in how it keeps them, on every frame
		layer.framebufferOnly = NO;
		layer.opaque = YES;
		layer.displaySyncEnabled = vsync != 0;
		// two drawables instead of three: an interface that draws on demand never has more than
		// one frame queued, and each drawable is a full window of pixels
		layer.maximumDrawableCount = 2;
		layer.contentsScale = renderer.window.backingScaleFactor;
		NSView *view = renderer.window.contentView;
		view.layer = layer; // set before wantsLayer: the view hosts this layer instead of making its own
		view.wantsLayer = YES;
		renderer.layer = layer;
		return (__bridge_retained void *)renderer;
	}
}

void flatRendererDestroy(void *renderer) {
	FlatRenderer *released = (__bridge_transfer FlatRenderer *)renderer;
	released = nil;
}

void flatRendererUploadAtlas(void *handle, const uint8_t *pixels, int size) {
	FlatRenderer *renderer = (__bridge FlatRenderer *)handle;
	// a new texture each time, so a frame still being drawn keeps reading the old one; the atlas
	// changes only when glyphs or corners are added
	MTLTextureDescriptor *descriptor = [MTLTextureDescriptor texture2DDescriptorWithPixelFormat:MTLPixelFormatR8Unorm
	                                                                                      width:size
	                                                                                     height:size
	                                                                                  mipmapped:NO];
	descriptor.usage = MTLTextureUsageShaderRead;
	id<MTLTexture> atlas = [renderer.device newTextureWithDescriptor:descriptor];
	[atlas replaceRegion:MTLRegionMake2D(0, 0, size, size) mipmapLevel:0 withBytes:pixels bytesPerRow:size];
	renderer.atlas = atlas;
}

void flatRendererUploadPictures(void *handle, const uint8_t *pixels, int size, int whole, int x, int y, int width,
                                int height) {
	FlatRenderer *renderer = (__bridge FlatRenderer *)handle;
	NSUInteger bytesPerRow = (NSUInteger)size * 4;
	if (whole || renderer.pictures == nil || renderer.pictures.width != (NSUInteger)size) {
		// a new texture, so a frame still being drawn keeps reading the old one
		MTLTextureDescriptor *descriptor = [MTLTextureDescriptor texture2DDescriptorWithPixelFormat:MTLPixelFormatRGBA8Unorm
		                                                                                      width:size
		                                                                                     height:size
		                                                                                  mipmapped:NO];
		descriptor.usage = MTLTextureUsageShaderRead;
		id<MTLTexture> pictures = [renderer.device newTextureWithDescriptor:descriptor];
		[pictures replaceRegion:MTLRegionMake2D(0, 0, size, size) mipmapLevel:0 withBytes:pixels bytesPerRow:bytesPerRow];
		renderer.pictures = pictures;
		return;
	}
	// an area of the current texture: a video's next frame, a picture added
	const uint8_t *start = pixels + (NSUInteger)y * bytesPerRow + (NSUInteger)x * 4;
	[renderer.pictures replaceRegion:MTLRegionMake2D(x, y, width, height) mipmapLevel:0 withBytes:start bytesPerRow:bytesPerRow];
}

// instanceBuffer copies the instances into the buffer of this frame, growing it when they do not fit.
static id<MTLBuffer> instanceBuffer(FlatRenderer *renderer, const flatInstance *instances, int count) {
	NSUInteger index = renderer.nextBuffer;
	renderer.nextBuffer = (index + 1) % framesInFlight;
	NSUInteger needed = MAX((NSUInteger)count, 1) * sizeof(flatInstance);
	id<MTLBuffer> buffer = index < renderer.instanceBuffers.count ? renderer.instanceBuffers[index] : nil;
	if (buffer == nil || buffer.length < needed) {
		NSUInteger length = buffer == nil ? 4096 : buffer.length;
		while (length < needed) {
			length *= 2;
		}
		buffer = [renderer.device newBufferWithLength:length
		                                      options:MTLResourceStorageModeShared | MTLResourceCPUCacheModeWriteCombined];
		if (index < renderer.instanceBuffers.count) {
			renderer.instanceBuffers[index] = buffer;
		} else {
			[renderer.instanceBuffers addObject:buffer];
		}
	}
	if (count > 0) {
		memcpy(buffer.contents, instances, count * sizeof(flatInstance));
	}
	return buffer;
}

// newBlurTexture makes a texture the blur draws into and reads from, on the GPU only.
static id<MTLTexture> newBlurTexture(FlatRenderer *renderer, NSUInteger width, NSUInteger height) {
	MTLTextureDescriptor *descriptor = [MTLTextureDescriptor texture2DDescriptorWithPixelFormat:MTLPixelFormatBGRA8Unorm
	                                                                                      width:MAX(width, 1)
	                                                                                     height:MAX(height, 1)
	                                                                                  mipmapped:NO];
	descriptor.usage = MTLTextureUsageRenderTarget | MTLTextureUsageShaderRead;
	descriptor.storageMode = MTLStorageModePrivate;
	return [renderer.device newTextureWithDescriptor:descriptor];
}

// prepareBlur makes the halvings for a target of this size, when it changed, and enough of them.
static void prepareBlur(FlatRenderer *renderer, NSUInteger width, NSUInteger height, int halvings) {
	if (width != renderer.levelsWidth || height != renderer.levelsHeight) {
		[renderer.levels removeAllObjects];
		renderer.spare = nil;
		renderer.unhalved = nil;
		renderer.levelsWidth = width;
		renderer.levelsHeight = height;
	}
	while ((int)renderer.levels.count < halvings) {
		NSUInteger beforeWidth = width, beforeHeight = height;
		if (renderer.levels.count > 0) {
			beforeWidth = renderer.levels.lastObject.width;
			beforeHeight = renderer.levels.lastObject.height;
		}
		[renderer.levels addObject:newBlurTexture(renderer, beforeWidth / 2, beforeHeight / 2)];
	}
}

// blurPass draws a texture into another through the blur's shader, with these weights along step.
static void blurPass(FlatRenderer *renderer, id<MTLCommandBuffer> commands, id<MTLTexture> from, id<MTLTexture> into,
                     const float weights[5], float stepX, float stepY) {
	MTLRenderPassDescriptor *pass = [MTLRenderPassDescriptor renderPassDescriptor];
	pass.colorAttachments[0].texture = into;
	pass.colorAttachments[0].loadAction = MTLLoadActionDontCare; // every texel is written
	pass.colorAttachments[0].storeAction = MTLStoreActionStore;
	blurPassBytes bytes = {{stepX, stepY}, {0, 0}, {weights[0], weights[1], weights[2], weights[3]}, weights[4], {0, 0, 0}};
	id<MTLRenderCommandEncoder> encoder = [commands renderCommandEncoderWithDescriptor:pass];
	[encoder setRenderPipelineState:renderer.blurPipeline];
	[encoder setFragmentTexture:from atIndex:0];
	[encoder setFragmentSamplerState:renderer.sampler atIndex:0];
	[encoder setFragmentBytes:&bytes length:sizeof(bytes) atIndex:0];
	[encoder drawPrimitives:MTLPrimitiveTypeTriangle vertexStart:0 vertexCount:3];
	[encoder endEncoding];
}

// blurSnapshot halves what was drawn into target and blurs the smallest copy, in passes of their
// own: the frame's pass must have ended. It returns the blurred texture.
static id<MTLTexture> blurSnapshot(FlatRenderer *renderer, id<MTLCommandBuffer> commands, id<MTLTexture> target,
                                   const flatSnapshot *snapshot) {
	static const float halving[5] = {1, 0, 0, 0, 0}; // read between four texels: their average
	int halvings = MAX(snapshot->halvings, 0);
	prepareBlur(renderer, target.width, target.height, halvings);
	id<MTLTexture> smallest = target;
	for (int level = 0; level < halvings; level++) {
		blurPass(renderer, commands, smallest, renderer.levels[level], halving, 0, 0);
		smallest = renderer.levels[level];
	}
	if (renderer.spare == nil || renderer.spare.width != smallest.width || renderer.spare.height != smallest.height) {
		renderer.spare = newBlurTexture(renderer, smallest.width, smallest.height);
	}
	id<MTLTexture> result = smallest;
	if (halvings == 0) {
		// a blur too small to halve for: the window's drawable cannot be written back into, so the
		// second pass writes a texture of its own, made the first time it is needed
		if (renderer.unhalved == nil) {
			renderer.unhalved = newBlurTexture(renderer, target.width, target.height);
		}
		result = renderer.unhalved;
	}
	blurPass(renderer, commands, smallest, renderer.spare, snapshot->weights, 1.0f / smallest.width, 0);
	blurPass(renderer, commands, renderer.spare, result, snapshot->weights, 0, 1.0f / smallest.height);
	return result;
}

// drawRange draws the instances from first up to (not including) last into target, in a pass that
// clears it to the background or keeps what earlier passes drew. blurred is the snapshot the
// backdrops among them read, or nil before the first.
static void drawRange(FlatRenderer *renderer, id<MTLCommandBuffer> commands, id<MTLTexture> target, id<MTLBuffer> buffer,
                      int first, int last, BOOL clear, id<MTLTexture> blurred, const float background[4], int width,
                      int height) {
	MTLRenderPassDescriptor *pass = [MTLRenderPassDescriptor renderPassDescriptor];
	pass.colorAttachments[0].texture = target;
	pass.colorAttachments[0].loadAction = clear ? MTLLoadActionClear : MTLLoadActionLoad;
	pass.colorAttachments[0].storeAction = MTLStoreActionStore;
	pass.colorAttachments[0].clearColor = MTLClearColorMake(background[0], background[1], background[2], 1);
	id<MTLRenderCommandEncoder> encoder = [commands renderCommandEncoderWithDescriptor:pass];
	if (last > first && renderer.atlas != nil) {
		id<MTLTexture> pictures = renderer.pictures != nil ? renderer.pictures : renderer.atlas; // read by no quad without any
		float sizes[8] = {(float)width, (float)height, (float)renderer.atlas.width, (float)renderer.atlas.height,
		                  (float)pictures.width, (float)pictures.height, 0, 0};
		[encoder setRenderPipelineState:renderer.pipeline];
		[encoder setVertexBuffer:buffer offset:(NSUInteger)first * sizeof(flatInstance) atIndex:0];
		[encoder setVertexBytes:sizes length:sizeof(sizes) atIndex:1];
		[encoder setFragmentBytes:sizes length:sizeof(sizes) atIndex:0];
		[encoder setFragmentTexture:renderer.atlas atIndex:0];
		[encoder setFragmentTexture:pictures atIndex:1];
		[encoder setFragmentTexture:(blurred != nil ? blurred : renderer.atlas) atIndex:2]; // read by no quad before a snapshot
		[encoder setFragmentSamplerState:renderer.sampler atIndex:0];
		[encoder drawPrimitives:MTLPrimitiveTypeTriangleStrip vertexStart:0 vertexCount:4 instanceCount:last - first];
	}
	[encoder endEncoding];
}

// encodeFrame draws the instances into target: in one pass when nothing is frosted, else cut at
// each snapshot, where the frame so far is blurred for the backdrops after it.
static void encodeFrame(FlatRenderer *renderer, id<MTLCommandBuffer> commands, id<MTLTexture> target,
                        id<MTLBuffer> buffer, int count, const flatSnapshot *snapshots, int snapshotCount,
                        const float background[4], int width, int height) {
	int drawn = 0;
	BOOL clear = YES;
	id<MTLTexture> blurred = nil;
	for (int index = 0; index < snapshotCount; index++) {
		int at = MIN(MAX(snapshots[index].at, drawn), count);
		drawRange(renderer, commands, target, buffer, drawn, at, clear, blurred, background, width, height);
		clear = NO;
		blurred = blurSnapshot(renderer, commands, target, &snapshots[index]);
		drawn = at;
	}
	drawRange(renderer, commands, target, buffer, drawn, count, clear, blurred, background, width, height);
}

int flatRendererDraw(void *handle, const flatInstance *instances, int count, const flatSnapshot *snapshots,
                     int snapshotCount, const float background[4], int width, int height) {
	FlatRenderer *renderer = (__bridge FlatRenderer *)handle;
	@autoreleasepool {
		CAMetalLayer *layer = renderer.layer;
		CGFloat scale = renderer.window.backingScaleFactor;
		if (layer.contentsScale != scale) {
			layer.contentsScale = scale; // the window moved to a screen of another density
		}
		CGSize size = CGSizeMake(width, height);
		if (!CGSizeEqualToSize(layer.drawableSize, size)) {
			layer.drawableSize = size;
		}
		// while the window is being resized, show the frame in the same transaction as the window's
		// new size, so the picture is never stretched to a size it was not drawn for
		BOOL resizing = renderer.window.inLiveResize;
		layer.presentsWithTransaction = resizing;

		dispatch_semaphore_wait(renderer.inFlight, DISPATCH_TIME_FOREVER);
		id<CAMetalDrawable> drawable = [layer nextDrawable];
		if (drawable == nil) {
			dispatch_semaphore_signal(renderer.inFlight);
			return 0;
		}
		id<MTLBuffer> buffer = instanceBuffer(renderer, instances, count);
		id<MTLCommandBuffer> commands = [renderer.queue commandBuffer];
		encodeFrame(renderer, commands, drawable.texture, buffer, count, snapshots, snapshotCount, background, width, height);
		dispatch_semaphore_t inFlight = renderer.inFlight;
		[commands addCompletedHandler:^(id<MTLCommandBuffer> finished) {
			dispatch_semaphore_signal(inFlight);
		}];
		if (resizing) {
			[commands commit];
			[commands waitUntilScheduled];
			[drawable present];
		} else {
			[commands presentDrawable:drawable];
			[commands commit];
		}
	}
	return 1;
}

void flatRendererCapture(void *handle, const flatInstance *instances, int count, const flatSnapshot *snapshots,
                         int snapshotCount, const float background[4], int width, int height, uint8_t *pixels) {
	FlatRenderer *renderer = (__bridge FlatRenderer *)handle;
	@autoreleasepool {
		MTLTextureDescriptor *descriptor = [MTLTextureDescriptor texture2DDescriptorWithPixelFormat:MTLPixelFormatBGRA8Unorm
		                                                                                      width:width
		                                                                                     height:height
		                                                                                  mipmapped:NO];
		descriptor.usage = MTLTextureUsageRenderTarget | MTLTextureUsageShaderRead; // read by the backdrop's blur
		descriptor.storageMode = MTLStorageModeManaged;
		id<MTLTexture> target = [renderer.device newTextureWithDescriptor:descriptor];

		dispatch_semaphore_wait(renderer.inFlight, DISPATCH_TIME_FOREVER);
		id<MTLBuffer> buffer = instanceBuffer(renderer, instances, count);
		id<MTLCommandBuffer> commands = [renderer.queue commandBuffer];
		encodeFrame(renderer, commands, target, buffer, count, snapshots, snapshotCount, background, width, height);
		id<MTLBlitCommandEncoder> blit = [commands blitCommandEncoder];
		[blit synchronizeResource:target];
		[blit endEncoding];
		[commands commit];
		[commands waitUntilCompleted];
		dispatch_semaphore_signal(renderer.inFlight);
		[target getBytes:pixels bytesPerRow:width * 4 fromRegion:MTLRegionMake2D(0, 0, width, height) mipmapLevel:0];
	}
}
