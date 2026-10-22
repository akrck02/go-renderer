package windowloop

import "testing"

func TestNotOnMainThreadIsNamedForItsPackage(test *testing.T) {
	if NotOnMainThread.Name() != "windowloop.not-on-main-thread" {
		test.Fatalf("NotOnMainThread is %s", NotOnMainThread.Name())
	}
}
