//go:build android

// The eyes' swapchains: the images the runtime shows, one swapchain an eye, and an OpenGL ES
// framebuffer for each image so a renderer draws into it as into any other target.

#include <stdlib.h>

#include "bridge.h"

// choose_format takes sRGB colour when the runtime has it (the compositor then reads the image as
// sRGB, and shaders write linear colour), else plain 8-bit RGBA.
static XrResult choose_format(xr_program *program) {
	uint32_t count = 0;
	XrResult result = program->call.xrEnumerateSwapchainFormats(program->session, 0, &count, NULL);
	if (XR_FAILED(result)) return result;
	int64_t *formats = calloc(count, sizeof(int64_t));
	result = program->call.xrEnumerateSwapchainFormats(program->session, count, &count, formats);
	program->format = 0;
	for (uint32_t index = 0; XR_SUCCEEDED(result) && index < count; index++) {
		if (formats[index] == GL_SRGB8_ALPHA8) program->format = GL_SRGB8_ALPHA8;
		if (formats[index] == GL_RGBA8 && program->format == 0) program->format = GL_RGBA8;
	}
	free(formats);
	if (XR_FAILED(result)) return result;
	if (program->format == 0) return XR_ERROR_SWAPCHAIN_FORMAT_UNSUPPORTED;
	program->srgb = program->format == GL_SRGB8_ALPHA8;
	return XR_SUCCESS;
}

// read_views asks the size the runtime recommends for each eye.
static XrResult read_views(xr_program *program, XrViewConfigurationView *views) {
	for (int eye = 0; eye < XR_EYES; eye++) views[eye] = (XrViewConfigurationView){XR_TYPE_VIEW_CONFIGURATION_VIEW};
	uint32_t count = 0;
	XrResult result = program->call.xrEnumerateViewConfigurationViews(program->instance, program->system,
		XR_VIEW_CONFIGURATION_TYPE_PRIMARY_STEREO, XR_EYES, &count, views);
	if (XR_SUCCEEDED(result) && count != XR_EYES) return XR_ERROR_VIEW_CONFIGURATION_TYPE_UNSUPPORTED;
	return result;
}

static GLuint make_depth(int32_t width, int32_t height) {
	GLuint depth = 0;
	glGenRenderbuffers(1, &depth);
	glBindRenderbuffer(GL_RENDERBUFFER, depth);
	glRenderbufferStorage(GL_RENDERBUFFER, GL_DEPTH_COMPONENT24, width, height);
	glBindRenderbuffer(GL_RENDERBUFFER, 0);
	return depth;
}

// make_framebuffers gives each image a framebuffer, all sharing the eye's depth buffer (only one
// image is drawn at a time). It returns 0 when OpenGL ES cannot draw into one.
static int make_framebuffers(xr_eye *eye, const XrSwapchainImageOpenGLESKHR *images) {
	eye->depth = make_depth(eye->width, eye->height);
	eye->framebuffers = calloc(eye->image_count, sizeof(GLuint));
	glGenFramebuffers(eye->image_count, eye->framebuffers);
	int complete = 1;
	for (uint32_t index = 0; index < eye->image_count; index++) {
		glBindFramebuffer(GL_FRAMEBUFFER, eye->framebuffers[index]);
		glFramebufferTexture2D(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_TEXTURE_2D, images[index].image, 0);
		glFramebufferRenderbuffer(GL_FRAMEBUFFER, GL_DEPTH_ATTACHMENT, GL_RENDERBUFFER, eye->depth);
		if (glCheckFramebufferStatus(GL_FRAMEBUFFER) != GL_FRAMEBUFFER_COMPLETE) complete = 0;
	}
	glBindFramebuffer(GL_FRAMEBUFFER, 0);
	return complete;
}

// read_images lists a swapchain's images (OpenGL ES textures); the caller frees the list.
static XrResult read_images(xr_program *program, XrSwapchain swapchain, uint32_t *count, XrSwapchainImageOpenGLESKHR **images) {
	XrResult result = program->call.xrEnumerateSwapchainImages(swapchain, 0, count, NULL);
	if (XR_FAILED(result)) return result;
	*images = calloc(*count, sizeof(XrSwapchainImageOpenGLESKHR));
	for (uint32_t index = 0; index < *count; index++) (*images)[index].type = XR_TYPE_SWAPCHAIN_IMAGE_OPENGL_ES_KHR;
	return program->call.xrEnumerateSwapchainImages(swapchain, *count, count, (XrSwapchainImageBaseHeader *)*images);
}

// create_swapchain makes a swapchain of the eyes' format, width × height pixels.
static XrResult create_swapchain(xr_program *program, int32_t width, int32_t height, XrSwapchain *swapchain) {
	XrSwapchainCreateInfo info = {XR_TYPE_SWAPCHAIN_CREATE_INFO};
	info.usageFlags = XR_SWAPCHAIN_USAGE_COLOR_ATTACHMENT_BIT | XR_SWAPCHAIN_USAGE_SAMPLED_BIT;
	info.format = program->format;
	info.sampleCount = 1;
	info.width = width;
	info.height = height;
	info.faceCount = 1;
	info.arraySize = 1;
	info.mipCount = 1;
	return program->call.xrCreateSwapchain(program->session, &info, swapchain);
}

