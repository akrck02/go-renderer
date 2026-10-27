//go:build !(js && wasm)

package overlay

// emojiSourceFor is where the emoji drawn come from: the system's colour emoji fonts in these
// folders.
func emojiSourceFor(folders []string) emojiSource {
	return newFontFileEmoji(folders)
}
