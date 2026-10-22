package android

import (
	"os"
	"strings"
	"testing"

	"github.com/akrck02/go-renderer/android/java"
)

func TestPositionsCrossToJavaInUTF16Units(test *testing.T) {
	text := TextInput{Text: "a🌊bc", SelectionStart: 2, SelectionEnd: 2, ComposingStart: 2, ComposingEnd: 4}
	characters, selectionStart, selectionEnd, composingStart, composingEnd := utf16Text(text)
	if len(characters) != 5 || selectionStart != 3 || selectionEnd != 3 || composingStart != 3 || composingEnd != 5 {
		test.Fatalf("the wave is two units: %d units, %d %d %d %d", len(characters), selectionStart, selectionEnd, composingStart, composingEnd)
	}
	if back := fromUTF16(characters, selectionStart, selectionEnd, composingStart, composingEnd); back != text {
		test.Fatalf("and comes back as it went: %+v", back)
	}
	if _, _, _, start, end := utf16Text(TextInput{Text: "ab"}); start != -1 || end != -1 {
		test.Fatalf("nothing composed crosses as -1: %d %d", start, end)
	}
	if back := fromUTF16(characters, 99, -4, -1, -1); back.SelectionStart != 4 || back.SelectionEnd != 0 || back.ComposingEnd != 0 {
		test.Fatalf("positions out of the text are kept inside it: %+v", back)
	}
}

func TestTheProgramsTextIsSentOnlyWhenItDiffers(test *testing.T) {
	var keyboard keyboardText
	field := TextInput{Text: "Harbour", SelectionStart: 7, SelectionEnd: 7}
	if !keyboard.toSend(1, field, false) || keyboard.toSend(1, field, false) {
		test.Fatal("a field is sent when it starts being typed in, then not again while unchanged")
	}
	typed := TextInput{Text: "Harbour li", SelectionStart: 10, SelectionEnd: 10, ComposingStart: 8, ComposingEnd: 10}
	keyboard.keep(1, 3, typed)
	if got, ok := keyboard.take(1); !ok || got != typed || keyboard.takenEdit != 3 {
		test.Fatalf("the keyboard's edit is taken: %+v %v", got, ok)
	}
	if keyboard.toSend(1, typed, false) {
		test.Fatal("the keyboard's own edit is not sent back to it")
	}
	if !keyboard.toSend(1, TextInput{Text: "Harbour l", SelectionStart: 9, SelectionEnd: 9}, false) {
		test.Fatal("a change of the program's (a hardware key) is sent")
	}
	if _, ok := keyboard.take(1); ok {
		test.Fatal("an edit is taken once")
	}
	keyboard.keep(1, 4, typed)
	keyboard.forget()
	if _, ok := keyboard.take(2); ok {
		test.Fatal("an edit of a field no longer typed in is dropped")
	}
}

// textinput.c names the class and its methods' types: TextInput.java must keep them.
func TestTheKeyboardsViewIsWhereTheBridgeLooks(test *testing.T) {
	bridge, err := os.ReadFile("textinput.c")
	if err != nil {
		test.Fatal(err)
	}
	for _, both := range [][2]string{
		{`"io.github.akrck02.gorenderer.TextInput"`, "package io.github.akrck02.gorenderer;"},
		{`"(JILjava/lang/String;IIII)V"`, "static native void edited(long field, int edit, String text, int selectionStart, int selectionEnd,"},
		{`"(JILjava/lang/String;IIIIZ)V"`, "public static void set(final long field, final int edit, final String text, final int selectionStart,"},
		{`"(Z)V"`, "public static void show(final boolean typing)"},
	} {
		if !strings.Contains(string(bridge), both[0]) || !strings.Contains(java.TextInput, both[1]) {
			test.Fatalf("textinput.c and TextInput.java no longer agree on %s", both[0])
		}
	}
	if len(java.Sources) != 2 || java.Sources[1].Text != java.TextInput {
		test.Fatal("the tool compiles both classes")
	}
}
