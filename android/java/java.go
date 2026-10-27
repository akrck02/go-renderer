// Package java holds the Java a go-renderer program for Android carries, as source, for the tool
// that packages it: it compiles the source against the platform's android.jar and puts the classes
// in the package's classes.dex. Everything else runs in Go and C; this is the part Android only
// offers to Java (a screen reader's view of the window, see ScreenReader.java, a view the
// on-screen keyboard types in, see TextInput.java, and the phone's own emoji, see EmojiPicture.java).
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

// TextInput is TextInput.java: the view the on-screen keyboard types in, standing in for a field
// the program draws.
//
//go:embed TextInput.java
var TextInput string

// TextInputPath is where TextInput's source goes for javac.
const TextInputPath = "io/github/akrck02/gorenderer/TextInput.java"

// EmojiPicture is EmojiPicture.java: an emoji drawn with the phone's own emoji font.
//
//go:embed EmojiPicture.java
var EmojiPicture string

// EmojiPictureClass is the class's full name, as a class loader finds it, and EmojiPicturePath is
// where its source goes for javac.
const (
	EmojiPictureClass = "io.github.akrck02.gorenderer.EmojiPicture"
	EmojiPicturePath  = "io/github/akrck02/gorenderer/EmojiPicture.java"
)

// Source is one Java file a program carries: where it goes for javac, and what it holds.
type Source struct {
	Path, Text string
}

// Sources are every Java file a program carries, for a tool to compile together.
var Sources = []Source{{ScreenReaderPath, ScreenReader}, {TextInputPath, TextInput}, {EmojiPicturePath, EmojiPicture}}