static XrResult create_eye(xr_program *program, xr_eye *eye, const XrViewConfigurationView *view) {
	eye->width = view->recommendedImageRectWidth;
	eye->height = view->recommendedImageRectHeight;
	XrResult result = create_swapchain(program, eye->width, eye->height, &eye->swapchain);
	if (XR_FAILED(result)) return result;
	XrSwapchainImageOpenGLESKHR *images = NULL;
	result = read_images(program, eye->swapchain, &eye->image_count, &images);
	if (XR_SUCCEEDED(result) && !make_framebuffers(eye, images)) {
		program->framebuffer_incomplete = 1;
		result = XR_ERROR_VALIDATION_FAILURE;
	}
	free(images);
	return result;
}

// xr_create_swapchains makes both eyes' swapchains at the size the runtime recommends. A
// framebuffer OpenGL ES cannot complete sets framebuffer_incomplete.
XrResult xr_create_swapchains(xr_program *program) {
	XrResult result = choose_format(program);
	if (XR_FAILED(result)) return result;
	XrViewConfigurationView views[XR_EYES];
	result = read_views(program, views);
	if (XR_FAILED(result)) return result;
	for (int eye = 0; eye < XR_EYES; eye++) {
		result = create_eye(program, &program->eyes[eye], &views[eye]);
		if (XR_FAILED(result)) return result;
	}
	return XR_SUCCESS;
}

// make_panel_framebuffers gives each of a panel's images a framebuffer, with no depth buffer. It
// returns 0 when OpenGL ES cannot draw into one.
static int make_panel_framebuffers(xr_panel *panel, const XrSwapchainImageOpenGLESKHR *images) {
	panel->framebuffers = calloc(panel->image_count, sizeof(GLuint));
	glGenFramebuffers(panel->image_count, panel->framebuffers);
	int complete = 1;
	for (uint32_t index = 0; index < panel->image_count; index++) {
		glBindFramebuffer(GL_FRAMEBUFFER, panel->framebuffers[index]);
		glFramebufferTexture2D(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_TEXTURE_2D, images[index].image, 0);
		if (glCheckFramebufferStatus(GL_FRAMEBUFFER) != GL_FRAMEBUFFER_COMPLETE) complete = 0;
	}
	glBindFramebuffer(GL_FRAMEBUFFER, 0);
	return complete;
}

// xr_create_panel makes a panel of width × height pixels, in the eyes' format (so it encodes sRGB
// when they do), and says which it is. It fails with XR_ERROR_LIMIT_REACHED past XR_MOST_PANELS.
XrResult xr_create_panel(xr_program *program, int32_t width, int32_t height, int *index) {
	if (program->panel_count >= XR_MOST_PANELS) return XR_ERROR_LIMIT_REACHED;
	xr_panel *panel = &program->panels[program->panel_count];
	panel->width = width;
	panel->height = height;
	XrResult result = create_swapchain(program, width, height, &panel->swapchain);
	if (XR_FAILED(result)) return result;
	*index = program->panel_count;
	program->panel_count++; // destroyed with the others from here on, even if what follows fails
	XrSwapchainImageOpenGLESKHR *images = NULL;
	result = read_images(program, panel->swapchain, &panel->image_count, &images);
	if (XR_SUCCEEDED(result) && !make_panel_framebuffers(panel, images)) {
		program->framebuffer_incomplete = 1;
		result = XR_ERROR_VALIDATION_FAILURE;
	}
	free(images);
	return result;
}

static void destroy_panels(xr_program *program) {
	for (int index = 0; index < program->panel_count; index++) {
		xr_panel *panel = &program->panels[index];
		if (panel->framebuffers != NULL) {
			glDeleteFramebuffers(panel->image_count, panel->framebuffers);
			free(panel->framebuffers);
		}
		if (panel->swapchain != XR_NULL_HANDLE) program->call.xrDestroySwapchain(panel->swapchain);
		*panel = (xr_panel){0};
	}
	program->panel_count = 0;
}

void xr_destroy_swapchains(xr_program *program) {
	destroy_panels(program);
	for (int index = 0; index < XR_EYES; index++) {
		xr_eye *eye = &program->eyes[index];
		if (eye->framebuffers != NULL) {
			glDeleteFramebuffers(eye->image_count, eye->framebuffers);
			free(eye->framebuffers);
		}
		if (eye->depth != 0) glDeleteRenderbuffers(1, &eye->depth);
		if (eye->swapchain != XR_NULL_HANDLE) program->call.xrDestroySwapchain(eye->swapchain);
		*eye = (xr_eye){0};
	}
}
