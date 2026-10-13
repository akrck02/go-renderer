//go:build android

// What the window asks Android through JNI: how much of the screen the system's bars cover,
// whether the phone is in its dark theme, and the colours it took from the wallpaper. Every call checks for a Java exception and gives up
// quietly: the window then works with no insets and the light theme.

#include <stddef.h>
#include <stdio.h>
#include "activity.h"

JNIEnv *attach_thread(void) {
	JNIEnv *env = NULL;
	JavaVM *vm = current_activity->vm;
	if ((*vm)->GetEnv(vm, (void **)&env, JNI_VERSION_1_6) == JNI_OK) {
		return env;
	}
	if ((*vm)->AttachCurrentThread(vm, &env, NULL) != JNI_OK) {
		return NULL;
	}
	return env;
}

static int failed(JNIEnv *env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
		return 1;
	}
	return 0;
}

static jobject call_object(JNIEnv *env, jobject object, const char *name, const char *signature) {
	if (object == NULL) return NULL;
	jclass class = (*env)->GetObjectClass(env, object);
	jmethodID method = (*env)->GetMethodID(env, class, name, signature);
	(*env)->DeleteLocalRef(env, class);
	if (failed(env) || method == NULL) return NULL;
	jobject result = (*env)->CallObjectMethod(env, object, method);
	if (failed(env)) return NULL;
	return result;
}

static int call_int(JNIEnv *env, jobject object, const char *name) {
	if (object == NULL) return 0;
	jclass class = (*env)->GetObjectClass(env, object);
	jmethodID method = (*env)->GetMethodID(env, class, name, "()I");
	(*env)->DeleteLocalRef(env, class);
	if (failed(env) || method == NULL) return 0;
	int result = (*env)->CallIntMethod(env, object, method);
	if (failed(env)) return 0;
	return result;
}

// window_insets reads what the status bar, the navigation or gesture bar and a display cutout cover,
// in pixels. It returns 0 when the window has none yet (it is not shown).
int window_insets(JNIEnv *env, int *top, int *bottom, int *left, int *right) {
	jobject activity = current_activity->clazz;
	jobject window = call_object(env, activity, "getWindow", "()Landroid/view/Window;");
	jobject decor = call_object(env, window, "getDecorView", "()Landroid/view/View;");
	jobject insets = call_object(env, decor, "getRootWindowInsets", "()Landroid/view/WindowInsets;");
	int found = insets != NULL;
	if (found) {
		*top = call_int(env, insets, "getSystemWindowInsetTop");
		*bottom = call_int(env, insets, "getSystemWindowInsetBottom");
		*left = call_int(env, insets, "getSystemWindowInsetLeft");
		*right = call_int(env, insets, "getSystemWindowInsetRight");
		// a camera's cutout: on its side of the screen when the phone lies on its side (Android 9+)
		jobject cutout = call_object(env, insets, "getDisplayCutout", "()Landroid/view/DisplayCutout;");
		if (cutout != NULL) {
			int safe;
			if ((safe = call_int(env, cutout, "getSafeInsetTop")) > *top) *top = safe;
			if ((safe = call_int(env, cutout, "getSafeInsetBottom")) > *bottom) *bottom = safe;
			if ((safe = call_int(env, cutout, "getSafeInsetLeft")) > *left) *left = safe;
			if ((safe = call_int(env, cutout, "getSafeInsetRight")) > *right) *right = safe;
			(*env)->DeleteLocalRef(env, cutout);
		}
		(*env)->DeleteLocalRef(env, insets);
	}
	if (decor != NULL) (*env)->DeleteLocalRef(env, decor);
	if (window != NULL) (*env)->DeleteLocalRef(env, window);
	return found;
}

