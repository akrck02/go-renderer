//go:build nintendo3ds

package overlay

// warnByDefault writes the problem as a line on the console. On the Nintendo 3DS log/slog would
// add about 42 KB to every program for a message the console shows just as well.
func warnByDefault(message string) { println("warning:", message) }
