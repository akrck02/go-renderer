//go:build unix

package overlay

import (
	"os"
	"syscall"
)

// mapFile gives a file's bytes without copying them into the program's memory: the system maps
// the file and reads its pages only when they are touched, shares them with every other program
// that maps the same font, and drops them again when memory is short. A font file holds every
// glyph, and a program draws few of them. release unmaps it; the bytes must not be used after.
// An empty file, or one that cannot be mapped, is read as usual.
func mapFile(path string) (data []byte, release func(), err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close() // a mapping outlives the file it was made from
	information, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	size := information.Size()
	if size <= 0 || int64(int(size)) != size {
		return readWhole(path)
	}
	data, err = syscall.Mmap(int(file.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return readWhole(path)
	}
	// Unmapping can only fail for a range that was never mapped, which this one was.
	return data, func() { _ = syscall.Munmap(data) }, nil
}
