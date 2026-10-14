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
)

// textScale is the person's text size, as last read (readSystem).
var textScale atomic.Value

// TextScale is how much larger the person wants text than the default (Settings › Display › Font
// size): 1, or 1.3 for a larger text. Layout pixels stay the same; only text grows.
func TextScale() float32 {
	if scale, found := textScale.Load().(float32); found {
		return scale
	}
	return 1
}

// systemClipboard is Android's clipboard, which the input state's text fields copy to and paste
// from.
type systemClipboard struct{}

func (systemClipboard) ClipboardText() string {
	text := C.clipboard_text()
	if text == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(text))
	return C.GoString(text)
}

func (systemClipboard) SetClipboardText(text string) {
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.set_clipboard_text(value)
}

// OpenLink opens a link in the application that shows it: a web page in the browser, mailto: in
// the mail, tel: in the phone, geo: in the maps. It returns false when no application opens it.
// Safe from any goroutine.
func OpenLink(link string) bool {
	value := C.CString(link)
	defer C.free(unsafe.Pointer(value))
	return C.open_link(value) != 0
}

// Share offers a text (a link, a note) to the person's other applications, through the system's
// sheet with a title over it. It returns false when the sheet could not open. Safe from any
// goroutine.
func Share(title, text string) bool {
	heading, value := C.CString(title), C.CString(text)
	defer C.free(unsafe.Pointer(heading))
	defer C.free(unsafe.Pointer(value))
	return C.share_text(heading, value) != 0
}
