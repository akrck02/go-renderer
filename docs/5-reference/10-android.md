# android

```go
import "github.com/akrck02/go-renderer/android"
```

> Built only with `GOOS=android`.

### Constants and variables

```go
var VideoNotOpened = failure.Define("android.video-not-opened", "the video could not be opened")
```

VideoNotOpened is a film the phone's decoders cannot open; the details say what Android said.

### AccessibleNode

AccessibleNode is one thing on the screen a screen reader (TalkBack) reads and acts on: a button, a switch, a line of text. The program's interface makes the list in every frame while a screen reader is on (ScreenReaderOn), and ShowAccessibleNodes hands it to Android.

```go
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
```

### Feeling

Feeling is a kind of buzz a touch gives, as Android's own views give them.

```go
type Feeling int
```

```go
const (
	// FeelingLongPress is the buzz of a touch that became a long press.
	FeelingLongPress Feeling = iota
	// FeelingTick is a light tick: something chosen, a switch flipped, a step on a dial.
	FeelingTick
	// FeelingConfirm says something went through. Before Android 11 it is a tick.
	FeelingConfirm
	// FeelingReject says something went wrong or needs a look. Before Android 11 it is the buzz of a
	// long press, the strongest the older phones have.
	FeelingReject
)
```

### ReaderAction

ReaderAction is what a screen reader asked of which node.

```go
type ReaderAction struct {
	Node int32
	Kind ReaderActionKind
	// Text is a field's whole new text, for ReaderSetText: what the screen reader typed, pasted or
	// spoke into it.
	Text string
}
```

### ReaderActionKind

ReaderActionKind is what a screen reader asks of a node, as ScreenReader.java numbers it.

```go
type ReaderActionKind int
```

```go
const (
	ReaderPress     ReaderActionKind = iota + 1 // a double tap
	ReaderLongPress                             // a double tap held
	ReaderForward                               // a list scrolls on, a slider goes up a step
	ReaderBackward                              // a list scrolls back, a slider goes down a step
	ReaderSetText                               // a field's text is replaced by the action's Text
)
```

### TextInput

TextInput is the text of the field being typed in, as the on-screen keyboard sees and edits it. Positions count characters (runes); a composing word is the one the keyboard is still making (a prediction, a Chinese or Japanese reading), shown underlined until the keyboard commits it.

```go
type TextInput struct {
	Text                         string
	SelectionStart, SelectionEnd int // the caret when they are equal
	ComposingStart, ComposingEnd int // equal when nothing is being composed
}
```

### YUVPlanes

YUVPlanes is a decoded video picture: a full-size luma plane and two chroma planes of half its width and height, each with how far apart its rows are, and its chroma pixels (1 when a plane holds only its own, 2 when the two are interleaved). Crop is the part that shows.

```go
type YUVPlanes struct {
	Luma, Blue, Red          []byte
	LumaRow, BlueRow, RedRow int
	BluePixel, RedPixel      int
	Crop                     image.Rectangle
}
```

| Function or method | |
|---|---|
| `func (planes YUVPlanes) ToRGBA(into *image.NRGBA)` | ToRGBA turns the picture into straight RGBA, as many pixels as fit into, from the top-left of the crop. |

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
