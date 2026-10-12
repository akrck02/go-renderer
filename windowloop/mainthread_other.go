//go:build !darwin

package windowloop

// OnMainThread is only checked on macOS, the one system that requires it.
func OnMainThread() bool { return true }
