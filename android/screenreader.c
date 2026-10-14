//go:build android

// The screen reader's side in C: it finds ScreenReader.java in the package, gives it the Go
// functions it calls (acted, focused, readerChanged) and calls it with the frame's nodes and with
// the finger exploring the screen. A package without the class (made by an older tool) runs without
// a screen reader, and says so once in the log.

#include <android/log.h>
#include "_cgo_export.h"
#include "activity.h"

#define READER_CLASS "io.github.akrck02.gorenderer.ScreenReader"

static jclass reader_class;
static jmethodID install_method, update_method, hover_method;

static int thrown(JNIEnv *env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
		return 1;
	}
	return 0;
}

static void JNICALL reader_acted(JNIEnv *env, jclass class, jint node, jint action) { goReaderActed(node, action); }
static void JNICALL reader_focused(JNIEnv *env, jclass class, jint node) { goReaderFocused(node); }
static void JNICALL reader_changed(JNIEnv *env, jclass class, jboolean on) { goReaderChanged(on ? 1 : 0); }

// load_reader_class loads the class through the activity's class loader: on the interface thread,
// FindClass would look among Android's own classes, where called from native code.
static jclass load_reader_class(JNIEnv *env, jobject activity) {
	jclass activity_class = (*env)->GetObjectClass(env, activity);
	jmethodID get_loader = (*env)->GetMethodID(env, activity_class, "getClassLoader", "()Ljava/lang/ClassLoader;");
	(*env)->DeleteLocalRef(env, activity_class);
	if (thrown(env) || get_loader == NULL) return NULL;
	jobject loader = (*env)->CallObjectMethod(env, activity, get_loader);
	if (thrown(env) || loader == NULL) return NULL;
	jclass loader_class = (*env)->GetObjectClass(env, loader);
	jmethodID load = (*env)->GetMethodID(env, loader_class, "loadClass", "(Ljava/lang/String;)Ljava/lang/Class;");
	(*env)->DeleteLocalRef(env, loader_class);
	jclass found = NULL;
	if (!thrown(env) && load != NULL) {
		jstring name = (*env)->NewStringUTF(env, READER_CLASS);
		found = (jclass)(*env)->CallObjectMethod(env, loader, load, name);
		(*env)->DeleteLocalRef(env, name);
		if (thrown(env)) found = NULL;
	}
	(*env)->DeleteLocalRef(env, loader);
	return found;
}

// prepare_reader_class finds the class and gives it its native methods, the first time.
static int prepare_reader_class(JNIEnv *env, jobject activity) {
	if (reader_class != NULL) return 1;
	jclass found = load_reader_class(env, activity);
	if (found == NULL) {
		__android_log_write(ANDROID_LOG_WARN, "Go", "no ScreenReader class in the package: no screen reader");
		return 0;
	}
	JNINativeMethod natives[] = {
		{"acted", "(II)V", (void *)reader_acted},
		{"focused", "(I)V", (void *)reader_focused},
		{"readerChanged", "(Z)V", (void *)reader_changed},
	};
	if ((*env)->RegisterNatives(env, found, natives, 3) != 0 || thrown(env)) {
		(*env)->DeleteLocalRef(env, found);
		return 0;
	}
	install_method = (*env)->GetStaticMethodID(env, found, "install", "(Landroid/app/Activity;)V");
	update_method = (*env)->GetStaticMethodID(env, found, "update",
		"([I[I[I[F[F[Ljava/lang/String;[Ljava/lang/String;[Ljava/lang/String;)V");
	hover_method = (*env)->GetStaticMethodID(env, found, "hover", "(IFF)V");
	if (thrown(env) || install_method == NULL || update_method == NULL || hover_method == NULL) {
		(*env)->DeleteLocalRef(env, found);
		return 0;
	}
	reader_class = (jclass)(*env)->NewGlobalRef(env, found);
	(*env)->DeleteLocalRef(env, found);
	return 1;
}

// screen_reader_install puts the screen reader's provider on the activity's view. Called on the
// interface thread whenever Android makes the activity, a rotation included.
void screen_reader_install(JNIEnv *env, jobject activity) {
	if (!prepare_reader_class(env, activity)) return;
	(*env)->CallStaticVoidMethod(env, reader_class, install_method, activity);
	thrown(env);
}

static jintArray int_array(JNIEnv *env, int count, const int *values) {
	jintArray array = (*env)->NewIntArray(env, count);
	if (array != NULL) (*env)->SetIntArrayRegion(env, array, 0, count, (const jint *)values);
	return array;
}

static jfloatArray float_array(JNIEnv *env, int count, const float *values) {
	jfloatArray array = (*env)->NewFloatArray(env, count);
	if (array != NULL) (*env)->SetFloatArrayRegion(env, array, 0, count, values);
	return array;
}

// string_array makes the strings of one kind (class, label or value: which, 0 to 2) of every node,
// from UTF-16 characters at offsets.
static jobjectArray string_array(JNIEnv *env, int count, int which, const jchar *characters, const int *offsets) {
	jclass string_class = (*env)->FindClass(env, "java/lang/String");
	if (thrown(env) || string_class == NULL) return NULL;
	jobjectArray array = (*env)->NewObjectArray(env, count, string_class, NULL);
	(*env)->DeleteLocalRef(env, string_class);
	if (thrown(env) || array == NULL) return NULL;
	for (int node = 0; node < count; node++) {
		int start = offsets[node * 3 + which], end = offsets[node * 3 + which + 1];
		jstring text = (*env)->NewString(env, characters + start, end - start);
		if (thrown(env) || text == NULL) return array;
		(*env)->SetObjectArrayElement(env, array, node, text);
		(*env)->DeleteLocalRef(env, text); // a long list would fill the table of local references
	}
	return array;
}

static void delete_all(JNIEnv *env, jobject *references, int count) {
	for (int index = 0; index < count; index++) {
		if (references[index] != NULL) (*env)->DeleteLocalRef(env, references[index]);
	}
}

// screen_reader_update hands the frame's nodes to Java. Safe from any thread.
void screen_reader_update(int count, const int *ids, const int *parents, const int *flags, const float *bounds,
		const float *ranges, const jchar *characters, const int *offsets) {
	if (reader_class == NULL) return;
	JNIEnv *env = attach_thread();
	if (env == NULL) return;
	jobject arrays[8] = {
		int_array(env, count, ids), int_array(env, count, parents), int_array(env, count, flags),
		float_array(env, count * 4, bounds), float_array(env, count * 3, ranges),
		string_array(env, count, 0, characters, offsets), string_array(env, count, 1, characters, offsets),
		string_array(env, count, 2, characters, offsets),
	};
	int made = !thrown(env);
	for (int index = 0; index < 8; index++) made = made && arrays[index] != NULL;
	if (made) {
		(*env)->CallStaticVoidMethod(env, reader_class, update_method, arrays[0], arrays[1], arrays[2], arrays[3],
			arrays[4], arrays[5], arrays[6], arrays[7]);
		thrown(env);
	}
	delete_all(env, arrays, 8);
}

// screen_reader_hover passes on a finger exploring the screen: a hover event's action, and where,
// in window pixels. Safe from any thread.
void screen_reader_hover(int action, float x, float y) {
	if (reader_class == NULL) return;
	JNIEnv *env = attach_thread();
	if (env == NULL) return;
	(*env)->CallStaticVoidMethod(env, reader_class, hover_method, action, x, y);
	thrown(env);
}
