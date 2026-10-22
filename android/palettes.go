//go:build android

package android

/*
#include <stdlib.h>
#include "activity.h"
*/
import "C"

import (
	"sync/atomic"
	"unsafe"

	"github.com/akrck02/go-renderer/draw2d"
)

// paletteTones are the shades of each of Android's system palettes, by the numbers in their names:
// 0 is white, 1000 black, 500 the palette's middle.
var paletteTones = [13]int{0, 10, 50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}

// Palettes are the colours Android 12 and later make from the wallpaper (Material You): three
// accent palettes and two neutral ones, each in 13 shades from white (0) to black (1000), as Android names them
// (system_accent1_0, …_10, …_50, …_100 and on by hundreds). An application that takes its
// colours from them matches the phone, as the system's own do.
type Palettes struct {
	Accent1, Accent2, Accent3, Neutral1, Neutral2 [13]draw2d.Color
}

var systemPalettes atomic.Pointer[Palettes]

// SystemPalettes returns the phone's palettes as last read, and false on a phone without them
// (before Android 12) or before the window has read them. They are read again when the activity
// comes back or its configuration changes, which is when the wallpaper's colours change.
func SystemPalettes() (Palettes, bool) {
	palettes := systemPalettes.Load()
	if palettes == nil {
		return Palettes{}, false
	}
	return *palettes, true
}

// readPalettes asks Android for its palettes through JNI.
func readPalettes(env *C.JNIEnv) {
	names := []string{"accent1", "accent2", "accent3", "neutral1", "neutral2"}
	cNames := make([]*C.char, len(names))
	for index, name := range names {
		cNames[index] = C.CString(name)
		defer C.free(unsafe.Pointer(cNames[index]))
	}
	var tones [len(paletteTones)]C.int
	for index, tone := range paletteTones {
		tones[index] = C.int(tone)
	}
	pinnedNames := (**C.char)(C.malloc(C.size_t(len(names)) * C.size_t(unsafe.Sizeof(uintptr(0)))))
	defer C.free(unsafe.Pointer(pinnedNames))
	copy(unsafe.Slice(pinnedNames, len(names)), cNames)
	var colours [5 * len(paletteTones)]C.uint
	if C.system_colours(env, pinnedNames, C.int(len(names)), &tones[0], C.int(len(paletteTones)), &colours[0]) == 0 {
		systemPalettes.Store(nil)
		return
	}
	var palettes Palettes
	for palette, shades := range []*[13]draw2d.Color{&palettes.Accent1, &palettes.Accent2, &palettes.Accent3, &palettes.Neutral1, &palettes.Neutral2} {
		for tone := range paletteTones {
			shades[tone] = fromARGB(uint32(colours[palette*len(paletteTones)+tone]))
		}
	}
	systemPalettes.Store(&palettes)
}

// fromARGB turns Android's packed colour into the renderer's.
func fromARGB(packed uint32) draw2d.Color {
	channel := func(shift uint) float32 { return float32((packed>>shift)&0xff) / 255 }
	return draw2d.Color{channel(16), channel(8), channel(0), channel(24)}
}
