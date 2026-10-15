//go:build android

// The on-screen keyboard's side in C: it finds TextInput.java in the package, gives it the Go
// function it calls (edited), and passes on Go's calls: show or hide the keyboard, and the field's
// text as Go has it. A package without the class (made by an older tool) types key by key, as
// before, through soft_keyboard.

#include "_cgo_export.h"
#include "activity.h"

#define TEXT_INPUT_CLASS "io.github.akrck02.gorenderer.TextInput"

static jclass input_class;
static jmethodID install_method, show_method, set_method;

static int thrown(JNIEnv *env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
		return 1;
	}
	return 0;
}

// input_edited passes on the field's state after the keyboard edited it, its text as Java keeps it,
// UTF-16: JNI's own UTF-8 is not quite UTF-8, and an emoji in it stops the program.
static void JNICALL input_edited(JNIEnv *env, jclass class, jlong field, jint edit, jstring text, jint selection_start,
		jint selection_end, jint composing_start, jint composing_end) {
	jsize length = text == NULL ? 0 : (*env)->GetStringLength(env, text);
	const jchar *characters = text == NULL ? NULL : (*env)->GetStringChars(env, text, NULL);
	if (text != NULL && characters == NULL) return;
	goTextEdited(field, edit, (jchar *)characters, length, selection_start, selection_end, composing_start, composing_end);
	if (characters != NULL) (*env)->ReleaseStringChars(env, text, characters);
}

// prepare_input_class finds the class and gives it its native method, the first time.
static int prepare_input_class(JNIEnv *env, jobject activity) {
	if (input_class != NULL) return 1;
	jclass found = load_package_class(env, activity, TEXT_INPUT_CLASS);
	if (found == NULL) return 0;
	JNINativeMethod natives[] = {
		{"edited", "(JILjava/lang/String;IIII)V", (void *)input_edited},
	};
	if ((*env)->RegisterNatives(env, found, natives, sizeof natives / sizeof natives[0]) != 0 || thrown(env)) {
		(*env)->DeleteLocalRef(env, found);
		return 0;
	}
	install_method = (*env)->GetStaticMethodID(env, found, "install", "(Landroid/app/Activity;)V");
	show_method = (*env)->GetStaticMethodID(env, found, "show", "(Z)V");
	set_method = (*env)->GetStaticMethodID(env, found, "set", "(JILjava/lang/String;IIIIZ)V");
	if (thrown(env) || install_method == NULL || show_method == NULL || set_method == NULL) {
		(*env)->DeleteLocalRef(env, found);
		return 0;
	}
	input_class = (jclass)(*env)->NewGlobalRef(env, found);
	(*env)->DeleteLocalRef(env, found);
	return 1;
}

// text_input_install puts the view on the activity. Called on the interface thread whenever
// Android makes the activity, a rotation included.
int text_input_install(JNIEnv *env, jobject activity) {
	if (!prepare_input_class(env, activity)) return 0;
	(*env)->CallStaticVoidMethod(env, input_class, install_method, activity);
	return !thrown(env);
}

// text_input_show brings the keyboard up for the view, or puts it away; 0 when the package has no
// view for it, and soft_keyboard is the way.
int text_input_show(int typing) {
	if (input_class == NULL) return 0;
	JNIEnv *env = attach_thread();
	if (env == NULL) return 0;
	(*env)->CallStaticVoidMethod(env, input_class, show_method, typing ? JNI_TRUE : JNI_FALSE);
	return !thrown(env);
}

// text_input_set gives the view a field's text as Go has it; indices are in UTF-16 units.
void text_input_set(jlong field, int edit, const jchar *characters, int length, int selection_start,
		int selection_end, int composing_start, int composing_end, int secret) {
	if (input_class == NULL) return;
	JNIEnv *env = attach_thread();
	if (env == NULL) return;
	jstring text = (*env)->NewString(env, characters, length);
	if (thrown(env) || text == NULL) return;
	(*env)->CallStaticVoidMethod(env, input_class, set_method, field, edit, text, selection_start, selection_end,
		composing_start, composing_end, secret ? JNI_TRUE : JNI_FALSE);
	thrown(env);
	(*env)->DeleteLocalRef(env, text);
}
