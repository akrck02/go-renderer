//go:build android

// Notifications through JNI: Android's own Notification.Builder on a channel of the program's, and
// the permission Android 13 and later ask the person for. Every call gives up quietly on a Java
// exception, and says so with its result.

#include <stdarg.h>
#include <stddef.h>
#include "activity.h"

static int cleared(JNIEnv *env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
		return 0;
	}
	return 1;
}

// method finds a method of an object's class, or NULL.
static jmethodID method(JNIEnv *env, jobject object, const char *name, const char *signature) {
	jclass class = (*env)->GetObjectClass(env, object);
	jmethodID found = (*env)->GetMethodID(env, class, name, signature);
	(*env)->DeleteLocalRef(env, class);
	return cleared(env) ? found : NULL;
}

// call_setter calls a Notification.Builder setter, which returns the builder, and drops the ref.
static int call_setter(JNIEnv *env, jobject builder, const char *name, const char *signature, ...) {
	jmethodID setter = method(env, builder, name, signature);
	if (setter == NULL) return 0;
	va_list arguments;
	va_start(arguments, signature);
	jobject same = (*env)->CallObjectMethodV(env, builder, setter, arguments);
	va_end(arguments);
	if (same != NULL) (*env)->DeleteLocalRef(env, same);
	return cleared(env);
}

static int sdk_level(JNIEnv *env) {
	jclass version = (*env)->FindClass(env, "android/os/Build$VERSION");
	if (!cleared(env) || version == NULL) return 0;
	jfieldID field = (*env)->GetStaticFieldID(env, version, "SDK_INT", "I");
	int level = cleared(env) && field != NULL ? (*env)->GetStaticIntField(env, version, field) : 0;
	(*env)->DeleteLocalRef(env, version);
	return level;
}

static const char *permission_name = "android.permission.POST_NOTIFICATIONS";

// notifications_allowed tells whether notifications may be posted: always before Android 13, and
// after the person allowed them since.
int notifications_allowed(void) {
	JNIEnv *env = attach_thread();
	if (env == NULL) return 0;
	if (sdk_level(env) < 33) return 1;
	jobject activity = current_activity->clazz;
	jmethodID check = method(env, activity, "checkSelfPermission", "(Ljava/lang/String;)I");
	if (check == NULL) return 0;
	jstring name = (*env)->NewStringUTF(env, permission_name);
	int result = (*env)->CallIntMethod(env, activity, check, name);
	(*env)->DeleteLocalRef(env, name);
	return cleared(env) && result == 0; // PackageManager.PERMISSION_GRANTED
}

// ask_for_notifications shows Android's question to allow notifications (Android 13 and later),
// once the manifest names the permission. The answer is read later with notifications_allowed.
void ask_for_notifications(void) {
	JNIEnv *env = attach_thread();
	if (env == NULL || sdk_level(env) < 33) return;
	jobject activity = current_activity->clazz;
	jmethodID request = method(env, activity, "requestPermissions", "([Ljava/lang/String;I)V");
	jclass string = (*env)->FindClass(env, "java/lang/String");
	if (request == NULL || !cleared(env) || string == NULL) return;
	jstring name = (*env)->NewStringUTF(env, permission_name);
	jobjectArray names = (*env)->NewObjectArray(env, 1, string, name);
	(*env)->CallVoidMethod(env, activity, request, names, 1);
	cleared(env);
	(*env)->DeleteLocalRef(env, names);
	(*env)->DeleteLocalRef(env, name);
	(*env)->DeleteLocalRef(env, string);
}

// open_application is what touching the notification does: open the program, as its icon does.
static jobject open_application(JNIEnv *env, jobject context) {
	jmethodID packages = method(env, context, "getPackageManager", "()Landroid/content/pm/PackageManager;");
	jmethodID own_name = method(env, context, "getPackageName", "()Ljava/lang/String;");
	if (packages == NULL || own_name == NULL) return NULL;
	jobject manager = (*env)->CallObjectMethod(env, context, packages);
	jobject name = (*env)->CallObjectMethod(env, context, own_name);
	jobject result = NULL;
	if (cleared(env) && manager != NULL && name != NULL) {
		jmethodID launch = method(env, manager, "getLaunchIntentForPackage", "(Ljava/lang/String;)Landroid/content/Intent;");
		jobject intent = launch ? (*env)->CallObjectMethod(env, manager, launch, name) : NULL;
		jclass pending = (*env)->FindClass(env, "android/app/PendingIntent");
		if (cleared(env) && intent != NULL && pending != NULL) {
			jmethodID activity = (*env)->GetStaticMethodID(env, pending, "getActivity",
				"(Landroid/content/Context;ILandroid/content/Intent;I)Landroid/app/PendingIntent;");
			const int immutable_and_update = 0x04000000 | 0x08000000; // FLAG_IMMUTABLE | FLAG_UPDATE_CURRENT
			if (cleared(env) && activity != NULL) {
				result = (*env)->CallStaticObjectMethod(env, pending, activity, context, 0, intent, immutable_and_update);
				if (!cleared(env)) result = NULL;
			}
		}
		if (pending != NULL) (*env)->DeleteLocalRef(env, pending);
		if (intent != NULL) (*env)->DeleteLocalRef(env, intent);
	}
	if (manager != NULL) (*env)->DeleteLocalRef(env, manager);
	if (name != NULL) (*env)->DeleteLocalRef(env, name);
	return result;
}

