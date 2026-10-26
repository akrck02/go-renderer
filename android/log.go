//go:build android

package android

/*
#include <android/log.h>
#include <stdlib.h>
#include "activity.h"
*/
import "C"

import (
	"bufio"
	"os"
	"syscall"
	"unsafe"
)

// A shared library's standard output and error go nowhere on Android: what the program prints,
// log.Fatal's message and a panic's report would be lost. They are sent to Android's log instead
// (adb logcat -s Go), a line at a time.
func sendStandardErrorToLog() {
	reader, writer, err := os.Pipe()
	if err != nil {
		logError("standard error stays unread: " + err.Error())
		return
	}
	for _, descriptor := range []int{1, 2} {
		if err := syscall.Dup3(int(writer.Fd()), descriptor, 0); err != nil {
			logError("standard error stays unread: " + err.Error())
			return
		}
	}
	go func() {
		lines := bufio.NewScanner(reader)
		lines.Buffer(make([]byte, 4096), 1<<20)
		for lines.Scan() {
			logInfo(lines.Text())
		}
	}()
}

// LogError writes a line to Android's log (tag Go) at once. A program that ends right after an
// error should use it rather than only log.Fatal: standard error reaches the log through a
// goroutine, which the program's exit may not let run.
func LogError(text string) { logError(text) }

func logInfo(text string) { logText(C.ANDROID_LOG_INFO, text) }

func logError(text string) { logText(C.ANDROID_LOG_ERROR, text) }

func logText(priority C.int, text string) {
	line := C.CString(text)
	defer C.free(unsafe.Pointer(line))
	C.log_text(priority, line)
}
