//go:build android

// The library's entry point and Android's callbacks. Android calls them on its own interface thread;
// they hand everything to Go (activity.go), which draws on a thread of its own.

#include <android/configuration.h>
#include <android/log.h>
#include <dlfcn.h>
#include <stdlib.h>
#include "_cgo_export.h"
#include "activity.h"

ANativeActivity *current_activity;
static int main_started;

static void on_start(ANativeActivity *activity) { goActivityChanged(0); }
static void on_resume(ANativeActivity *activity) { goActivityChanged(1); }
static void on_pause(ANativeActivity *activity) { goActivityChanged(2); }
static void on_stop(ANativeActivity *activity) { goActivityChanged(3); }
static void on_destroy(ANativeActivity *activity) { goActivityChanged(4); }
static void on_configuration_changed(ANativeActivity *activity) { goActivityChanged(5); }
static void on_window_focus_changed(ANativeActivity *activity, int focused) { goActivityChanged(focused ? 6 : 7); }
static void on_low_memory(ANativeActivity *activity) {}
static void *on_save_instance_state(ANativeActivity *activity, size_t *size) {
	*size = 0;
	return NULL;
}

static void on_window_created(ANativeActivity *activity, ANativeWindow *window) { goWindowCreated(window); }
static void on_window_destroyed(ANativeActivity *activity, ANativeWindow *window) { goWindowDestroyed(window); }
static void on_window_redraw_needed(ANativeActivity *activity, ANativeWindow *window) { goActivityChanged(8); }
static void on_window_resized(ANativeActivity *activity, ANativeWindow *window) { goActivityChanged(8); }
static void on_input_queue_created(ANativeActivity *activity, AInputQueue *queue) { goInputQueueCreated(queue); }
static void on_input_queue_destroyed(ANativeActivity *activity, AInputQueue *queue) { goInputQueueDestroyed(queue); }

// ANativeActivity_onCreate is what Android calls when it makes the activity; it may make it many
// times in one process (after a rotation, back from the background). The first time, it starts the
// program's main, which the Go runtime does not run by itself in a shared library.
void ANativeActivity_onCreate(ANativeActivity *activity, void *saved_state, size_t saved_state_size) {
	current_activity = activity;
	if (!main_started) {
		uintptr_t main_code = (uintptr_t)dlsym(RTLD_DEFAULT, "main.main");
		if (main_code == 0) {
			__android_log_print(ANDROID_LOG_FATAL, "Go", "the library has no main.main");
			abort();
		}
		main_started = 1;
		__android_log_print(ANDROID_LOG_INFO, "Go", "the library is loaded; starting the Go runtime");
		// the program's own folder becomes os.TempDir: crash reports go there
		goStartMain(main_code, (char *)activity->internalDataPath);
	}
	activity->callbacks->onStart = on_start;
	activity->callbacks->onResume = on_resume;
	activity->callbacks->onPause = on_pause;
	activity->callbacks->onStop = on_stop;
	activity->callbacks->onDestroy = on_destroy;
	activity->callbacks->onConfigurationChanged = on_configuration_changed;
	activity->callbacks->onWindowFocusChanged = on_window_focus_changed;
	activity->callbacks->onLowMemory = on_low_memory;
	activity->callbacks->onSaveInstanceState = on_save_instance_state;
	activity->callbacks->onNativeWindowCreated = on_window_created;
	activity->callbacks->onNativeWindowDestroyed = on_window_destroyed;
	activity->callbacks->onNativeWindowRedrawNeeded = on_window_redraw_needed;
	activity->callbacks->onNativeWindowResized = on_window_resized;
	activity->callbacks->onInputQueueCreated = on_input_queue_created;
	activity->callbacks->onInputQueueDestroyed = on_input_queue_destroyed;
}

void log_text(int priority, const char *text) { __android_log_write(priority, "Go", text); }