// bar_height reads one of Android's own sizes ("status_bar_height"), in pixels; 0 if unknown.
int bar_height(JNIEnv *env, const char *name) {
	jobject resources = call_object(env, current_activity->clazz, "getResources", "()Landroid/content/res/Resources;");
	if (resources == NULL) return 0;
	jclass class = (*env)->GetObjectClass(env, resources);
	jmethodID identifier = (*env)->GetMethodID(env, class, "getIdentifier", "(Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;)I");
	jmethodID size = (*env)->GetMethodID(env, class, "getDimensionPixelSize", "(I)I");
	(*env)->DeleteLocalRef(env, class);
	int result = 0;
	if (!failed(env) && identifier != NULL && size != NULL) {
		jstring resource_name = (*env)->NewStringUTF(env, name);
		jstring kind = (*env)->NewStringUTF(env, "dimen");
		jstring package = (*env)->NewStringUTF(env, "android");
		int id = (*env)->CallIntMethod(env, resources, identifier, resource_name, kind, package);
		if (!failed(env) && id != 0) {
			result = (*env)->CallIntMethod(env, resources, size, id);
			if (failed(env)) result = 0;
		}
		(*env)->DeleteLocalRef(env, resource_name);
		(*env)->DeleteLocalRef(env, kind);
		(*env)->DeleteLocalRef(env, package);
	}
	(*env)->DeleteLocalRef(env, resources);
	return result;
}

// night_mode tells whether the system is in its dark theme (Configuration.uiMode's night bits).
int night_mode(JNIEnv *env) {
	jobject resources = call_object(env, current_activity->clazz, "getResources", "()Landroid/content/res/Resources;");
	jobject configuration = call_object(env, resources, "getConfiguration", "()Landroid/content/res/Configuration;");
	int night = 0;
	if (configuration != NULL) {
		jclass class = (*env)->GetObjectClass(env, configuration);
		jfieldID field = (*env)->GetFieldID(env, class, "uiMode", "I");
		(*env)->DeleteLocalRef(env, class);
		if (!failed(env) && field != NULL) {
			int mode = (*env)->GetIntField(env, configuration, field);
			night = (mode & 0x30) == 0x20; // UI_MODE_NIGHT_MASK, UI_MODE_NIGHT_YES
		}
		(*env)->DeleteLocalRef(env, configuration);
	}
	if (resources != NULL) (*env)->DeleteLocalRef(env, resources);
	return night;
}

// system_colours reads Android's own palettes, made from the wallpaper (Android 12 and later): the
// colours named system_<palette>_<tone> for each palette and tone given, as ARGB, into colours. It
// returns 0 when the phone has no such colours (an older Android), and reads nothing then.
int system_colours(JNIEnv *env, const char **palettes, int palette_count, const int *tones, int tone_count, unsigned int *colours) {
	jobject resources = call_object(env, current_activity->clazz, "getResources", "()Landroid/content/res/Resources;");
	if (resources == NULL) return 0;
	jclass class = (*env)->GetObjectClass(env, resources);
	jmethodID identifier = (*env)->GetMethodID(env, class, "getIdentifier", "(Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;)I");
	jmethodID colour = (*env)->GetMethodID(env, class, "getColor", "(ILandroid/content/res/Resources$Theme;)I");
	(*env)->DeleteLocalRef(env, class);
	int found = !failed(env) && identifier != NULL && colour != NULL;
	jstring kind = (*env)->NewStringUTF(env, "color");
	jstring package = (*env)->NewStringUTF(env, "android");
	char name[48];
	for (int palette = 0; found && palette < palette_count; palette++) {
		for (int tone = 0; found && tone < tone_count; tone++) {
			snprintf(name, sizeof name, "system_%s_%d", palettes[palette], tones[tone]);
			jstring resource_name = (*env)->NewStringUTF(env, name);
			int id = (*env)->CallIntMethod(env, resources, identifier, resource_name, kind, package);
			(*env)->DeleteLocalRef(env, resource_name);
			if (failed(env) || id == 0) {
				found = 0;
				break;
			}
			colours[palette * tone_count + tone] = (unsigned int)(*env)->CallIntMethod(env, resources, colour, id, NULL);
			if (failed(env)) found = 0;
		}
	}
	(*env)->DeleteLocalRef(env, kind);
	(*env)->DeleteLocalRef(env, package);
	(*env)->DeleteLocalRef(env, resources);
	return found;
}
