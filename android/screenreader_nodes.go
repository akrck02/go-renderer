package android

import "unicode/utf16"

// AccessibleNode is one thing on the screen a screen reader (TalkBack) reads and acts on: a button,
// a switch, a line of text. The program's interface makes the list in every frame while a screen
// reader is on (ScreenReaderOn), and ShowAccessibleNodes hands it to Android.
type AccessibleNode struct {
	// ID tells the node apart from one frame to the next: the same while the thing stays, never 0.
	ID int32
	// Parent is the node it is inside (a list that scrolls), or 0 for the window.
	Parent int32
	// X, Y, Width and Height are where it shows, in layout pixels.
	X, Y, Width, Height float32
	// Class is the Android widget it reads as, such as "android.widget.Button": the screen reader
	// says its kind from it ("button", "switch").
	Class        string
	Label, Value string // what it is called, and what it holds (a field's text)

	Pressable, LongPressable bool // a double tap presses it; a double tap held long-presses it
	Checkable, Checked       bool // a checkbox, a switch, a radio button, and whether it is on
	Selected, Disabled       bool
	Scrollable               bool // scrolls, or moves a step: a list, a slider
	Editable, Typing, Secret bool // a field, whether the keyboard types in it, a password
	Heading                  bool

	// Minimum, Maximum and Current are a slider's span and value, when Maximum is above Minimum.
	Minimum, Maximum, Current float32
}

// ReaderActionKind is what a screen reader asks of a node, as ScreenReader.java numbers it.
type ReaderActionKind int

const (
	ReaderPress     ReaderActionKind = iota + 1 // a double tap
	ReaderLongPress                             // a double tap held
	ReaderForward                               // a list scrolls on, a slider goes up a step
	ReaderBackward                              // a list scrolls back, a slider goes down a step
	ReaderSetText                               // a field's text is replaced by the action's Text
)

// ReaderAction is what a screen reader asked of which node.
type ReaderAction struct {
	Node int32
	Kind ReaderActionKind
	// Text is a field's whole new text, for ReaderSetText: what the screen reader typed, pasted or
	// spoke into it.
	Text string
}

// textFrom is a Java string's UTF-16 as Go text.
func textFrom(characters []uint16) string {
	return string(utf16.Decode(characters))
}

// The bits of a node's flags, as ScreenReader.java reads them.
const (
	flagPressable = 1 << iota
	flagLongPressable
	flagCheckable
	flagChecked
	flagSelected
	flagDisabled
	flagScrollable
	flagEditable
	flagTyping
	flagSecret
	flagHeading
	flagRanged
)

// packedNodes are nodes as arrays side by side, as they cross to Java: bounds in window pixels
// (left, top, right, bottom), ranges (minimum, maximum, current), and every node's class, label
// and value in UTF-16, one after the other, each starting at its offset.
type packedNodes struct {
	ids, parents, flags []int32
	bounds, ranges      []float32
	characters          []uint16
	offsets             []int32 // 3 per node and one more: where each text starts, and the end
}

// packNodes packs nodes for Java, at a screen's pixels per layout pixel.
func packNodes(nodes []AccessibleNode, scale float32) packedNodes {
	packed := packedNodes{
		ids: make([]int32, len(nodes)), parents: make([]int32, len(nodes)), flags: make([]int32, len(nodes)),
		bounds: make([]float32, 0, 4*len(nodes)), ranges: make([]float32, 0, 3*len(nodes)),
		offsets: make([]int32, 0, 3*len(nodes)+1),
	}
	for index, node := range nodes {
		packed.ids[index], packed.parents[index], packed.flags[index] = node.ID, node.Parent, flagsOf(node)
		packed.bounds = append(packed.bounds, node.X*scale, node.Y*scale, (node.X+node.Width)*scale, (node.Y+node.Height)*scale)
		packed.ranges = append(packed.ranges, node.Minimum, node.Maximum, node.Current)
		for _, text := range [3]string{node.Class, node.Label, node.Value} {
			packed.offsets = append(packed.offsets, int32(len(packed.characters)))
			// UTF-16, as Java keeps strings: JNI's own UTF-8 is not quite UTF-8, and an emoji in it
			// stops the program
			packed.characters = append(packed.characters, utf16.Encode([]rune(text))...)
		}
	}
	packed.offsets = append(packed.offsets, int32(len(packed.characters)))
	return packed
}

// flagsOf packs what a node is and can do into bits.
func flagsOf(node AccessibleNode) int32 {
	return bit(node.Pressable, flagPressable) | bit(node.LongPressable, flagLongPressable) |
		bit(node.Checkable, flagCheckable) | bit(node.Checked, flagChecked) | bit(node.Selected, flagSelected) |
		bit(node.Disabled, flagDisabled) | bit(node.Scrollable, flagScrollable) | bit(node.Editable, flagEditable) |
		bit(node.Typing, flagTyping) | bit(node.Secret, flagSecret) | bit(node.Heading, flagHeading) |
		bit(node.Maximum > node.Minimum, flagRanged)
}

func bit(set bool, flag int32) int32 {
	if set {
		return flag
	}
	return 0
}
