//go:build android

// The phone's own emoji in C: it finds EmojiPicture.java in the package and calls it for an emoji
// no font file the overlay reads has (emoji.go). A package without the class (made by an older
// tool) draws those emoji as question marks, as before, and says so once in the log.

#include <android/log.h>
#include <stdlib.h>
#include <string.h>
#include "activity.h"

#define EMOJI_CLASS "io.github.akrck02.gorenderer.EmojiPicture"

static jclass emoji_class;
static jmethodID draw_method;

static int thrown(JNIEnv *env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
		return 1;
	}
	return 0;
}

// emoji_install finds the class, the first time. Called on the interface thread whenever Android
// makes the activity: there the activity's class loader finds the package's classes.
void emoji_install(JNIEnv *env, jobject activity) {
	if (emoji_class != NULL) return;
	jclass found = load_package_class(env, activity, EMOJI_CLASS);
	if (found == NULL) {
		__android_log_write(ANDROID_LOG_WARN, "Go", "no EmojiPicture class in the package: emoji only from font files");
		return;
	}
	draw_method = (*env)->GetStaticMethodID(env, found, "draw", "(II)[I");
	if (thrown(env) || draw_method == NULL) {
		(*env)->DeleteLocalRef(env, found);
		return;
	}
	emoji_class = (jclass)(*env)->NewGlobalRef(env, found);
	(*env)->DeleteLocalRef(env, found);
}

int emoji_available(void) { return emoji_class != NULL; }

// emoji_draw asks Java to draw a character at a size in pixels, and returns a copy of its answer
// (EmojiPicture.draw's header and pixels), for the caller to free, with its length; NULL when the
// phone has no colour emoji of it, or Java failed. Safe from any thread, which it attaches to Java.
jint *emoji_draw(int character, int pixels, int *length) {
	*length = 0;
	if (emoji_class == NULL) return NULL;
	JNIEnv *env = attach_thread();
	if (env == NULL) return NULL;
	jintArray answer = (jintArray)(*env)->CallStaticObjectMethod(env, emoji_class, draw_method, character, pixels);
	if (thrown(env) || answer == NULL) return NULL;
	jsize count = (*env)->GetArrayLength(env, answer);
	jint *copy = malloc(sizeof(jint) * (count > 0 ? count : 1));
	if (copy != NULL) {
		(*env)->GetIntArrayRegion(env, answer, 0, count, copy);
		if (thrown(env)) {
			free(copy);
			copy = NULL;
		} else {
			*length = count;
		}
	}
	(*env)->DeleteLocalRef(env, answer);
	return copy;
}
