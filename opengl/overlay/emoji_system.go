//go:build !(js && wasm)

package overlay

// emojiSourceFor is where the emoji drawn come from: the system's colour emoji fonts in these
// folders, and then the platform, for those no font file has a picture of.
func emojiSourceFor(folders []string) emojiSource {
	return emojiSources{newFontFileEmoji(folders), systemEmoji{}}
}
