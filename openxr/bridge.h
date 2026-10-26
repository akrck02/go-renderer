// The C side of an OpenXR session: the loader opened at run time, a table of the runtime's
// functions, and the calls whose structures are easier to fill in C. Go (session.go) holds one
// xr_program and decides what to call when.

#ifndef GO_RENDERER_OPENXR_BRIDGE_H
#define GO_RENDERER_OPENXR_BRIDGE_H

#include <jni.h>
#include <EGL/egl.h>
#include <GLES3/gl3.h>

#define XR_NO_PROTOTYPES // every function comes through xrGetInstanceProcAddr, none is linked
#define XR_USE_PLATFORM_ANDROID
#define XR_USE_GRAPHICS_API_OPENGL_ES
#include <openxr/openxr.h>
#include <openxr/openxr_platform.h>

// The functions there are before an instance exists.
#define XR_GLOBAL_FUNCTIONS(X)                \
	X(xrInitializeLoaderKHR)                  \
	X(xrEnumerateInstanceExtensionProperties) \
	X(xrCreateInstance)

// The functions an instance gives.
#define XR_INSTANCE_FUNCTIONS(X)              \
	X(xrDestroyInstance)                      \
	X(xrResultToString)                       \
	X(xrGetSystem)                            \
	X(xrGetOpenGLESGraphicsRequirementsKHR)   \
	X(xrCreateSession)                        \
	X(xrDestroySession)                       \
	X(xrBeginSession)                         \
	X(xrEndSession)                           \
	X(xrRequestExitSession)                   \
	X(xrPollEvent)                            \
	X(xrEnumerateReferenceSpaces)             \
	X(xrCreateReferenceSpace)                 \
	X(xrDestroySpace)                         \
	X(xrLocateSpace)                          \
	X(xrEnumerateViewConfigurationViews)      \
	X(xrEnumerateSwapchainFormats)            \
	X(xrCreateSwapchain)                      \
	X(xrDestroySwapchain)                     \
	X(xrEnumerateSwapchainImages)             \
	X(xrAcquireSwapchainImage)                \
	X(xrWaitSwapchainImage)                   \
	X(xrReleaseSwapchainImage)                \
	X(xrWaitFrame)                            \
	X(xrBeginFrame)                           \
	X(xrEndFrame)                             \
	X(xrLocateViews)                          \
	X(xrStringToPath)                         \
	X(xrCreateActionSet)                      \
	X(xrDestroyActionSet)                     \
	X(xrCreateAction)                         \
	X(xrSuggestInteractionProfileBindings)    \
	X(xrAttachSessionActionSets)              \
	X(xrSyncActions)                          \
	X(xrGetActionStateBoolean)                \
	X(xrGetActionStateFloat)                  \
	X(xrGetActionStateVector2f)               \
	X(xrCreateActionSpace)                    \
	X(xrApplyHapticFeedback)

// The functions XR_FB_passthrough gives, found only when the program asked to see the room and the
// runtime has the extension.
#define XR_PASSTHROUGH_FUNCTIONS(X) \
	X(xrCreatePassthroughFB)        \
	X(xrDestroyPassthroughFB)       \
	X(xrCreatePassthroughLayerFB)   \
	X(xrDestroyPassthroughLayerFB)

#define XR_FUNCTION_FIELD(name) PFN_##name name;

typedef struct {
	PFN_xrGetInstanceProcAddr xrGetInstanceProcAddr;
	XR_GLOBAL_FUNCTIONS(XR_FUNCTION_FIELD)
	XR_INSTANCE_FUNCTIONS(XR_FUNCTION_FIELD)
	XR_PASSTHROUGH_FUNCTIONS(XR_FUNCTION_FIELD)
} xr_functions;

#define XR_EYES 2
#define XR_HANDS 2 // left, right
#define XR_MOST_PANELS 8

// xr_eye is one eye's swapchain, and a framebuffer for each of its images sharing one depth buffer.
typedef struct {
	XrSwapchain swapchain;
	int32_t width, height;
	uint32_t image_count;
	GLuint *framebuffers;
	GLuint depth;
	int acquired; // an image is held between xr_acquire_images and xr_end_frame
} xr_eye;

// xr_panel is a panel's swapchain, a flat picture the runtime places in the world (a quad layer),
// and a framebuffer for each of its images. It has no depth buffer: a panel is an interface.
typedef struct {
	XrSwapchain swapchain;
	int32_t width, height;
	uint32_t image_count;
	GLuint *framebuffers;
	int acquired;   // an image is held between xr_acquire_panel and xr_end_frame
	uint32_t image; // which, while one is held
	int ready;    // an image was drawn and given back: the runtime has something to show
} xr_panel;

