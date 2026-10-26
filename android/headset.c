//go:build android

// The C side of running without a window (headset.go): an OpenGL ES 3 context for an OpenXR runtime,
// which draws into its own images and never shows a window surface, and the input queue read and let
// go of.

#include <string.h>
#include "activity.h"

// pbuffer_config finds a configuration OpenGL ES 3 can draw with on a pbuffer, 8 bits a channel.
// The pbuffer is only there to make the context current: what is drawn goes to the runtime's
// images, which have their own depth, so the pbuffer needs none.
static int pbuffer_config(EGLDisplay display, EGLConfig *config) {
	const EGLint wanted[] = {
		EGL_RENDERABLE_TYPE, EGL_OPENGL_ES3_BIT, EGL_SURFACE_TYPE, EGL_PBUFFER_BIT,
		EGL_RED_SIZE, 8, EGL_GREEN_SIZE, 8, EGL_BLUE_SIZE, 8, EGL_ALPHA_SIZE, 8, EGL_NONE,
	};
	EGLint found = 0;
	return eglChooseConfig(display, wanted, config, 1, &found) && found > 0;
}

// any_config is the fallback for a driver with no pbuffer configuration: the context is then made
// current with no surface at all, which EGL_KHR_surfaceless_context allows.
static int any_config(EGLDisplay display, EGLConfig *config) {
	const char *extensions = eglQueryString(display, EGL_EXTENSIONS);
	if (extensions == NULL || strstr(extensions, "EGL_KHR_surfaceless_context") == NULL) {
		return 0;
	}
	const EGLint wanted[] = {
		EGL_RENDERABLE_TYPE, EGL_OPENGL_ES3_BIT,
		EGL_RED_SIZE, 8, EGL_GREEN_SIZE, 8, EGL_BLUE_SIZE, 8, EGL_ALPHA_SIZE, 8, EGL_NONE,
	};
	EGLint found = 0;
	return eglChooseConfig(display, wanted, config, 1, &found) && found > 0;
}

static EGLSurface small_pbuffer(EGLDisplay display, EGLConfig config) {
	const EGLint size[] = {EGL_WIDTH, 16, EGL_HEIGHT, 16, EGL_NONE};
	return eglCreatePbufferSurface(display, config, size);
}

// headset_context_open makes an OpenGL ES 3 context current on this thread, on a small pbuffer or,
// when the driver has no pbuffer configuration, on no surface. Returns 0 when it cannot; what was
// made is then let go.
int headset_context_open(headset_context *opened) {
	opened->display = EGL_NO_DISPLAY;
	opened->context = EGL_NO_CONTEXT;
	opened->surface = EGL_NO_SURFACE;
	EGLDisplay display = eglGetDisplay(EGL_DEFAULT_DISPLAY);
	if (display == EGL_NO_DISPLAY || !eglInitialize(display, NULL, NULL)) {
		return 0;
	}
	opened->display = display;
	int with_pbuffer = pbuffer_config(display, &opened->config);
	if (!with_pbuffer && !any_config(display, &opened->config)) {
		headset_context_close(opened);
		return 0;
	}
	const EGLint attributes[] = {EGL_CONTEXT_CLIENT_VERSION, 3, EGL_NONE};
	opened->context = eglCreateContext(display, opened->config, EGL_NO_CONTEXT, attributes);
	if (opened->context == EGL_NO_CONTEXT) {
		headset_context_close(opened);
		return 0;
	}
	if (with_pbuffer) {
		opened->surface = small_pbuffer(display, opened->config);
		if (opened->surface == EGL_NO_SURFACE) {
			headset_context_close(opened);
			return 0;
		}
	}
	if (!eglMakeCurrent(display, opened->surface, opened->surface, opened->context)) {
		headset_context_close(opened);
		return 0;
	}
	return 1;
}

void headset_context_close(headset_context *opened) {
	if (opened->display == EGL_NO_DISPLAY) {
		return;
	}
	eglMakeCurrent(opened->display, EGL_NO_SURFACE, EGL_NO_SURFACE, EGL_NO_CONTEXT);
	if (opened->context != EGL_NO_CONTEXT) {
		eglDestroyContext(opened->display, opened->context);
	}
	if (opened->surface != EGL_NO_SURFACE) {
		eglDestroySurface(opened->display, opened->surface);
	}
	// the display is not terminated: EGL shares one per process, and other threads may hold it
	opened->context = EGL_NO_CONTEXT;
	opened->surface = EGL_NO_SURFACE;
	opened->display = EGL_NO_DISPLAY;
}

// headset_drain_queue takes every waiting event from the input queue and finishes it, as Android
// requires (an unfinished event makes it report the program as not responding). The program's
// controls come through OpenXR, so a touch is taken and dropped; keys are left to Android (the
// volume), except back, which would otherwise end the activity behind the program's back.
void headset_drain_queue(AInputQueue *queue) {
	AInputEvent *event = NULL;
	while (AInputQueue_getEvent(queue, &event) >= 0) {
		if (AInputQueue_preDispatchEvent(queue, event) != 0) {
			continue; // the on-screen keyboard took it
		}
		int handled = AInputEvent_getType(event) == AINPUT_EVENT_TYPE_MOTION;
		if (AInputEvent_getType(event) == AINPUT_EVENT_TYPE_KEY && AKeyEvent_getKeyCode(event) == AKEYCODE_BACK) {
			handled = 1;
		}
		AInputQueue_finishEvent(queue, event, handled);
	}
}

// headset_handles_fill hands over what an OpenXR runtime needs, as numbers: cgo maps jobject and
// EGL's handles to uintptr in Go, and some of them are not addresses at all. The activity is a
// global reference, for a runtime that keeps it beyond one JNI call; headset_handles_release lets
// it go.
void headset_handles_fill(JNIEnv *env, headset_context *opened, headset_handles *handles) {
	handles->java_vm = (uintptr_t)current_activity->vm;
	handles->activity = env == NULL ? 0 : (uintptr_t)(*env)->NewGlobalRef(env, current_activity->clazz);
	handles->display = (uintptr_t)opened->display;
	handles->config = (uintptr_t)opened->config;
	handles->context = (uintptr_t)opened->context;
}

void headset_handles_release(JNIEnv *env, headset_handles *handles) {
	if (env != NULL && handles->activity != 0) (*env)->DeleteGlobalRef(env, (jobject)handles->activity);
	handles->activity = 0;
}

// detach_thread lets the Java VM forget this thread: a thread that ends while attached stops the
// whole program.
void detach_thread(void) {
	JavaVM *vm = current_activity->vm;
	(*vm)->DetachCurrentThread(vm);
}

void finish_activity(void) { ANativeActivity_finish(current_activity); }
