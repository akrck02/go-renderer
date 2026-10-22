package windowloop

import "github.com/akrck02/go-renderer/failure"

// NotOnMainThread is returned by a window's StartLoop called from another thread than the
// program's main one, where macOS delivers no window events. The fix: call runtime.LockOSThread in
// an init function of package main.
var NotOnMainThread = failure.Define("windowloop.not-on-main-thread",
	"the window loop must run on the main thread; call runtime.LockOSThread in an init function of package main")
