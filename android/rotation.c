//go:build android

// The rotation sensor: a thread of its own waits on Android's sensor queue and keeps the latest
// reading, which the program takes whenever it draws. The game rotation vector is preferred, as it
// leaves the compass out: the view does not jump when a magnet passes, and only the heading it
// starts with is arbitrary.

#include <android/sensor.h>
#include <pthread.h>
#include "activity.h"

// readingEvery is how often the sensor is asked to report, in microseconds: twice a frame of a
// phone's screen, so a frame never draws a reading much older than itself.
#define readingEvery 8000

static pthread_mutex_t lock = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t settled = PTHREAD_COND_INITIALIZER;
static pthread_t reader;
static ALooper *reader_looper;
static int running, starting, stopping, found, has_reading;
static float latest[4];

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
static ASensorManager *sensor_manager(void) { return ASensorManager_getInstance(); }
#pragma clang diagnostic pop

static void settle(ALooper *looper, int with_sensor) {
	pthread_mutex_lock(&lock);
	reader_looper = looper;
	found = with_sensor;
	starting = 0;
	pthread_cond_broadcast(&settled);
	pthread_mutex_unlock(&lock);
}

static void *read_rotation(void *unused) {
	(void)unused;
	ALooper *looper = ALooper_prepare(ALOOPER_PREPARE_ALLOW_NON_CALLBACKS);
	ASensorManager *manager = sensor_manager();
	const ASensor *sensor = ASensorManager_getDefaultSensor(manager, ASENSOR_TYPE_GAME_ROTATION_VECTOR);
	if (sensor == NULL) sensor = ASensorManager_getDefaultSensor(manager, ASENSOR_TYPE_ROTATION_VECTOR);
	ASensorEventQueue *queue = sensor == NULL ? NULL : ASensorManager_createEventQueue(manager, looper, 1, NULL, NULL);
	if (queue != NULL && ASensorEventQueue_registerSensor(queue, sensor, readingEvery, 0) != 0) {
		ASensorManager_destroyEventQueue(manager, queue);
		queue = NULL;
	}
	settle(looper, queue != NULL);
	if (queue == NULL) return NULL;
	for (;;) {
		pthread_mutex_lock(&lock);
		int stop = stopping;
		pthread_mutex_unlock(&lock);
		if (stop) break;
		ALooper_pollOnce(-1, NULL, NULL, NULL);
		ASensorEvent events[8];
		ssize_t count;
		while ((count = ASensorEventQueue_getEvents(queue, events, 8)) > 0) {
			const float *data = events[count - 1].data;
			pthread_mutex_lock(&lock);
			latest[0] = data[0];
			latest[1] = data[1];
			latest[2] = data[2];
			latest[3] = data[3]; // cos(θ/2), which Android gives since 4.3
			has_reading = 1;
			pthread_mutex_unlock(&lock);
		}
	}
	ASensorEventQueue_disableSensor(queue, sensor);
	ASensorManager_destroyEventQueue(manager, queue);
	return NULL;
}

int rotation_start(void) {
	pthread_mutex_lock(&lock);
	if (!running) {
		running = 1;
		starting = 1;
		stopping = 0;
		has_reading = 0;
		if (pthread_create(&reader, NULL, read_rotation, NULL) != 0) {
			running = 0;
			starting = 0;
			found = 0;
		}
	}
	while (starting) pthread_cond_wait(&settled, &lock);
	int with_sensor = found;
	pthread_mutex_unlock(&lock);
	return with_sensor;
}

void rotation_stop(void) {
	pthread_mutex_lock(&lock);
	if (!running) {
		pthread_mutex_unlock(&lock);
		return;
	}
	stopping = 1;
	if (reader_looper != NULL) ALooper_wake(reader_looper);
	pthread_mutex_unlock(&lock);
	pthread_join(reader, NULL);
	pthread_mutex_lock(&lock);
	running = 0;
	has_reading = 0;
	reader_looper = NULL;
	pthread_mutex_unlock(&lock);
}

int rotation_read(float *values) {
	pthread_mutex_lock(&lock);
	int read = has_reading;
	for (int index = 0; index < 4; index++) values[index] = latest[index];
	pthread_mutex_unlock(&lock);
	return read;
}
