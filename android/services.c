//go:build android

// What a program asks of the phone through JNI, beyond its window: the person's text size, the
// clipboard, opening a link in the application that shows it, and sharing a text through the
// system's sheet. Every call gives up quietly on a Java exception, and says so with its result.

#include <stdlib.h>
#include <string.h>
#include "activity.h"

static int cleared(JNIEnv *env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
		return 0;
	}
	return 1;
}

static jobject object_call(JNIEnv *env, jobject object, const char *name, const char *signature) {
	if (object == NULL) return NULL;
	jclass class = (*env)->GetObjectClass(env, object);
	jmethodID method = (*env)->GetMethodID(env, class, name, signature);
	(*env)->DeleteLocalRef(env, class);
	if (!cleared(env) || method == NULL) return NULL;
	jobject result = (*env)->CallObjectMethod(env, object, method);
	return cleared(env) ? result : NULL;
}

// font_scale is the person's text size (Settings › Display › Font size), 1 for the default.
float font_scale(JNIEnv *env) {
	jobject resources = object_call(env, current_activity->clazz, "getResources", "()Landroid/content/res/Resources;");
	jobject configuration = object_call(env, resources, "getConfiguration", "()Landroid/content/res/Configuration;");
	float scale = 1;
	if (configuration != NULL) {
		jclass class = (*env)->GetObjectClass(env, configuration);
		jfieldID field = (*env)->GetFieldID(env, class, "fontScale", "F");
		(*env)->DeleteLocalRef(env, class);
		if (cleared(env) && field != NULL) {
			scale = (*env)->GetFloatField(env, configuration, field);
		}
		(*env)->DeleteLocalRef(env, configuration);
	}
	if (resources != NULL) (*env)->DeleteLocalRef(env, resources);
	return scale > 0 ? scale : 1;
}

