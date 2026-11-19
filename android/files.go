//go:build android

package android

/*
#include "activity.h"
*/
import "C"

import (
	"unsafe"
)

// files is the one file chooser the program has.
var files fileChooser

// CanChooseFiles tells whether the package carries the file chooser (FileChooser.java): a package
// made by an older tool does not.
func CanChooseFiles() bool { return C.files_available() != 0 }

// ChooseFiles opens Android's document chooser, where the person picks one file or several.
// chosen is called once when they are done, never on Android's main thread, with what they chose
// (nothing if they cancelled, or the chooser could not open after all); a file over largest bytes
// comes unread, with its name and size. It returns false when there is no chooser to open: the
// package has none, or one is open already. It answers with a bool, as Go UI's ChooseFiles does:
// a caller that cannot open the chooser hides its button, whichever the reason.
func ChooseFiles(largest int64, chosen func([]ChosenFile)) bool {
	if !CanChooseFiles() || !files.start(largest, chosen) {
		return false
	}
	if C.files_open(C.jlong(largest)) == 0 {
		files.abandon()
		return false
	}
	return true
}

//export goFileChosen
func goFileChosen(characters *C.jchar, length C.int, size C.jlong, content unsafe.Pointer, contentLength C.int, read C.int) {
	name := textFrom(unsafe.Slice((*uint16)(unsafe.Pointer(characters)), int(length)))
	var bytes []byte
	if read != 0 {
		bytes = C.GoBytes(content, contentLength)
	}
	files.add(name, int64(size), bytes)
}

//export goFilesChosen
func goFilesChosen() { files.finish() }
