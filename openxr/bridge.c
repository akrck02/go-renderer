//go:build android

// The loader, the instance, the headset, the session and its spaces. The loader is opened when the
// program runs, so an application without it still starts and can say what is missing.

#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "bridge.h"

xr_program *xr_new_program(void) { return calloc(1, sizeof(xr_program)); }

// resolve asks the loader (or, once there is an instance, the runtime) for a function by name.
static XrResult resolve(xr_program *program, const char *name, PFN_xrVoidFunction *function) {
	return program->call.xrGetInstanceProcAddr(program->instance, name, function);
}

#define XR_RESOLVE_GLOBAL(name)                                                         \
	if (XR_FAILED(resolve(program, #name, (PFN_xrVoidFunction *)&program->call.name))) \
		return "the loader has no " #name;

// xr_open_loader opens libopenxr_loader.so and finds the functions there are before an instance.
// It returns NULL, or what went wrong.
const char *xr_open_loader(xr_program *program) {
	program->library = dlopen("libopenxr_loader.so", RTLD_NOW | RTLD_LOCAL);
	if (program->library == NULL) return dlerror();
	program->call.xrGetInstanceProcAddr = (PFN_xrGetInstanceProcAddr)dlsym(program->library, "xrGetInstanceProcAddr");
	if (program->call.xrGetInstanceProcAddr == NULL) return "the loader has no xrGetInstanceProcAddr";
	XR_GLOBAL_FUNCTIONS(XR_RESOLVE_GLOBAL)
	return NULL;
}

// xr_start_loader hands the loader the Java VM and the activity: on Android it finds the runtime
// through them, before anything else is called.
XrResult xr_start_loader(xr_program *program, uintptr_t java_vm, uintptr_t activity) {
	XrLoaderInitInfoAndroidKHR info = {XR_TYPE_LOADER_INIT_INFO_ANDROID_KHR};
	info.applicationVM = (void *)java_vm;
	info.applicationContext = (void *)activity;
	return program->call.xrInitializeLoaderKHR((const XrLoaderInitInfoBaseHeaderKHR *)&info);
}

static int offered(const XrExtensionProperties *extensions, uint32_t count, const char *name) {
	for (uint32_t index = 0; index < count; index++) {
		if (strcmp(extensions[index].extensionName, name) == 0) return 1;
	}
	return 0;
}

// read_extensions lists the runtime's instance extensions; the caller frees the list.
static XrResult read_extensions(xr_program *program, XrExtensionProperties **extensions, uint32_t *count) {
	XrResult result = program->call.xrEnumerateInstanceExtensionProperties(NULL, 0, count, NULL);
	if (XR_FAILED(result)) return result;
	*extensions = calloc(*count, sizeof(XrExtensionProperties));
	for (uint32_t index = 0; index < *count; index++) (*extensions)[index].type = XR_TYPE_EXTENSION_PROPERTIES;
	return program->call.xrEnumerateInstanceExtensionProperties(NULL, *count, count, *extensions);
}

#define XR_RESOLVE_INSTANCE(name)                                                          \
	if (XR_FAILED(result = resolve(program, #name, (PFN_xrVoidFunction *)&program->call.name))) \
		return result;

static XrResult resolve_instance_functions(xr_program *program) {
	XrResult result = XR_SUCCESS;
	XR_INSTANCE_FUNCTIONS(XR_RESOLVE_INSTANCE)
	if (program->has_passthrough) {
		XR_PASSTHROUGH_FUNCTIONS(XR_RESOLVE_INSTANCE)
	}
	return result;
}

// create_instance asks for OpenXR 1.1 and, from a runtime that only knows 1.0, for 1.0: nothing
// used here is newer than 1.0 with its extensions.
static XrResult create_instance(xr_program *program, XrInstanceCreateInfo *info) {
	info->applicationInfo.apiVersion = XR_API_VERSION_1_1;
	XrResult result = program->call.xrCreateInstance(info, &program->instance);
	if (result != XR_ERROR_API_VERSION_UNSUPPORTED) return result;
	info->applicationInfo.apiVersion = XR_API_VERSION_1_0;
	return program->call.xrCreateInstance(info, &program->instance);
}

// xr_create_instance starts the runtime for the program, with the extensions it cannot do without
// (Android, OpenGL ES), the floor-level space when the runtime has it, and the passthrough when the
// program asked for it and the runtime has it. A missing extension is named in missing.
XrResult xr_create_instance(xr_program *program, const char *name, uintptr_t java_vm, uintptr_t activity, const char **missing) {
	XrExtensionProperties *extensions = NULL;
	uint32_t count = 0;
	XrResult result = read_extensions(program, &extensions, &count);
	if (XR_FAILED(result)) {
		free(extensions);
		return result;
	}
	static const char *needed[] = {XR_KHR_ANDROID_CREATE_INSTANCE_EXTENSION_NAME, XR_KHR_OPENGL_ES_ENABLE_EXTENSION_NAME};
	const char *enabled[4] = {needed[0], needed[1]};
	uint32_t enabled_count = 2;
	for (int index = 0; index < 2; index++) {
		if (offered(extensions, count, needed[index])) continue;
		*missing = needed[index];
		free(extensions);
		return XR_ERROR_EXTENSION_NOT_PRESENT;
	}
	if (offered(extensions, count, XR_EXT_LOCAL_FLOOR_EXTENSION_NAME)) enabled[enabled_count++] = XR_EXT_LOCAL_FLOOR_EXTENSION_NAME;
	if (program->wants_passthrough && offered(extensions, count, XR_FB_PASSTHROUGH_EXTENSION_NAME)) {
		enabled[enabled_count++] = XR_FB_PASSTHROUGH_EXTENSION_NAME;
		program->has_passthrough = 1;
	}
	free(extensions);

	XrInstanceCreateInfoAndroidKHR android = {XR_TYPE_INSTANCE_CREATE_INFO_ANDROID_KHR};
	android.applicationVM = (void *)java_vm;
	android.applicationActivity = (void *)activity;
	XrInstanceCreateInfo info = {XR_TYPE_INSTANCE_CREATE_INFO};
	info.next = &android;
	strncpy(info.applicationInfo.applicationName, name, XR_MAX_APPLICATION_NAME_SIZE - 1);
	info.applicationInfo.applicationVersion = 1;
	strncpy(info.applicationInfo.engineName, "Go Renderer", XR_MAX_ENGINE_NAME_SIZE - 1);
	info.applicationInfo.engineVersion = 1;
	info.enabledExtensionCount = enabled_count;
	info.enabledExtensionNames = enabled;
	result = create_instance(program, &info);
	if (XR_FAILED(result)) return result;
	return resolve_instance_functions(program);
}

// xr_find_headset finds the head-mounted display the runtime drives.
XrResult xr_find_headset(xr_program *program) {
	XrSystemGetInfo info = {XR_TYPE_SYSTEM_GET_INFO};
	info.formFactor = XR_FORM_FACTOR_HEAD_MOUNTED_DISPLAY;
	return program->call.xrGetSystem(program->instance, &info, &program->system);
}

// xr_check_graphics asks the runtime which OpenGL ES it needs (it must be asked before a session
// is made) and compares it with the current context's.
XrResult xr_check_graphics(xr_program *program, int *supported) {
	XrGraphicsRequirementsOpenGLESKHR requirements = {XR_TYPE_GRAPHICS_REQUIREMENTS_OPENGL_ES_KHR};
	XrResult result = program->call.xrGetOpenGLESGraphicsRequirementsKHR(program->instance, program->system, &requirements);
	if (XR_FAILED(result)) return result;
	GLint major = 0, minor = 0;
	glGetIntegerv(GL_MAJOR_VERSION, &major);
	glGetIntegerv(GL_MINOR_VERSION, &minor);
	*supported = XR_MAKE_VERSION(major, minor, 0) >= requirements.minApiVersionSupported;
	return XR_SUCCESS;
}

// xr_create_session makes the session for the EGL context the drawing thread holds.
XrResult xr_create_session(xr_program *program, uintptr_t display, uintptr_t config, uintptr_t context) {
	XrGraphicsBindingOpenGLESAndroidKHR binding = {XR_TYPE_GRAPHICS_BINDING_OPENGL_ES_ANDROID_KHR};
	binding.display = (EGLDisplay)display;
	binding.config = (EGLConfig)config;
	binding.context = (EGLContext)context;
	XrSessionCreateInfo info = {XR_TYPE_SESSION_CREATE_INFO};
	info.next = &binding;
	info.systemId = program->system;
	return program->call.xrCreateSession(program->instance, &info, &program->session);
}

// world_space is the best space the runtime has for a scene: the floor where the person stands
// (LOCAL_FLOOR), the play area's floor (STAGE), or where the head was when the session began (LOCAL).
static XrReferenceSpaceType world_space(const XrReferenceSpaceType *spaces, uint32_t count) {
	static const XrReferenceSpaceType preferred[] = {XR_REFERENCE_SPACE_TYPE_LOCAL_FLOOR, XR_REFERENCE_SPACE_TYPE_STAGE};
	for (int choice = 0; choice < 2; choice++) {
		for (uint32_t index = 0; index < count; index++) {
			if (spaces[index] == preferred[choice]) return preferred[choice];
		}
	}
	return XR_REFERENCE_SPACE_TYPE_LOCAL;
}

static XrResult create_space(xr_program *program, XrReferenceSpaceType type, XrSpace *space) {
	XrReferenceSpaceCreateInfo info = {XR_TYPE_REFERENCE_SPACE_CREATE_INFO};
	info.referenceSpaceType = type;
	info.poseInReferenceSpace.orientation.w = 1;
	return program->call.xrCreateReferenceSpace(program->session, &info, space);
}

// xr_create_spaces makes the space the scene is placed in, and the head's.
XrResult xr_create_spaces(xr_program *program) {
	uint32_t count = 0;
	XrResult result = program->call.xrEnumerateReferenceSpaces(program->session, 0, &count, NULL);
	if (XR_FAILED(result)) return result;
	XrReferenceSpaceType *spaces = calloc(count, sizeof(XrReferenceSpaceType));
	result = program->call.xrEnumerateReferenceSpaces(program->session, count, &count, spaces);
	XrReferenceSpaceType chosen = world_space(spaces, XR_FAILED(result) ? 0 : count);
	free(spaces);
	if (XR_FAILED(result)) return result;
	result = create_space(program, chosen, &program->world);
	if (XR_FAILED(result)) return result;
	return create_space(program, XR_REFERENCE_SPACE_TYPE_VIEW, &program->head);
}

// xr_poll_event takes the runtime's next event, if any: a new session state, the runtime going
// away, or something this package does not follow.
XrResult xr_poll_event(xr_program *program, int *kind, int *state) {
	XrEventDataBuffer event = {XR_TYPE_EVENT_DATA_BUFFER};
	XrResult result = program->call.xrPollEvent(program->instance, &event);
	*kind = xr_event_none;
	if (result == XR_EVENT_UNAVAILABLE || XR_FAILED(result)) return result == XR_EVENT_UNAVAILABLE ? XR_SUCCESS : result;
	switch (event.type) {
	case XR_TYPE_EVENT_DATA_SESSION_STATE_CHANGED:
		*kind = xr_event_state;
		*state = ((XrEventDataSessionStateChanged *)&event)->state;
		break;
	case XR_TYPE_EVENT_DATA_INSTANCE_LOSS_PENDING:
		*kind = xr_event_loss;
		break;
	default:
		*kind = xr_event_other;
	}
	return result;
}

XrResult xr_begin_session(xr_program *program) {
	XrSessionBeginInfo info = {XR_TYPE_SESSION_BEGIN_INFO};
	info.primaryViewConfigurationType = XR_VIEW_CONFIGURATION_TYPE_PRIMARY_STEREO;
	return program->call.xrBeginSession(program->session, &info);
}

XrResult xr_end_session(xr_program *program) { return program->call.xrEndSession(program->session); }

// xr_request_exit asks the runtime to stop a running session the program wants to leave: it then
// goes STOPPING (xr_end_session) and, usually, EXITING.
XrResult xr_request_exit(xr_program *program) { return program->call.xrRequestExitSession(program->session); }

// xr_result_name writes a result's name (XR_ERROR_…) into a buffer of XR_MAX_RESULT_STRING_SIZE;
// before there is an instance to ask, its number.
void xr_result_name(xr_program *program, XrResult result, char *buffer) {
	if (program->instance != XR_NULL_HANDLE && program->call.xrResultToString != NULL &&
	    XR_SUCCEEDED(program->call.xrResultToString(program->instance, result, buffer))) {
		return;
	}
	snprintf(buffer, XR_MAX_RESULT_STRING_SIZE, "XrResult %d", (int)result);
}

// xr_destroy lets go of everything, in the order it was made backwards. It is called on the
// drawing thread: the framebuffers are OpenGL ES objects of its context.
void xr_destroy(xr_program *program) {
	if (program == NULL) return;
	xr_destroy_controls(program);
	xr_destroy_swapchains(program);
	xr_stop_passthrough(program);
	if (program->head != XR_NULL_HANDLE) program->call.xrDestroySpace(program->head);
	if (program->world != XR_NULL_HANDLE) program->call.xrDestroySpace(program->world);
	if (program->session != XR_NULL_HANDLE) program->call.xrDestroySession(program->session);
	if (program->instance != XR_NULL_HANDLE) program->call.xrDestroyInstance(program->instance);
	if (program->library != NULL) dlclose(program->library);
	free(program);
}
