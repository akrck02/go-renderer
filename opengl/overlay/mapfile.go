package overlay

import "os"

// readWhole reads a file into the program's memory; there is nothing to release.
func readWhole(path string) (data []byte, release func(), err error) {
	data, err = os.ReadFile(path)
	return data, func() {}, err
}
