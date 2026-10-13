// The C side of the Android window: what Android's NativeActivity calls, EGL, and the system's
// insets and theme through JNI. Go does the rest (activity.go, loop.go).

#ifndef GO_RENDERER_ANDROID_ACTIVITY_H
#define GO_RENDERER_ANDROID_ACTIVITY_H

#include <EGL/egl.h>
#include <android/input.h>
#include <android/looper.h>
#include <android/native_activity.h>
#include <android/native_window.h>
#include <jni.h>
#include <stdint.h>

// The activity this library runs in, set when Android creates it.
extern ANativeActivity *current_activity;

// surface is an OpenGL ES context drawing into a window.
typedef struct {
	EGLDisplay display;
	EGLSurface surface;
	EGLContext context;
	int version; // the OpenGL ES major version the context speaks (3 or 2)
} surface;

int surface_open(ANativeWindow *window, surface *opened);
void surface_close(surface *opened);
int surface_present(surface *opened);
void surface_size(surface *opened, int *width, int *height);

// Called on the drawing thread, with the JNI environment of that thread.
JNIEnv *attach_thread(void);
int window_insets(JNIEnv *env, int *top, int *bottom, int *left, int *right);
int bar_height(JNIEnv *env, const char *name);
int night_mode(JNIEnv *env);
int screen_density(void);

// log_text writes a line to Android's log, under the tag Go.
void log_text(int priority, const char *text);

#endif