// surface_open makes an OpenGL ES context drawing into the window: version 3 when the phone has
// it, else 2. The renderers need only 2; 3 is for later (instancing). Returns 0 when it cannot.
int surface_open(ANativeWindow *window, surface *opened) {
	opened->window = window;
	opened->display = eglGetDisplay(EGL_DEFAULT_DISPLAY);
	if (opened->display == EGL_NO_DISPLAY || !eglInitialize(opened->display, NULL, NULL)) {
		return 0;
	}
	const EGLint wanted[] = {
		EGL_RENDERABLE_TYPE, EGL_OPENGL_ES2_BIT, EGL_SURFACE_TYPE, EGL_WINDOW_BIT,
		EGL_RED_SIZE, 8, EGL_GREEN_SIZE, 8, EGL_BLUE_SIZE, 8, EGL_DEPTH_SIZE, 24, EGL_NONE,
	};
	const EGLint enough[] = {
		EGL_RENDERABLE_TYPE, EGL_OPENGL_ES2_BIT, EGL_SURFACE_TYPE, EGL_WINDOW_BIT,
		EGL_RED_SIZE, 8, EGL_GREEN_SIZE, 8, EGL_BLUE_SIZE, 8, EGL_DEPTH_SIZE, 16, EGL_NONE,
	};
	EGLConfig config;
	EGLint found = 0;
	// a 24-bit depth buffer keeps a 3D floor and what stands on it apart; 16 is what every phone has
	if (!eglChooseConfig(opened->display, wanted, &config, 1, &found) || found == 0) {
		if (!eglChooseConfig(opened->display, enough, &config, 1, &found) || found == 0) {
			return 0;
		}
	}
	opened->surface = eglCreateWindowSurface(opened->display, config, window, NULL);
	if (opened->surface == EGL_NO_SURFACE) {
		return 0;
	}
	for (int version = 3; version >= 2; version--) {
		const EGLint attributes[] = {EGL_CONTEXT_CLIENT_VERSION, version, EGL_NONE};
		opened->context = eglCreateContext(opened->display, config, EGL_NO_CONTEXT, attributes);
		if (opened->context != EGL_NO_CONTEXT) {
			opened->version = version;
			break;
		}
	}
	if (opened->context == EGL_NO_CONTEXT) {
		eglDestroySurface(opened->display, opened->surface);
		return 0;
	}
	if (!eglMakeCurrent(opened->display, opened->surface, opened->surface, opened->context)) {
		surface_close(opened);
		return 0;
	}
	return 1;
}

void surface_close(surface *opened) {
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
	opened->context = EGL_NO_CONTEXT;
	opened->surface = EGL_NO_SURFACE;
	opened->window = NULL; // Android takes it away after this
}

int surface_present(surface *opened) { return eglSwapBuffers(opened->display, opened->surface); }

// surface_size is the size of what is drawn into, in pixels.
void surface_size(surface *opened, int *width, int *height) {
	EGLint value = 0;
	eglQuerySurface(opened->display, opened->surface, EGL_WIDTH, &value);
	*width = value;
	eglQuerySurface(opened->display, opened->surface, EGL_HEIGHT, &value);
	*height = value;
}

// screen_density is the screen's dots per inch, as Android's configuration says.
int screen_density(void) {
	AConfiguration *configuration = AConfiguration_new();
	AConfiguration_fromAssetManager(configuration, current_activity->assetManager);
	int density = AConfiguration_getDensity(configuration);
	AConfiguration_delete(configuration);
	return density;
}

// surface_behind tells whether the window has changed size (a rotation) and the surface not yet:
// EGL's surface takes the window's new size only once a frame has been shown, so one more frame is
// needed, at the new size.
int surface_behind(surface *opened) {
	if (opened->window == NULL) return 0;
	int width, height;
	surface_size(opened, &width, &height);
	return width != ANativeWindow_getWidth(opened->window) || height != ANativeWindow_getHeight(opened->window);
}
