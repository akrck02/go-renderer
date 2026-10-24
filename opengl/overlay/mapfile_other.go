//go:build !unix

package overlay

// mapFile reads a whole file: mapping is left to the systems that share it as Unix does.
func mapFile(path string) (data []byte, release func(), err error) {
	return readWhole(path)
}
