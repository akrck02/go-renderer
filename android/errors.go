//go:build android

package android

import "github.com/akrck02/go-renderer/failure"

// VideoNotOpened is a film the phone's decoders cannot open; the details say what Android said.
var VideoNotOpened = failure.Define("android.video-not-opened", "the video could not be opened")