// xr_quad is a panel shown in a frame: where its middle is, which way it faces (+z toward who looks
// at it) and its size in metres.
typedef struct {
	int panel;
	XrPosef pose;
	float width, height;
} xr_quad;

// xr_controls are the controllers' actions and where their poses are.
typedef struct {
	XrActionSet set;
	XrAction grip, aim, trigger, squeeze, stick, stick_click, upper, lower, menu, vibration;
	XrPath hands[XR_HANDS];
	XrSpace grip_spaces[XR_HANDS], aim_spaces[XR_HANDS];
	int attached;
} xr_controls;

typedef struct {
	void *library;
	xr_functions call;
	XrInstance instance;
	XrSystemId system;
	XrSession session;
	XrSpace world, head;
	int has_local_floor;
	int wants_passthrough; // the program asked to see the room behind its scene (set before the instance)
	int has_passthrough;   // the runtime has XR_FB_passthrough and it was enabled
	XrPassthroughFB passthrough;
	XrPassthroughLayerFB passthrough_layer;
	int passthrough_running; // the room is shown under the eyes' layer
	int64_t format;
	int srgb;
	int framebuffer_incomplete;
	xr_eye eyes[XR_EYES];
	xr_panel panels[XR_MOST_PANELS];
	int panel_count;
	XrView views[XR_EYES]; // the frame's, located in xr_locate and submitted in xr_end_frame
	xr_controls controls;
} xr_program;

// xr_hand is a controller as Go reads it.
typedef struct {
	XrPosef grip, aim;
	int tracked;
	float trigger, squeeze, stick_x, stick_y;
	int stick_click, upper, lower;
} xr_hand;

// What xr_poll_event found.
enum { xr_event_none = 0, xr_event_state = 1, xr_event_loss = 2, xr_event_other = 3 };

// bridge.c
xr_program *xr_new_program(void);
const char *xr_open_loader(xr_program *program);
// The Java VM, the activity and EGL's handles come from Go as numbers (uintptr_t), as package
// android hands them over: some are not addresses (a global reference, EGLDisplay 1).
XrResult xr_start_loader(xr_program *program, uintptr_t java_vm, uintptr_t activity);
XrResult xr_create_instance(xr_program *program, const char *name, uintptr_t java_vm, uintptr_t activity, const char **missing);
XrResult xr_find_headset(xr_program *program);
XrResult xr_check_graphics(xr_program *program, int *supported);
XrResult xr_create_session(xr_program *program, uintptr_t display, uintptr_t config, uintptr_t context);
XrResult xr_create_spaces(xr_program *program);
XrResult xr_poll_event(xr_program *program, int *kind, int *state);
XrResult xr_begin_session(xr_program *program);
XrResult xr_end_session(xr_program *program);
XrResult xr_request_exit(xr_program *program);
void xr_result_name(xr_program *program, XrResult result, char *buffer);
void xr_destroy(xr_program *program);

// swapchain.c
XrResult xr_create_swapchains(xr_program *program);
XrResult xr_create_panel(xr_program *program, int32_t width, int32_t height, int *index);
void xr_destroy_swapchains(xr_program *program);

// passthrough.c
XrResult xr_start_passthrough(xr_program *program);
void xr_stop_passthrough(xr_program *program);

// frame.c
XrResult xr_wait_frame(xr_program *program, int *should_render, int64_t *display_time);
XrResult xr_begin_frame(xr_program *program);
XrResult xr_locate_head(xr_program *program, int64_t time, int *tracked, XrPosef *head);
XrResult xr_locate_eyes(xr_program *program, int64_t time, int *located);
XrResult xr_acquire_images(xr_program *program, GLuint *framebuffers);
XrResult xr_acquire_panel(xr_program *program, int index, GLuint *framebuffer);
XrResult xr_end_frame(xr_program *program, int64_t time, int render, const xr_quad *quads, int quad_count);

// actions.c
XrResult xr_create_controls(xr_program *program);
XrResult xr_read_controls(xr_program *program, int64_t time, xr_hand *hands, int *menu);
XrResult xr_vibrate(xr_program *program, int hand, float amplitude, int64_t nanoseconds);
void xr_destroy_controls(xr_program *program);

#endif
