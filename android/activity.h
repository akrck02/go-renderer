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
	ANativeWindow *window; // what it draws into: its size is the one to draw at
	int version; // the OpenGL ES major version the context speaks (3 or 2)
} surface;

int surface_open(ANativeWindow *window, surface *opened);
void surface_close(surface *opened);
int surface_present(surface *opened);
void surface_size(surface *opened, int *width, int *height);
int surface_behind(surface *opened);

// Called on the drawing thread, with the JNI environment of that thread.
JNIEnv *attach_thread(void);
int window_insets(JNIEnv *env, int *top, int *bottom, int *left, int *right);
int bar_height(JNIEnv *env, const char *name);
int night_mode(JNIEnv *env);
int key_character(JNIEnv *env, int action, int code, int meta);
void soft_keyboard(int show);
void leave_program(void);

// Notifications (notify.c): safe from any thread, which they attach to Java.
int notifications_allowed(void);
void ask_for_notifications(void);
int post_notification(int id, const char *channel_id, const char *channel_label, const char *title, const char *text);
int system_colours(JNIEnv *env, const char **palettes, int palette_count, const int *tones, int tone_count, unsigned int *colours);
int screen_density(void);

// The buzz of a long press (system.c): safe from any thread.
void feel_long_press(void);

// log_text writes a line to Android's log, under the tag Go.
void log_text(int priority, const char *text);

#endif
