//go:build android

// The file chooser in C: it finds FileChooser.java in the package, gives it the Go function it
// calls (chosen), and opens it when Go asks. A package without the class (made by an older tool)
// has no chooser: files_open says so, and Go's ChooseFiles returns false.

#include <android/log.h>
#include "_cgo_export.h"
#include "activity.h"

#define CHOOSER_CLASS "io.github.akrck02.gorenderer.FileChooser"

static jclass chooser_class;
static jmethodID open_method;

static int thrown(JNIEnv *env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
		return 1;
	}
	return 0;
}

// files_chosen is FileChooser.chosen: one chosen file, its name in UTF-16, as Java keeps it (JNI's
// own UTF-8 is not quite UTF-8), its size, and its bytes when Java read them.
static void JNICALL files_chosen(JNIEnv *env, jclass class, jstring name, jlong size, jbyteArray content) {
	const jchar *characters = name == NULL ? NULL : (*env)->GetStringChars(env, name, NULL);
	jsize name_length = characters == NULL ? 0 : (*env)->GetStringLength(env, name);
	jbyte *bytes = content == NULL ? NULL : (*env)->GetByteArrayElements(env, content, NULL);
	jsize content_length = bytes == NULL ? 0 : (*env)->GetArrayLength(env, content);
	goFileChosen((jchar *)characters, name_length, size, bytes, content_length, bytes != NULL);
	if (bytes != NULL) (*env)->ReleaseByteArrayElements(env, content, bytes, JNI_ABORT);
	if (characters != NULL) (*env)->ReleaseStringChars(env, name, characters);
}

// files_done is FileChooser.done: the end of the choice.
static void JNICALL files_done(JNIEnv *env, jclass class) { goFilesChosen(); }

// files_install finds the class and gives it its native method, the first time. Called on the
// interface thread whenever Android makes the activity: there the activity's class loader finds
// the package's classes.
void files_install(JNIEnv *env, jobject activity) {
	if (chooser_class != NULL) return;
	jclass found = load_package_class(env, activity, CHOOSER_CLASS);
	if (found == NULL) {
		__android_log_write(ANDROID_LOG_WARN, "Go", "no FileChooser class in the package: no file chooser");
		return;
	}
	JNINativeMethod natives[] = {
		{"chosen", "(Ljava/lang/String;J[B)V", (void *)files_chosen},
		{"done", "()V", (void *)files_done},
	};
	if ((*env)->RegisterNatives(env, found, natives, sizeof natives / sizeof natives[0]) != 0 || thrown(env)) {
		(*env)->DeleteLocalRef(env, found);
		return;
	}
	open_method = (*env)->GetStaticMethodID(env, found, "open", "(Landroid/app/Activity;J)Z");
	if (thrown(env) || open_method == NULL) {
		(*env)->DeleteLocalRef(env, found);
		return;
	}
	chooser_class = (jclass)(*env)->NewGlobalRef(env, found);
	(*env)->DeleteLocalRef(env, found);
}

int files_available(void) { return chooser_class != NULL; }

// files_open opens the chooser over the activity, for files of at most largest bytes; 0 when it
// could not. Safe from any thread, which it attaches to Java.
int files_open(jlong largest) {
	if (chooser_class == NULL) return 0;
	JNIEnv *env = attach_thread();
	if (env == NULL) return 0;
	jboolean opened = (*env)->CallStaticBooleanMethod(env, chooser_class, open_method, current_activity->clazz, largest);
	return !thrown(env) && opened;
}
