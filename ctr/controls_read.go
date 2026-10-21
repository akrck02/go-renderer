//go:build nintendo3ds

package ctr

/*
#include <3ds.h>

static void readControls(u32* held, int* touchX, int* touchY, int* circleX, int* circleY) {
	hidScanInput();
	*held = hidKeysHeld();
	touchPosition touch;
	hidTouchRead(&touch);
	*touchX = touch.px;
	*touchY = touch.py;
	circlePosition circle;
	hidCircleRead(&circle);
	*circleX = circle.dx;
	*circleY = circle.dy;
}

static int startCStick(void) {
	return R_SUCCEEDED(irrstInit());
}

static void stopCStick(void) {
	irrstExit();
}

static void readCStick(int* x, int* y) {
	irrstScanInput();
	circlePosition position;
	irrstCstickRead(&position);
	*x = position.dx;
	*y = position.dy;
}

static float startGyroscope(void) {
	float stepsPerDegree = 0;
	if (R_FAILED(HIDUSER_EnableGyroscope())) return 0;
	if (R_FAILED(HIDUSER_GetGyroscopeRawToDpsCoefficient(&stepsPerDegree))) {
		HIDUSER_DisableGyroscope(); // without its coefficient it cannot be read: stop it using power
		return 0;
	}
	return stepsPerDegree;
}

static void stopGyroscope(void) {
	HIDUSER_DisableGyroscope();
}

static void readGyroscope(int* roll, int* pitch, int* yaw) {
	angularRate rate;
	hidGyroRead(&rate);
	*roll = rate.x;
	*pitch = rate.y;
	*yaw = rate.z;
}
*/
import "C"

// The C-stick (the New 3DS's) and the gyroscope are services of their own, started only when a
// program asks: the gyroscope uses power, and the C-stick's service is not on older consoles.
// (After Go for 3DS's app package, the same author's, MIT.)
var motion struct {
	cStick         bool
	stepsPerDegree float64 // the gyroscope's raw steps in a degree a second; 0 while it is off
}

// StartMotionControls starts the C-stick and the gyroscope, and tells which started: a console
// without one, or an emulator, says no. ReadControls reads them from then on.
func StartMotionControls() (cStick, gyroscope bool) {
	motion.cStick = C.startCStick() != 0
	motion.stepsPerDegree = float64(C.startGyroscope())
	return motion.cStick, motion.stepsPerDegree != 0
}

// StopMotionControls stops what StartMotionControls started.
func StopMotionControls() {
	if motion.cStick {
		C.stopCStick()
	}
	if motion.stepsPerDegree != 0 {
		C.stopGyroscope()
	}
	motion.cStick, motion.stepsPerDegree = false, 0
}

// ReadControls reads the buttons, the touch screen and the circle pad now, and the C-stick and the
// gyroscope once StartMotionControls has started them.
func ReadControls() Reading {
	var held C.u32
	var touchX, touchY, circleX, circleY C.int
	C.readControls(&held, &touchX, &touchY, &circleX, &circleY)
	reading := Reading{Held: Buttons(held), TouchX: int(touchX), TouchY: int(touchY), CircleX: int(circleX), CircleY: int(circleY)}
	if motion.cStick {
		var x, y C.int
		C.readCStick(&x, &y)
		reading.CStickX, reading.CStickY = int(x), int(y)
	}
	if motion.stepsPerDegree != 0 {
		var roll, pitch, yaw C.int
		C.readGyroscope(&roll, &pitch, &yaw)
		reading.Gyroscope = gyroscopeFrom(int(roll), int(pitch), int(yaw), motion.stepsPerDegree)
	}
	return reading
}