// clipboard_manager is the activity's ClipboardManager (a local reference to delete), or NULL.
static jobject clipboard_manager(JNIEnv *env) {
	jobject activity = current_activity->clazz;
	jclass class = (*env)->GetObjectClass(env, activity);
	jmethodID service = (*env)->GetMethodID(env, class, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;");
	(*env)->DeleteLocalRef(env, class);
	if (!cleared(env) || service == NULL) return NULL;
	jstring name = (*env)->NewStringUTF(env, "clipboard");
	jobject manager = (*env)->CallObjectMethod(env, activity, service, name);
	(*env)->DeleteLocalRef(env, name);
	return cleared(env) ? manager : NULL;
}

// clipboard_text is the clipboard's text, as UTF-8 to free, or NULL when it holds none.
char *clipboard_text(void) {
	JNIEnv *env = attach_thread();
	if (env == NULL) return NULL;
	jobject manager = clipboard_manager(env);
	jobject clip = object_call(env, manager, "getPrimaryClip", "()Landroid/content/ClipData;");
	char *text = NULL;
	if (clip != NULL) {
		jclass class = (*env)->GetObjectClass(env, clip);
		jmethodID item_at = (*env)->GetMethodID(env, class, "getItemAt", "(I)Landroid/content/ClipData$Item;");
		(*env)->DeleteLocalRef(env, class);
		jobject item = cleared(env) && item_at != NULL ? (*env)->CallObjectMethod(env, clip, item_at, 0) : NULL;
		if (cleared(env) && item != NULL) {
			jclass item_class = (*env)->GetObjectClass(env, item);
			jmethodID coerce = (*env)->GetMethodID(env, item_class, "coerceToText", "(Landroid/content/Context;)Ljava/lang/CharSequence;");
			(*env)->DeleteLocalRef(env, item_class);
			jobject sequence = cleared(env) && coerce != NULL ? (*env)->CallObjectMethod(env, item, coerce, current_activity->clazz) : NULL;
			jstring string = (jstring)object_call(env, cleared(env) ? sequence : NULL, "toString", "()Ljava/lang/String;");
			if (string != NULL) {
				const char *characters = (*env)->GetStringUTFChars(env, string, NULL);
				if (characters != NULL) {
					text = strdup(characters);
					(*env)->ReleaseStringUTFChars(env, string, characters);
				}
				(*env)->DeleteLocalRef(env, string);
			}
			if (sequence != NULL) (*env)->DeleteLocalRef(env, sequence);
			(*env)->DeleteLocalRef(env, item);
		}
		(*env)->DeleteLocalRef(env, clip);
	}
	if (manager != NULL) (*env)->DeleteLocalRef(env, manager);
	return text;
}

// set_clipboard_text puts a text on the clipboard; 0 when it could not.
int set_clipboard_text(const char *text) {
	JNIEnv *env = attach_thread();
	if (env == NULL) return 0;
	jobject manager = clipboard_manager(env);
	if (manager == NULL) return 0;
	int done = 0;
	jclass data = (*env)->FindClass(env, "android/content/ClipData");
	jmethodID plain = cleared(env) && data != NULL ? (*env)->GetStaticMethodID(env, data, "newPlainText",
		"(Ljava/lang/CharSequence;Ljava/lang/CharSequence;)Landroid/content/ClipData;") : NULL;
	if (cleared(env) && plain != NULL) {
		jstring label = (*env)->NewStringUTF(env, "text");
		jstring value = (*env)->NewStringUTF(env, text);
		jobject clip = (*env)->CallStaticObjectMethod(env, data, plain, label, value);
		if (cleared(env) && clip != NULL) {
			jclass class = (*env)->GetObjectClass(env, manager);
			jmethodID set = (*env)->GetMethodID(env, class, "setPrimaryClip", "(Landroid/content/ClipData;)V");
			(*env)->DeleteLocalRef(env, class);
			if (cleared(env) && set != NULL) {
				(*env)->CallVoidMethod(env, manager, set, clip);
				done = cleared(env);
			}
			(*env)->DeleteLocalRef(env, clip);
		}
		(*env)->DeleteLocalRef(env, label);
		(*env)->DeleteLocalRef(env, value);
	}
	if (data != NULL) (*env)->DeleteLocalRef(env, data);
	(*env)->DeleteLocalRef(env, manager);
	return done;
}

// new_intent makes an Intent for an action (a local reference to delete), or NULL.
static jobject new_intent(JNIEnv *env, const char *action) {
	jclass class = (*env)->FindClass(env, "android/content/Intent");
	if (!cleared(env) || class == NULL) return NULL;
	jmethodID make = (*env)->GetMethodID(env, class, "<init>", "(Ljava/lang/String;)V");
	jobject intent = NULL;
	if (cleared(env) && make != NULL) {
		jstring name = (*env)->NewStringUTF(env, action);
		intent = (*env)->NewObject(env, class, make, name);
		(*env)->DeleteLocalRef(env, name);
		if (!cleared(env)) intent = NULL;
	}
	(*env)->DeleteLocalRef(env, class);
	return intent;
}

// start_activity starts what an intent names from the program's activity; 0 when nothing can
// (no application opens such a link).
static int start_activity(JNIEnv *env, jobject intent) {
	jobject activity = current_activity->clazz;
	jclass class = (*env)->GetObjectClass(env, activity);
	jmethodID start = (*env)->GetMethodID(env, class, "startActivity", "(Landroid/content/Intent;)V");
	(*env)->DeleteLocalRef(env, class);
	if (!cleared(env) || start == NULL) return 0;
	(*env)->CallVoidMethod(env, activity, start, intent);
	return cleared(env);
}

// open_link opens a link (a web page, mailto:, tel:, geo:) in the application that shows it.
int open_link(const char *link) {
	JNIEnv *env = attach_thread();
	if (env == NULL) return 0;
	jobject intent = new_intent(env, "android.intent.action.VIEW");
	if (intent == NULL) return 0;
	int done = 0;
	jclass uri = (*env)->FindClass(env, "android/net/Uri");
	jmethodID parse = cleared(env) && uri != NULL ? (*env)->GetStaticMethodID(env, uri, "parse", "(Ljava/lang/String;)Landroid/net/Uri;") : NULL;
	if (cleared(env) && parse != NULL) {
		jstring text = (*env)->NewStringUTF(env, link);
		jobject parsed = (*env)->CallStaticObjectMethod(env, uri, parse, text);
		(*env)->DeleteLocalRef(env, text);
		jclass class = (*env)->GetObjectClass(env, intent);
		jmethodID set_data = (*env)->GetMethodID(env, class, "setData", "(Landroid/net/Uri;)Landroid/content/Intent;");
		(*env)->DeleteLocalRef(env, class);
		if (cleared(env) && parsed != NULL && set_data != NULL) {
			jobject same = (*env)->CallObjectMethod(env, intent, set_data, parsed);
			if (same != NULL) (*env)->DeleteLocalRef(env, same);
			done = cleared(env) && start_activity(env, intent);
		}
		if (parsed != NULL) (*env)->DeleteLocalRef(env, parsed);
	}
	if (uri != NULL) (*env)->DeleteLocalRef(env, uri);
	(*env)->DeleteLocalRef(env, intent);
	return done;
}

// share_text offers a text to the person's other applications, through the system's sheet with a
// title over it.
int share_text(const char *title, const char *text) {
	JNIEnv *env = attach_thread();
	if (env == NULL) return 0;
	jobject intent = new_intent(env, "android.intent.action.SEND");
	if (intent == NULL) return 0;
	int done = 0;
	jclass class = (*env)->GetObjectClass(env, intent);
	jmethodID set_type = (*env)->GetMethodID(env, class, "setType", "(Ljava/lang/String;)Landroid/content/Intent;");
	jmethodID put = (*env)->GetMethodID(env, class, "putExtra", "(Ljava/lang/String;Ljava/lang/String;)Landroid/content/Intent;");
	jmethodID chooser = (*env)->GetStaticMethodID(env, class, "createChooser",
		"(Landroid/content/Intent;Ljava/lang/CharSequence;)Landroid/content/Intent;");
	if (cleared(env) && set_type != NULL && put != NULL && chooser != NULL) {
		jstring type = (*env)->NewStringUTF(env, "text/plain");
		jstring key = (*env)->NewStringUTF(env, "android.intent.extra.TEXT");
		jstring value = (*env)->NewStringUTF(env, text);
		jstring heading = (*env)->NewStringUTF(env, title);
		jobject same = (*env)->CallObjectMethod(env, intent, set_type, type);
		if (same != NULL) (*env)->DeleteLocalRef(env, same);
		same = (*env)->CallObjectMethod(env, intent, put, key, value);
		if (same != NULL) (*env)->DeleteLocalRef(env, same);
		jobject sheet = (*env)->CallStaticObjectMethod(env, class, chooser, intent, heading);
		if (cleared(env) && sheet != NULL) {
			done = start_activity(env, sheet);
			(*env)->DeleteLocalRef(env, sheet);
		}
		(*env)->DeleteLocalRef(env, type);
		(*env)->DeleteLocalRef(env, key);
		(*env)->DeleteLocalRef(env, value);
		(*env)->DeleteLocalRef(env, heading);
	}
	(*env)->DeleteLocalRef(env, class);
	(*env)->DeleteLocalRef(env, intent);
	return done;
}
