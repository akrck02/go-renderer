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
*/
import "C"

// ReadControls reads the buttons, the touch screen and the circle pad now.
func ReadControls() Reading {
	var held C.u32
	var touchX, touchY, circleX, circleY C.int
	C.readControls(&held, &touchX, &touchY, &circleX, &circleY)
	return Reading{Held: Buttons(held), TouchX: int(touchX), TouchY: int(touchY), CircleX: int(circleX), CircleY: int(circleY)}
}
