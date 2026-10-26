//go:build android

// A frame: wait for the runtime's moment, begin, find the head and the eyes, take an image for
// each eye, and give them back with a projection layer.

#include "bridge.h"

// xr_wait_frame waits until the runtime wants the next frame (this paces the program).
// should_render is the runtime's: false while nothing of the program's is shown.
XrResult xr_wait_frame(xr_program *program, int *should_render, int64_t *display_time) {
	XrFrameWaitInfo wait = {XR_TYPE_FRAME_WAIT_INFO};
	XrFrameState state = {XR_TYPE_FRAME_STATE};
	XrResult result = program->call.xrWaitFrame(program->session, &wait, &state);
	*should_render = state.shouldRender;
	*display_time = state.predictedDisplayTime;
	return result;
}

XrResult xr_begin_frame(xr_program *program) {
	XrFrameBeginInfo begin = {XR_TYPE_FRAME_BEGIN_INFO};
	return program->call.xrBeginFrame(program->session, &begin);
}

// xr_locate_head finds the head at the frame's display time, in the world space. tracked is true
// only when both where it is and which way it looks are known: a position the runtime merely
// remembers (tracking lost, the headset just put on) would put things in the wrong place. A head
// whose orientation is not known comes back as the identity pose, never as a zero rotation.
XrResult xr_locate_head(xr_program *program, int64_t time, int *tracked, XrPosef *head) {
	static const XrSpaceLocationFlags known = XR_SPACE_LOCATION_POSITION_VALID_BIT | XR_SPACE_LOCATION_ORIENTATION_VALID_BIT;
	XrSpaceLocation location = {XR_TYPE_SPACE_LOCATION};
	*tracked = 0;
	*head = (XrPosef){{0, 0, 0, 1}, {0, 0, 0}};
	XrResult result = program->call.xrLocateSpace(program->head, program->world, time, &location);
	if (XR_FAILED(result)) return result;
	*tracked = (location.locationFlags & known) == known;
	if ((location.locationFlags & XR_SPACE_LOCATION_ORIENTATION_VALID_BIT) != 0) *head = location.pose;
	return XR_SUCCESS;
}

// xr_locate_eyes finds the eyes at the frame's display time (kept in program->views for the layer),
// in the world space. located is false when the runtime does not know which way the eyes look:
// drawing then would show a wrong picture. A known orientation with a remembered position is still
// drawn, as runtimes expect: the picture turns with the head while the position catches up.
XrResult xr_locate_eyes(xr_program *program, int64_t time, int *located) {
	XrViewLocateInfo info = {XR_TYPE_VIEW_LOCATE_INFO};
	info.viewConfigurationType = XR_VIEW_CONFIGURATION_TYPE_PRIMARY_STEREO;
	info.displayTime = time;
	info.space = program->world;
	XrViewState state = {XR_TYPE_VIEW_STATE};
	for (int eye = 0; eye < XR_EYES; eye++) program->views[eye] = (XrView){XR_TYPE_VIEW};
	uint32_t count = 0;
	*located = 0;
	XrResult result = program->call.xrLocateViews(program->session, &info, &state, XR_EYES, &count, program->views);
	if (XR_FAILED(result)) return result;
	*located = count == XR_EYES && (state.viewStateFlags & XR_VIEW_STATE_ORIENTATION_VALID_BIT) != 0;
	return XR_SUCCESS;
}

// xr_acquire_images takes an image of each eye's swapchain and waits until it can be drawn,
// handing out its framebuffer.
XrResult xr_acquire_images(xr_program *program, GLuint *framebuffers) {
	for (int index = 0; index < XR_EYES; index++) {
		xr_eye *eye = &program->eyes[index];
		XrSwapchainImageAcquireInfo acquire = {XR_TYPE_SWAPCHAIN_IMAGE_ACQUIRE_INFO};
		uint32_t image = 0;
		XrResult result = program->call.xrAcquireSwapchainImage(eye->swapchain, &acquire, &image);
		if (XR_FAILED(result)) return result;
		eye->acquired = 1;
		XrSwapchainImageWaitInfo wait = {XR_TYPE_SWAPCHAIN_IMAGE_WAIT_INFO};
		wait.timeout = XR_INFINITE_DURATION;
		result = program->call.xrWaitSwapchainImage(eye->swapchain, &wait);
		if (XR_FAILED(result)) return result;
		framebuffers[index] = eye->framebuffers[image];
	}
	return XR_SUCCESS;
}

// xr_acquire_panel takes an image of a panel's swapchain and waits until it can be drawn, handing
// out its framebuffer. A panel already acquired this frame hands out the same image again.
XrResult xr_acquire_panel(xr_program *program, int index, GLuint *framebuffer) {
	if (index < 0 || index >= program->panel_count) return XR_ERROR_HANDLE_INVALID;
	xr_panel *panel = &program->panels[index];
	if (panel->acquired) {
		*framebuffer = panel->framebuffers[panel->image];
		return XR_SUCCESS;
	}
	XrSwapchainImageAcquireInfo acquire = {XR_TYPE_SWAPCHAIN_IMAGE_ACQUIRE_INFO};
	XrResult result = program->call.xrAcquireSwapchainImage(panel->swapchain, &acquire, &panel->image);
	if (XR_FAILED(result)) return result;
	panel->acquired = 1;
	XrSwapchainImageWaitInfo wait = {XR_TYPE_SWAPCHAIN_IMAGE_WAIT_INFO};
	wait.timeout = XR_INFINITE_DURATION;
	result = program->call.xrWaitSwapchainImage(panel->swapchain, &wait);
	if (XR_FAILED(result)) return result;
	*framebuffer = panel->framebuffers[panel->image];
	return XR_SUCCESS;
}

