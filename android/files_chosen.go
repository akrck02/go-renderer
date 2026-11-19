package android

import "sync"

// ChosenFile is a file the person chose in Android's file chooser: its name as the chooser shows
// it, its size in bytes, and its bytes when Read. A file that is not read comes with its name and
// size only, so the caller can say why it was left out: a Size over the largest asked, or
// UnknownSize, is too big; a smaller Size means it could not be read.
type ChosenFile struct {
	Name    string
	Size    int64
	Content []byte
	Read    bool
}

// UnknownSize is the Size of a file too big to read whose real size the app holding it did not
// tell.
const UnknownSize = -1

// fileChooser is the one file chooser open at a time: the largest file its caller takes, the files
// Java has handed over so far, and whom to tell when the person is done.
type fileChooser struct {
	mutex   sync.Mutex
	open    bool
	largest int64
	files   []ChosenFile
	chosen  func([]ChosenFile)
}

// start takes the chooser for a caller; false if another caller has it.
func (chooser *fileChooser) start(largest int64, chosen func([]ChosenFile)) bool {
	chooser.mutex.Lock()
	defer chooser.mutex.Unlock()
	if chooser.open {
		return false
	}
	chooser.open, chooser.largest, chooser.files, chooser.chosen = true, largest, nil, chosen
	return true
}

// add keeps one chosen file of the open chooser. Its content is nil when Java did not read it; content longer than
// the size Java was told is believed over the size, as names and sizes come from other apps.
func (chooser *fileChooser) add(name string, size int64, content []byte) {
	chooser.mutex.Lock()
	defer chooser.mutex.Unlock()
	if !chooser.open {
		return // a late answer, to a chooser Go already forgot
	}
	if int64(len(content)) > size {
		size = int64(len(content))
	}
	file := ChosenFile{Name: name, Size: size}
	if content != nil && size <= chooser.largest {
		file.Content, file.Read = content, true
	}
	chooser.files = append(chooser.files, file)
}

// finish frees the chooser and tells its caller what was chosen, nothing if the person
// cancelled. The caller is told outside the lock, so it may open the chooser again.
func (chooser *fileChooser) finish() {
	chooser.mutex.Lock()
	if !chooser.open {
		chooser.mutex.Unlock()
		return // a late answer, to a chooser Go already forgot
	}
	files, chosen := chooser.files, chooser.chosen
	chooser.open, chooser.files, chooser.chosen = false, nil, nil
	chooser.mutex.Unlock()
	chosen(files)
}

// abandon frees a chooser that could not open, without telling its caller, who was told false.
func (chooser *fileChooser) abandon() {
	chooser.mutex.Lock()
	defer chooser.mutex.Unlock()
	chooser.open, chooser.files, chooser.chosen = false, nil, nil
}
