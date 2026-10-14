// Package java holds the Java a go-renderer program for Android carries, as source, for the tool
// that packages it: it compiles the source against the platform's android.jar and puts the classes
// in the package's classes.dex. Everything else runs in Go and C; this is the part Android only
// offers to Java (a screen reader's view of the window, see ScreenReader.java).
package java

import _ "embed"

// ScreenReader is ScreenReader.java: what TalkBack reads of the window.
//
//go:embed ScreenReader.java
var ScreenReader string

// ScreenReaderClass is the class's full name, as a class loader finds it, and ScreenReaderPath is
// where its source goes for javac, under the folder it compiles from.
const (
	ScreenReaderClass = "io.github.akrck02.gorenderer.ScreenReader"
	ScreenReaderPath  = "io/github/akrck02/gorenderer/ScreenReader.java"
)
