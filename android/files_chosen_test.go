package android

import (
	"slices"
	"strings"
	"testing"

	"github.com/akrck02/go-renderer/android/java"
)

// One chooser covers the screen at a time: a second, opened while the first waits for the person,
// would leave Go not knowing which answer is which.
func TestASecondChooserWhileOneIsOpenIsRefused(test *testing.T) {
	var chooser fileChooser
	if !chooser.start(10, func([]ChosenFile) {}) {
		test.Fatal("the first chooser was refused")
	}
	if chooser.start(10, func([]ChosenFile) {}) {
		test.Fatal("a second chooser opened over the first")
	}
	chooser.finish()
	if !chooser.start(10, func([]ChosenFile) {}) {
		test.Fatal("no chooser opens once the first has answered")
	}
}

// A file bigger than the caller takes comes back by its name and size, never read whole: a video
// chosen by mistake does not fill the phone's memory.
func TestAFileOverTheLimitComesBackUnread(test *testing.T) {
	var chooser fileChooser
	var got []ChosenFile
	chooser.start(4, func(chosen []ChosenFile) { got = chosen })
	chooser.add("notes.txt", 3, []byte("abc"))
	chooser.add("video.mp4", 5, nil)
	chooser.add("liar.txt", 2, []byte("abcde")) // a size the phone told wrong
	chooser.finish()
	want := []ChosenFile{
		{Name: "notes.txt", Size: 3, Content: []byte("abc"), Read: true},
		{Name: "video.mp4", Size: 5},
		{Name: "liar.txt", Size: 5},
	}
	if !slices.EqualFunc(got, want, sameFile) {
		test.Fatalf("got %+v; want %+v", got, want)
	}
}

// Cancelling is an answer too: the caller hears it, with no files, and may open the chooser again.
func TestACancelledChooserHandsBackNoFiles(test *testing.T) {
	var chooser fileChooser
	called := false
	chooser.start(10, func(chosen []ChosenFile) { called = len(chosen) == 0 })
	chooser.finish()
	if !called {
		test.Fatal("the caller was not told the chooser closed empty")
	}
}

// A chooser that could not open is forgotten, so the next one can.
func TestAChooserThatCouldNotOpenLetsTheNextOne(test *testing.T) {
	var chooser fileChooser
	chooser.start(10, func([]ChosenFile) { test.Fatal("called for a chooser that never opened") })
	chooser.abandon()
	if !chooser.start(10, func([]ChosenFile) {}) {
		test.Fatal("the abandoned chooser still blocks the next")
	}
}

// An answer that reaches Go after it forgot its chooser (the activity was rebuilt meanwhile) is
// dropped, never mixed into the files of the next one.
func TestAnAnswerWithNoChooserOpenIsIgnored(test *testing.T) {
	var chooser fileChooser
	chooser.add("late.txt", 4, []byte("late"))
	chooser.finish()
	var got []ChosenFile
	chooser.start(10, func(chosen []ChosenFile) { got = chosen })
	chooser.add("notes.txt", 3, []byte("abc"))
	chooser.finish()
	if len(got) != 1 || got[0].Name != "notes.txt" {
		test.Fatalf("got %+v; want notes.txt alone", got)
	}
}

// The class is loaded by its name, and compiled only if the tool is given it.
func TestTheFileChooserClassIsWhereItsNameSays(test *testing.T) {
	if !strings.Contains(java.FileChooser, "package io.github.akrck02.gorenderer;") ||
		!strings.Contains(java.FileChooser, "public final class FileChooser") ||
		!strings.HasSuffix(java.FileChooserClass, ".FileChooser") ||
		!strings.HasPrefix(java.FileChooserPath, strings.ReplaceAll(java.FileChooserClass, ".", "/")) {
		test.Fatal("the class's package, name and path disagree")
	}
	if !slices.Contains(java.Sources, java.Source{Path: java.FileChooserPath, Text: java.FileChooser}) {
		test.Fatal("the tool is not given the class to compile")
	}
}

func sameFile(one, other ChosenFile) bool {
	return one.Name == other.Name && one.Size == other.Size && one.Read == other.Read && string(one.Content) == string(other.Content)
}
