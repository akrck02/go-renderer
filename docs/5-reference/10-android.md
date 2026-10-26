# android

```go
import "github.com/akrck02/go-renderer/android"
```

> Built only with `GOOS=android`.

Package android runs go-renderer programs as Android applications, with no
golang.org/x/mobile and no Java but one class for screen readers (package java):
the program is a shared library (-buildmode=c-shared) that Android's own
NativeActivity loads. A little C (activity.c) receives the activity's callbacks
and starts main; a thread of the program's own holds the OpenGL ES context,
waits on an Android looper for touches and the activity's news, and draws only
when something happens. Its OpenGL ES calls are direct (gl.go), which matters:
going through x/mobile's bindings, which cross to another thread every three
calls, was about two thirds of what a 3D view cost on a phone.

### Functions

| Function | |
|---|---|
| `func AskForNotifications()` | AskForNotifications shows Android's question to allow the program's notifications, on Android 13 and later; the package's manifest must name android.permission.POST_NOTIFICATIONS. |
| `func EditText(field uint64, text TextInput, secret bool)` | EditText tells the on-screen keyboard the text of the field being typed in, so it edits that text: it suggests words from it, composes in it and deletes from it. |
| `func Feel(feeling Feeling)` | Feel gives a buzz of a kind, if the person has touch feedback on. |
| `func FeelLongPress()` | FeelLongPress gives the short buzz Android's own applications give when a touch becomes a long press, if the person has touch feedback on. |
| `func Graphics() string` | Graphics names the phone's OpenGL ES and GPU once the window has drawn, or what stopped it. |
| `func LastCrash() string` | LastCrash is the report of the last run's crash, or "" when it ended well. |
| `func NotificationsAllowed() bool` | NotificationsAllowed tells whether the program may show notifications: always before Android 13, and on Android 13 and later once the person allowed them (AskForNotifications). |
| `func Notify(channel Channel, id int, title, text string) bool` | Notify shows a notification on a channel in the system's shade, as other applications' are, even while the program is in the background: a title and a line, and touching it opens the program. |
| `func OpenLink(link string) bool` | OpenLink opens a link in the application that shows it: a web page in the browser, mailto: in the mail, tel: in the phone, geo: in the maps. |
| `func Run(app *window.Application, options Options, draw func(functions GL, fresh bool) error) error` | Run draws the application in the activity's window until the program ends (Android ends it). |
| `func ScreenReaderFocus() int32` | ScreenReaderFocus is the node the screen reader is on, which the program rings; 0 for none. |
| `func ScreenReaderOn() bool` | ScreenReaderOn tells whether a screen reader (or another accessibility service) is on: only then is a list of nodes worth making. |
| `func Share(title, text string) bool` | Share offers a text (a link, a note) to the person's other applications, through the system's sheet with a title over it. |
| `func ShowAccessibleNodes(nodes []AccessibleNode)` | ShowAccessibleNodes hands Android what the frame shows, for the screen reader: call it on the drawing goroutine (in a step) with the nodes in reading order. |
| `func ShowKeyboard(typing bool)` | ShowKeyboard asks for the on-screen keyboard while a text field is being typed in, and puts it away when none is: call it in every step (go-ui's window does, with Context.Typing). |
| `func SystemPalettes() (Palettes, bool)` | SystemPalettes returns the phone's palettes as last read, and false on a phone without them (before Android 12) or before the window has read them. |
| `func SystemPrefersDark() bool` | SystemPrefersDark tells whether the phone is in its dark theme, as last read. |
| `func TakeTextEdit(field uint64) (TextInput, bool)` | TakeTextEdit returns what the on-screen keyboard made of a field's text since the last call: its whole text, the selection and the word being composed. |
| `func TextScale() float32` | TextScale is how much larger the person wants text than the default (Settings › Display › Font size): 1, or 1.3 for a larger text. |
| `func Wake()` | Wake makes the window draw a frame now: for a program whose data changes on another goroutine. |

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

### Channel

Channel is a channel notifications go on: Android 8 and later list it, by its label, in the application's settings, where the person can turn it off. The label should be in the person's language.

```go
type Channel struct {
	ID, Label string
}
```

| Function or method | |
|---|---|
| `func MessagesChannel() Channel` | MessagesChannel is a channel for the program's messages, with an English label. |

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

### GL

GL is OpenGL ES 2, called directly on the drawing thread, for go-renderer's gles renderers. Use it only from that thread, inside the drawing function Run calls.

```go
type GL struct{}
```

| Function or method | |
|---|---|
| `func (GL) AttachShader(program gles.Program, shader gles.Shader)` | AttachShader implements gles.Functions. |
| `func (GL) AttachTextureToFramebuffer(target, attachment, textureTarget gles.Enum, texture gles.Texture, level int)` | AttachTextureToFramebuffer implements gles.Functions. |
| `func (GL) BindAttributeLocation(program gles.Program, index uint32, name string)` | BindAttributeLocation implements gles.Functions. |
| `func (GL) BindBuffer(target gles.Enum, buffer gles.Buffer)` | BindBuffer implements gles.Functions. |
| `func (GL) BindFramebuffer(target gles.Enum, framebuffer gles.Framebuffer)` | BindFramebuffer implements gles.Functions. |
| `func (GL) BindTexture(target gles.Enum, texture gles.Texture)` | BindTexture implements gles.Functions. |
| `func (GL) Clear(mask gles.Enum)` | Clear implements gles.Functions. |
| `func (GL) CompileShader(shader gles.Shader)` | CompileShader implements gles.Functions. |
| `func (GL) CopyFramebufferToTexture(target gles.Enum, level, xOffset, yOffset, x, y, width, height int)` | CopyFramebufferToTexture implements gles.Functions. |
| `func (GL) CreateBuffer() gles.Buffer` | CreateBuffer implements gles.Functions. |
| `func (GL) CreateFramebuffer() gles.Framebuffer` | CreateFramebuffer implements gles.Functions. |
| `func (GL) CreateProgram() gles.Program` | CreateProgram implements gles.Functions. |
| `func (GL) CreateShader(stage gles.Enum) gles.Shader` | CreateShader implements gles.Functions. |
| `func (GL) CreateTexture() gles.Texture` | CreateTexture implements gles.Functions. |
| `func (GL) DeleteFramebuffer(framebuffer gles.Framebuffer)` | DeleteFramebuffer implements gles.Functions. |
| `func (GL) DeleteShader(shader gles.Shader)` | DeleteShader implements gles.Functions. |
| `func (GL) DeleteTexture(texture gles.Texture)` | DeleteTexture implements gles.Functions. |
| `func (GL) DescribeVertexAttribute(index uint32, size int, kind gles.Enum, normalized bool, stride, offset int)` | DescribeVertexAttribute implements gles.Functions. |
| `func (GL) Disable(capability gles.Enum)` | Disable implements gles.Functions. |
| `func (GL) DisableVertexAttribute(index uint32)` | DisableVertexAttribute implements gles.Functions. |
| `func (GL) DrawArrays(mode gles.Enum, first, count int)` | DrawArrays implements gles.Functions. |
| `func (GL) DrawElements(mode gles.Enum, count int, kind gles.Enum, offset int)` | DrawElements implements gles.Functions. |
| `func (GL) Enable(capability gles.Enum)` | Enable implements gles.Functions. |
| `func (GL) EnableVertexAttribute(index uint32)` | EnableVertexAttribute implements gles.Functions. |
| `func (GL) GenerateMipmap(target gles.Enum)` | GenerateMipmap implements gles.Functions. |
| `func (GL) LinkProgram(program gles.Program)` | LinkProgram implements gles.Functions. |
| `func (GL) ProgramLinked(program gles.Program) (bool, string)` | ProgramLinked implements gles.Functions. |
| `func (GL) ReplaceTexturePart(target gles.Enum, level, x, y, width, height int, format, kind gles.Enum, pixels []byte)` | ReplaceTexturePart implements gles.Functions. |
| `func (GL) SelectTextureUnit(unit gles.Enum)` | SelectTextureUnit implements gles.Functions. |
| `func (GL) SetBlendFactors(sourceColor, destinationColor, sourceAlpha, destinationAlpha gles.Enum)` | SetBlendFactors implements gles.Functions. |
| `func (GL) SetBufferData(target gles.Enum, data []byte, usage gles.Enum)` | SetBufferData implements gles.Functions. |
| `func (GL) SetClearColor(red, green, blue, alpha float32)` | SetClearColor implements gles.Functions. |
| `func (GL) SetDepthFunction(function gles.Enum)` | SetDepthFunction implements gles.Functions. |
| `func (GL) SetDepthWrite(write bool)` | SetDepthWrite implements gles.Functions. |
| `func (GL) SetPixelStorage(name gles.Enum, value int32)` | SetPixelStorage implements gles.Functions. |
| `func (GL) SetShaderSource(shader gles.Shader, source string)` | SetShaderSource implements gles.Functions. |
| `func (GL) SetTextureImage(target gles.Enum, level int, internalFormat gles.Enum, width, height int, format, kind gles.Enum, pixels []byte)` | SetTextureImage implements gles.Functions. |
| `func (GL) SetTextureParameter(target, name gles.Enum, value int32)` | SetTextureParameter implements gles.Functions. |
| `func (GL) SetUniformFloat(location gles.Uniform, value float32)` | SetUniformFloat implements gles.Functions. |
| `func (GL) SetUniformFourFloats(location gles.Uniform, first, second, third, fourth float32)` | SetUniformFourFloats implements gles.Functions. |
| `func (GL) SetUniformInteger(location gles.Uniform, value int32)` | SetUniformInteger implements gles.Functions. |
| `func (GL) SetUniformMatrix3By3(location gles.Uniform, matrix [9]float32)` | SetUniformMatrix3By3 implements gles.Functions. |
| `func (GL) SetUniformMatrix4By4(location gles.Uniform, matrix [16]float32)` | SetUniformMatrix4By4 implements gles.Functions. |
| `func (GL) SetUniformThreeFloats(location gles.Uniform, first, second, third float32)` | SetUniformThreeFloats implements gles.Functions. |
| `func (GL) SetUniformTwoFloats(location gles.Uniform, first, second float32)` | SetUniformTwoFloats implements gles.Functions. |
| `func (GL) SetViewport(x, y, width, height int)` | SetViewport implements gles.Functions. |
| `func (GL) ShaderCompiled(shader gles.Shader) (bool, string)` | ShaderCompiled implements gles.Functions. |
| `func (GL) UniformLocation(program gles.Program, name string) gles.Uniform` | UniformLocation implements gles.Functions. |
| `func (GL) UseProgram(program gles.Program)` | UseProgram implements gles.Functions. |

### Insets

Insets are what the system's bars cover of each edge, in layout pixels.

```go
type Insets struct{ Top, Bottom, Left, Right float32 }
```

| Function or method | |
|---|---|
| `func CurrentInsets() Insets` | CurrentInsets returns what the system's bars cover, as last read. |

### Options

Options tune the window.

```go
type Options struct {
	// MaxFramesPerSecond caps how often it draws; 0 = 60. Phones refresh at 90 or 120 Hz, and a
	// finger dragging a 3D view would otherwise ask for a frame at every touch event.
	MaxFramesPerSecond int
	// Hidden is called on the drawing goroutine when the program leaves the screen (another
	// application comes up, the phone locks): Android may end it after that without a word, so it
	// is the moment to keep what must survive.
	Hidden func()
	// Shown is called on the drawing goroutine when the program comes on the screen: at its start,
	// and back from another application or the lock screen. It is the moment to play again what
	// Hidden stopped (a sound).
	Shown func()
}
```

### Palettes

Palettes are the colours Android 12 and later make from the wallpaper (Material You): three accent palettes and two neutral ones, each in 13 shades from white (0) to black (1000), as Android names them (system_accent1_0, …_10, …_50, …_100 and on by hundreds). An application that takes its colours from them matches the phone, as the system's own do.

```go
type Palettes struct {
	Accent1, Accent2, Accent3, Neutral1, Neutral2 [13]draw2d.Color
}
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

| Function or method | |
|---|---|
| `func TakeReaderActions() []ReaderAction` | TakeReaderActions returns what the screen reader asked since the last call, and forgets it. |

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

### Video

Video decodes a film's pictures with the phone's own decoders (the NDK's MediaCodec), a step at a time and never waiting, from the goroutine that draws. Its sound is not played.

```go
type Video struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func OpenVideo(path string) (*Video, error)` | OpenVideo reads a film (MP4 or WebM, as the phone's decoders read it: H.264, H.265, VP8, VP9, AV1) from a file and starts decoding its pictures. |
| `func (video *Video) Advance(until time.Duration, pixels *image.NRGBA) (changed, ended bool)` | Advance decodes the film up to a moment and, when a picture due by then has arrived, copies it into pixels (as much of it as fits). |
| `func (video *Video) Close()` | Close lets go of the decoder and the file. |
| `func (video *Video) Duration() time.Duration` | Duration is how long the film lasts; 0 when its file does not say. |
| `func (video *Video) Rewind()` | Rewind goes back to the film's start. |
| `func (video *Video) Size() (width, height int)` | Size is the film's pictures' width and height, in pixels. |

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
