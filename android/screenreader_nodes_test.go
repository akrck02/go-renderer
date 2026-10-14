package android

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/akrck02/go-renderer/android/java"
)

func TestNodesArePackedForJavaInScreenPixels(t *testing.T) {
	nodes := []AccessibleNode{
		{ID: 7, X: 10, Y: 20, Width: 100, Height: 40, Class: "android.widget.Switch", Label: "Lamp", Value: "",
			Pressable: true, Checkable: true, Checked: true},
		{ID: 9, Parent: 7, Class: "android.widget.SeekBar", Label: "Beam 🔦", Scrollable: true, Maximum: 10, Current: 4},
	}
	packed := packNodes(nodes, 2)
	if !slices.Equal(packed.ids, []int32{7, 9}) || !slices.Equal(packed.parents, []int32{0, 7}) {
		t.Fatalf("ids and parents side by side: %v %v", packed.ids, packed.parents)
	}
	if !slices.Equal(packed.bounds[:4], []float32{20, 40, 220, 120}) {
		t.Fatalf("left, top, right, bottom at twice the layout pixels: %v", packed.bounds[:4])
	}
	if packed.flags[0] != flagPressable|flagCheckable|flagChecked || packed.flags[1] != flagScrollable|flagRanged {
		t.Fatalf("flags: %b %b", packed.flags[0], packed.flags[1])
	}
	if !slices.Equal(packed.ranges[3:], []float32{0, 10, 4}) {
		t.Fatalf("the slider's range: %v", packed.ranges)
	}
	if len(packed.offsets) != 7 {
		t.Fatalf("three texts a node and the end: %v", packed.offsets)
	}
	label := string(utf16.Decode(packed.characters[packed.offsets[4]:packed.offsets[5]]))
	if label != "Beam 🔦" {
		t.Fatalf("the label, emoji and all, in UTF-16: %q", label)
	}
}

// The flags and actions are numbered twice, here and in ScreenReader.java: they must agree.
func TestJavaNumbersFlagsAndActionsAlike(t *testing.T) {
	for _, line := range []string{
		"PRESSABLE = 1, LONG_PRESSABLE = 2, CHECKABLE = 4, CHECKED = 8, SELECTED = 16,",
		"DISABLED = 32, SCROLLABLE = 64, EDITABLE = 128, TYPING = 256, SECRET = 512, HEADING = 1024,",
		"RANGED = 2048;",
		"PRESS = 1, LONG_PRESS = 2, FORWARD = 3, BACKWARD = 4;",
	} {
		if !strings.Contains(java.ScreenReader, line) {
			t.Fatalf("ScreenReader.java no longer says %q", line)
		}
	}
	if flagRanged != 2048 || flagHeading != 1024 || ReaderBackward != 4 {
		t.Fatal("the Go side's numbers moved")
	}
	if !strings.Contains(java.ScreenReader, "package io.github.akrck02.gorenderer;") ||
		!strings.HasSuffix(java.ScreenReaderClass, ".ScreenReader") {
		t.Fatal("the class is where its name says")
	}
}