static void release_image(xr_program *program, XrSwapchain swapchain, int *acquired) {
	if (!*acquired) return;
	XrSwapchainImageReleaseInfo release = {XR_TYPE_SWAPCHAIN_IMAGE_RELEASE_INFO};
	program->call.xrReleaseSwapchainImage(swapchain, &release);
	*acquired = 0;
}

// release_images gives back every image held. A panel's released image is what the runtime shows
// of it from then on, until another is released: a panel not drawn again keeps its last picture.
static void release_images(xr_program *program) {
	for (int index = 0; index < XR_EYES; index++) {
		release_image(program, program->eyes[index].swapchain, &program->eyes[index].acquired);
	}
	for (int index = 0; index < program->panel_count; index++) {
		xr_panel *panel = &program->panels[index];
		if (panel->acquired) panel->ready = 1;
		release_image(program, panel->swapchain, &panel->acquired);
	}
}

// quad_layer is a panel shown as a quad layer in the world space, blended by its alpha.
static XrCompositionLayerQuad quad_layer(xr_program *program, const xr_quad *quad) {
	xr_panel *panel = &program->panels[quad->panel];
	XrCompositionLayerQuad layer = {XR_TYPE_COMPOSITION_LAYER_QUAD};
	layer.layerFlags = XR_COMPOSITION_LAYER_BLEND_TEXTURE_SOURCE_ALPHA_BIT;
	layer.space = program->world;
	layer.eyeVisibility = XR_EYE_VISIBILITY_BOTH;
	layer.subImage.swapchain = panel->swapchain;
	layer.subImage.imageRect.extent.width = panel->width;
	layer.subImage.imageRect.extent.height = panel->height;
	layer.pose = quad->pose;
	layer.size.width = quad->width;
	layer.size.height = quad->height;
	return layer;
}

// xr_end_frame gives the images back and ends the frame: with the eyes as a projection layer when
// they were drawn, and over it each quad whose panel has a picture; with no layer (the runtime
// shows nothing of the program's) when the eyes were not drawn. While the room is shown, its
// passthrough layer goes under everything, and the eyes' layer is blended over it by its alpha.
XrResult xr_end_frame(xr_program *program, int64_t time, int render, const xr_quad *quads, int quad_count) {
	int drawn = render && program->eyes[0].acquired && program->eyes[1].acquired;
	release_images(program);
	XrCompositionLayerProjectionView views[XR_EYES];
	for (int index = 0; index < XR_EYES; index++) {
		views[index] = (XrCompositionLayerProjectionView){XR_TYPE_COMPOSITION_LAYER_PROJECTION_VIEW};
		views[index].pose = program->views[index].pose;
		views[index].fov = program->views[index].fov;
		views[index].subImage.swapchain = program->eyes[index].swapchain;
		views[index].subImage.imageRect.extent.width = program->eyes[index].width;
		views[index].subImage.imageRect.extent.height = program->eyes[index].height;
	}
	XrCompositionLayerProjection layer = {XR_TYPE_COMPOSITION_LAYER_PROJECTION};
	layer.space = program->world;
	layer.viewCount = XR_EYES;
	layer.views = views;
	const XrCompositionLayerBaseHeader *layers[2 + XR_MOST_PANELS];
	uint32_t count = 0;
	XrCompositionLayerPassthroughFB room = {XR_TYPE_COMPOSITION_LAYER_PASSTHROUGH_FB};
	if (program->passthrough_running) {
		room.layerHandle = program->passthrough_layer;
		room.flags = XR_COMPOSITION_LAYER_BLEND_TEXTURE_SOURCE_ALPHA_BIT;
		layers[count++] = (const XrCompositionLayerBaseHeader *)&room;
		layer.layerFlags = XR_COMPOSITION_LAYER_BLEND_TEXTURE_SOURCE_ALPHA_BIT;
	}
	uint32_t room_layers = count; // shown even when the eyes are not drawn
	layers[count++] = (const XrCompositionLayerBaseHeader *)&layer;
	XrCompositionLayerQuad shown[XR_MOST_PANELS];
	int quads_shown = 0;
	for (int index = 0; index < quad_count && index < XR_MOST_PANELS; index++) {
		int panel = quads[index].panel;
		if (panel < 0 || panel >= program->panel_count || !program->panels[panel].ready) continue;
		shown[quads_shown] = quad_layer(program, &quads[index]);
		layers[count++] = (const XrCompositionLayerBaseHeader *)&shown[quads_shown];
		quads_shown++;
	}
	XrFrameEndInfo end = {XR_TYPE_FRAME_END_INFO};
	end.displayTime = time;
	end.environmentBlendMode = XR_ENVIRONMENT_BLEND_MODE_OPAQUE;
	end.layerCount = drawn ? count : room_layers;
	end.layers = end.layerCount > 0 ? layers : NULL;
	return program->call.xrEndFrame(program->session, &end);
}
