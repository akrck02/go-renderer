//go:build !nintendo3ds

package overlay

import "log/slog"

// warnByDefault tells of a problem through the default slog logger, which the program chooses
// with slog.SetDefault.
func warnByDefault(message string) { slog.Warn(message) }