// small_icon is the status bar's icon: Android's own information icon, which reads on any bar.
static int small_icon(JNIEnv *env, jobject context) {
	jmethodID resources_of = method(env, context, "getResources", "()Landroid/content/res/Resources;");
	jobject resources = resources_of ? (*env)->CallObjectMethod(env, context, resources_of) : NULL;
	if (!cleared(env) || resources == NULL) return 0;
	jmethodID identifier = method(env, resources, "getIdentifier", "(Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;)I");
	int icon = 0;
	if (identifier != NULL) {
		jstring name = (*env)->NewStringUTF(env, "ic_dialog_info");
		jstring kind = (*env)->NewStringUTF(env, "drawable");
		jstring package = (*env)->NewStringUTF(env, "android");
		icon = (*env)->CallIntMethod(env, resources, identifier, name, kind, package);
		if (!cleared(env)) icon = 0;
		(*env)->DeleteLocalRef(env, name);
		(*env)->DeleteLocalRef(env, kind);
		(*env)->DeleteLocalRef(env, package);
	}
	(*env)->DeleteLocalRef(env, resources);
	return icon;
}

// make_channel makes the program's channel, which Android 8 and later need and show in the
// application's settings by its name. Making it again changes nothing.
static int make_channel(JNIEnv *env, jobject manager, jstring channel, jstring channel_name) {
	jclass class = (*env)->FindClass(env, "android/app/NotificationChannel");
	if (!cleared(env) || class == NULL) return 0;
	jmethodID make = (*env)->GetMethodID(env, class, "<init>", "(Ljava/lang/String;Ljava/lang/CharSequence;I)V");
	int made = 0;
	if (cleared(env) && make != NULL) {
		jobject made_channel = (*env)->NewObject(env, class, make, channel, channel_name, 3); // IMPORTANCE_DEFAULT
		jmethodID create = method(env, manager, "createNotificationChannel", "(Landroid/app/NotificationChannel;)V");
		if (cleared(env) && made_channel != NULL && create != NULL) {
			(*env)->CallVoidMethod(env, manager, create, made_channel);
			made = cleared(env);
		}
		if (made_channel != NULL) (*env)->DeleteLocalRef(env, made_channel);
	}
	(*env)->DeleteLocalRef(env, class);
	return made;
}

// post_notification shows a notification with a title and a text; the same id replaces the one
// shown before. It returns 0 when it could not.
int post_notification(int id, const char *channel_id, const char *channel_label, const char *title, const char *text) {
	JNIEnv *env = attach_thread();
	if (env == NULL) return 0;
	jobject context = current_activity->clazz;
	jmethodID service = method(env, context, "getSystemService", "(Ljava/lang/String;)Ljava/lang/Object;");
	if (service == NULL) return 0;
	jstring service_name = (*env)->NewStringUTF(env, "notification");
	jobject manager = (*env)->CallObjectMethod(env, context, service, service_name);
	(*env)->DeleteLocalRef(env, service_name);
	if (!cleared(env) || manager == NULL) return 0;
	jstring channel = (*env)->NewStringUTF(env, channel_id);
	jstring label = (*env)->NewStringUTF(env, channel_label);
	jstring heading = (*env)->NewStringUTF(env, title);
	jstring body = (*env)->NewStringUTF(env, text);
	int posted = 0;
	jclass builder_class = (*env)->FindClass(env, "android/app/Notification$Builder");
	jmethodID make = builder_class ? (*env)->GetMethodID(env, builder_class, "<init>", "(Landroid/content/Context;Ljava/lang/String;)V") : NULL;
	if (cleared(env) && make != NULL && make_channel(env, manager, channel, label)) {
		jobject builder = (*env)->NewObject(env, builder_class, make, context, channel);
		jobject opening = open_application(env, context);
		int icon = small_icon(env, context);
		if (cleared(env) && builder != NULL && icon != 0 &&
			call_setter(env, builder, "setSmallIcon", "(I)Landroid/app/Notification$Builder;", icon) &&
			call_setter(env, builder, "setContentTitle", "(Ljava/lang/CharSequence;)Landroid/app/Notification$Builder;", heading) &&
			call_setter(env, builder, "setContentText", "(Ljava/lang/CharSequence;)Landroid/app/Notification$Builder;", body) &&
			call_setter(env, builder, "setAutoCancel", "(Z)Landroid/app/Notification$Builder;", JNI_TRUE) &&
			(opening == NULL || call_setter(env, builder, "setContentIntent", "(Landroid/app/PendingIntent;)Landroid/app/Notification$Builder;", opening))) {
			jmethodID build = method(env, builder, "build", "()Landroid/app/Notification;");
			jobject notification = build ? (*env)->CallObjectMethod(env, builder, build) : NULL;
			jmethodID show = method(env, manager, "notify", "(ILandroid/app/Notification;)V");
			if (cleared(env) && notification != NULL && show != NULL) {
				(*env)->CallVoidMethod(env, manager, show, id, notification);
				posted = cleared(env);
			}
			if (notification != NULL) (*env)->DeleteLocalRef(env, notification);
		}
		if (opening != NULL) (*env)->DeleteLocalRef(env, opening);
		if (builder != NULL) (*env)->DeleteLocalRef(env, builder);
	}
	if (builder_class != NULL) (*env)->DeleteLocalRef(env, builder_class);
	(*env)->DeleteLocalRef(env, channel);
	(*env)->DeleteLocalRef(env, label);
	(*env)->DeleteLocalRef(env, heading);
	(*env)->DeleteLocalRef(env, body);
	(*env)->DeleteLocalRef(env, manager);
	return posted;
}
