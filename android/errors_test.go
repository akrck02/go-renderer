//go:build android

package android

import "testing"

func TestVideoNotOpenedIsNamedForItsPackage(test *testing.T) {
	if VideoNotOpened.Name() != "android.video-not-opened" {
		test.Fatalf("VideoNotOpened is %s", VideoNotOpened.Name())
	}
}
