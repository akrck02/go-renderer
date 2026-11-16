package android

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/akrck02/go-renderer/android/java"
)

// The activity a package names keeps the intent that brings it back, and notify.c reads the same
// extra it writes: a touched notification's target reaches the program.
func TestATouchedNotificationsTargetReachesTheProgram(test *testing.T) {
	if !strings.Contains(java.Activity, "extends NativeActivity") || !strings.Contains(java.Activity, "setIntent(intent)") {
		test.Fatal("Activity.java no longer keeps the intent that brings it back")
	}
	if !strings.HasPrefix(java.ActivityPath, strings.ReplaceAll(java.ActivityClass, ".", "/")) {
		test.Fatalf("the activity's class %s and its path %s disagree", java.ActivityClass, java.ActivityPath)
	}
	if !slices.Contains(java.Sources, java.Source{Path: java.ActivityPath, Text: java.Activity}) {
		test.Fatal("the tool does not compile the activity")
	}
	bridge, err := os.ReadFile("notify.c")
	if err != nil {
		test.Fatal(err)
	}
	for _, needed := range []string{"putExtra", "getStringExtra", "removeExtra", "0x20000000"} {
		if !strings.Contains(string(bridge), needed) {
			test.Fatalf("notify.c no longer has %s", needed)
		}
	}
}
