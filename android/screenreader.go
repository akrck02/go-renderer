//go:build android

package android

/*
#include "activity.h"
*/
import "C"

import (
	"slices"
	"sync"
	"sync/atomic"
	"unsafe"
)

// A screen reader (TalkBack) reads the window through ScreenReader.java, which answers Android with
// the nodes the program last showed. While one is on, a finger exploring the screen comes as hover
// events, which go to Java to be read; a double tap comes back here as an action on a node, and
// text the screen reader puts in a field as one carrying it.

var (
	readerOn      atomic.Bool
	readerFocus   atomic.Int32
	readerActions struct {
		sync.Mutex
		list []ReaderAction
	}
	shownNodes []AccessibleNode // the nodes Java has, on the drawing thread
)

// ScreenReaderOn tells whether a screen reader (or another accessibility service) is on: only then
// is a list of nodes worth making.
func ScreenReaderOn() bool { return readerOn.Load() }

// ScreenReaderFocus is the node the screen reader is on, which the program rings; 0 for none.
func ScreenReaderFocus() int32 { return readerFocus.Load() }

// TakeReaderActions returns what the screen reader asked since the last call, and forgets it.
func TakeReaderActions() []ReaderAction {
	readerActions.Lock()
	defer readerActions.Unlock()
	taken := readerActions.list
	readerActions.list = nil
	return taken
}

// ShowAccessibleNodes hands Android what the frame shows, for the screen reader: call it on the
// drawing goroutine (in a step) with the nodes in reading order. It costs nothing while no screen
// reader is on, and sends the nodes only when they changed.
func ShowAccessibleNodes(nodes []AccessibleNode) {
	if resendNodes.Swap(false) {
		shownNodes = shownNodes[:0]
	}
	if !readerOn.Load() || drawing == nil || slices.Equal(nodes, shownNodes) {
		return
	}
	shownNodes = append(shownNodes[:0], nodes...)
	packed := packNodes(nodes, drawing.reader.scale)
	C.screen_reader_update(C.int(len(nodes)), (*C.int)(first(packed.ids)), (*C.int)(first(packed.parents)),
		(*C.int)(first(packed.flags)), (*C.float)(first(packed.bounds)), (*C.float)(first(packed.ranges)),
		(*C.jchar)(first(packed.characters)), (*C.int)(first(packed.offsets)))
}

// first points at a slice's first element, or is nil for an empty one.
func first[T any](values []T) unsafe.Pointer {
	if len(values) == 0 {
		return nil
	}
	return unsafe.Pointer(&values[0])
}

// hoverForReader passes a finger exploring the screen on to the screen reader.
func hoverForReader(action C.int32_t, x, y float32) {
	if readerOn.Load() {
		C.screen_reader_hover(C.int(action), C.float(x), C.float(y))
	}
}

//export goReaderActed
func goReaderActed(node, action C.int) {
	readerActions.Lock()
	readerActions.list = append(readerActions.list, ReaderAction{Node: int32(node), Kind: ReaderActionKind(action)})
	readerActions.Unlock()
	Wake()
}

//export goReaderTyped
func goReaderTyped(node C.int, characters *C.jchar, length C.int) {
	text := textFrom(unsafe.Slice((*uint16)(unsafe.Pointer(characters)), int(length)))
	readerActions.Lock()
	readerActions.list = append(readerActions.list, ReaderAction{Node: int32(node), Kind: ReaderSetText, Text: text})
	readerActions.Unlock()
	Wake()
}

//export goReaderFocused
func goReaderFocused(node C.int) {
	readerFocus.Store(int32(node))
	Wake() // the ring moves
}

//export goReaderChanged
func goReaderChanged(on C.int) {
	readerOn.Store(on != 0)
	if on == 0 {
		readerFocus.Store(0)
	}
	resendNodes.Store(true)
	Wake()
}

// resendNodes asks the drawing thread to send its nodes again even if unchanged: a screen reader
// turned on finds Java's list empty.
var resendNodes atomic.Bool
