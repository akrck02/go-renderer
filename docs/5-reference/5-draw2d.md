# draw2d

```go
import "github.com/akrck02/go-renderer/draw2d"
```

Package draw2d is a list of flat shapes to draw over the 3D view, in screen
space: rounded rectangles with a border, soft shadows, lines of text and icons,
each optionally clipped to a rectangle and placed on a layer. A program (or
an interface library) fills a List every frame; a backend paints it on top of
everything else.

Coordinates are layout pixels with the origin at the top-left corner of the
window: the same units as the cursor in input.State. The backend multiplies them
by the screen's density (2 on a high-density screen), so a 13-pixel text has the
same size on every screen and stays sharp.

The package does no drawing and needs no graphics context: it is plain data,
easy to build and to check in tests.

### Constants and variables

```go
var Transparent = Color{}
```

Transparent paints nothing.

### Color

Color is a straight (not premultiplied) RGBA colour, each channel from 0 to 1, as seen on screen: it is not gamma-corrected like the lit scene.

```go
type Color [4]float32
```

| Function or method | |
|---|---|
| `func Hex(text string) Color` | Hex reads a colour written as "#rrggbb" or "#rrggbbaa". |
| `func ParseHex(text string) (Color, error)` | ParseHex reads a colour written as "#rrggbb" or "#rrggbbaa". |
| `func (color Color) IsVisible() bool` | IsVisible tells whether the colour paints anything. |
| `func (color Color) Mix(other Color, amount float32) Color` | Mix blends towards another colour: 0 keeps this one, 1 gives the other. |
| `func (color Color) WithAlpha(opacity float32) Color` | WithAlpha returns the colour with its alpha multiplied by an opacity. |

### Command

Command is one shape of the list. Only the fields of its kind are used; keeping every kind in one flat struct lets a list be reused frame after frame without allocating.

```go
type Command struct {
	Kind    Kind
	Area    Rect // the rectangle, the shadow's caster, the icon's square
	Layer   int  // higher layers cover lower ones; within a layer, later commands cover earlier ones
	Clip    Rect // nothing is drawn outside it, when HasClip
	HasClip bool

	Radius      float32 // rectangle and shadow: corner radius
	Fill        Color   // rectangle: inside; shadow, text, icon: their colour
	Border      Color   // rectangle: the outline
	BorderWidth float32 // rectangle: outline thickness, inside the area
	Blur        float32 // shadow: how far it fades out; Area is the caster grown by it and moved down

	Text     string  // text: the line; icon: the icon's name
	Size     float32 // text and icon: size in layout pixels (the font's em square)
	Weight   Weight  // text
	Baseline float32 // text: y of the baseline; the line starts at Area.X

	Picture *Image // image: what it shows; Fill tints it (white shows it as it is)
}
```

### Image

Image is a picture a list draws: a photo, a map tile, a video's frame. A program makes one for each picture it shows and keeps it between frames, so the backend uploads its pixels once. To show other pixels in it (the next frame of a video), the program writes them into Pixels and calls Changed.

```go
type Image struct {
	// Pixels are the picture's colours, not premultiplied by their opacity.
	Pixels *image.NRGBA

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func DecodeImage(reader io.Reader) (*Image, error)` | DecodeImage reads a PNG, JPEG or GIF picture (a file the program embeds, a download) into an Image. |
| `func NewBlankImage(width, height int) *Image` | NewBlankImage makes an Image of a size, transparent, for pixels the program writes itself (a video's frames). |
| `func NewImage(picture image.Image) *Image` | NewImage makes an Image of a decoded picture (image.Decode's PNG, JPEG or GIF), turning its colours into straight RGBA. |
| `func (picture *Image) Changed()` | Changed tells the backend that Pixels hold another picture of the same size, to upload again. |
| `func (picture *Image) ID() uint64` | ID tells images apart for the life of the program. |
| `func (picture *Image) Size() (width, height int)` | Size is the picture's width and height in pixels. |
| `func (picture *Image) Version() uint64` | Version counts the times the picture changed (Changed). |

### Kind

Kind says what a command draws.

```go
type Kind int
```

```go
const (
	RectangleKind Kind = iota // a rounded rectangle with a fill and a border
	ShadowKind                // a soft shadow under a rounded rectangle
	TextKind                  // one line of text
	IconKind                  // one icon
	ImageKind                 // a picture covering a rounded rectangle
)
```

### List

List collects the commands of a frame. The zero value is an empty list ready to use.

```go
type List struct {
	Commands []Command

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func (list *List) Fingerprint(seed maphash.Seed) uint64` | Fingerprint sums up everything the list draws in one number: two lists with the same fingerprint draw the same picture (barring a hash collision). |
| `func (list *List) Icon(name string, x, y, size float32, color Color)` | Icon draws an icon by name (a Lucide icon, https://lucide.dev) filling a square of size pixels whose top-left corner is at (x, y). |
| `func (list *List) Image(picture *Image, area Rect, radius float32, tint Color)` | Image draws a picture covering a rounded rectangle: scaled to fill it, and cropped evenly at the sides or at the top and bottom when their shapes differ, as photos fill cards and avatars. |
| `func (list *List) IsEmpty() bool` | IsEmpty tells whether the list has nothing to draw. |
| `func (list *List) PopClip()` | PopClip returns to the clip before the last PushClip. |
| `func (list *List) PushClip(area Rect)` | PushClip limits the following commands to a rectangle, inside the current clip if there is one. |
| `func (list *List) Rectangle(area Rect, radius float32, fill, border Color, borderWidth float32)` | Rectangle draws a rounded rectangle: fill inside, and a border of a width drawn inside the area. |
| `func (list *List) Reset()` | Reset empties the list, keeping its memory for the next frame. |
| `func (list *List) SetLayer(layer int)` | SetLayer makes the following commands go on a layer: for example 0 for panels, 1 for menus that open over them and 2 for tooltips. |
| `func (list *List) Shadow(area Rect, radius, blur, offsetY float32, color Color)` | Shadow draws a soft shadow for a rounded rectangle, fading out over blur pixels on every side and falling offsetY pixels below it. |
| `func (list *List) Text(text string, x, baseline, size float32, weight Weight, color Color)` | Text draws one line of text starting at x, with its baseline at baseline. |

### Measurer

Measurer measures text the way the backend draws it, so that what is laid out is what is painted. scale is the screen's density (framebuffer pixels per layout pixel): text is drawn at size·scale pixels, whose widths do not always scale exactly.

```go
type Measurer interface {
	MeasureText(text string, size float32, weight Weight, scale float32) TextMetrics
	// HasIcon tells whether an icon name is known to the backend.
	HasIcon(name string) bool
}
```

### Rect

Rect is an area of the screen: its top-left corner and its size, in layout pixels.

```go
type Rect struct {
	X, Y, Width, Height float32
}
```

| Function or method | |
|---|---|
| `func (rect Rect) Bottom() float32` | Bottom is the y of the rectangle's bottom edge. |
| `func (rect Rect) Contains(x, y float32) bool` | Contains tells whether a point is inside the rectangle (left and top edges included). |
| `func (rect Rect) Inset(amount float32) Rect` | Inset shrinks the rectangle by an amount on every side. |
| `func (rect Rect) Intersect(other Rect) Rect` | Intersect returns the area two rectangles share; it is empty when they do not overlap. |
| `func (rect Rect) IsEmpty() bool` | IsEmpty tells whether the rectangle covers no area. |
| `func (rect Rect) Right() float32` | Right is the x of the rectangle's right edge. |

### TextMetrics

TextMetrics are the measures of a line of text, in layout pixels.

```go
type TextMetrics struct {
	Width   float32 // how far the line reaches from its start
	Ascent  float32 // from the baseline up to the top of the tallest letters
	Descent float32 // from the baseline down to the bottom of the lowest letters
}
```

| Function or method | |
|---|---|
| `func (metrics TextMetrics) Height() float32` | Height is the line's height: ascent plus descent. |

### Weight

Weight is how heavy a text is drawn.

```go
type Weight int
```

```go
const (
	Regular Weight = iota // body text
	Medium                // labels that should stand out a little
	Bold                  // titles
	Mono                  // codes and keys, in a monospaced face, so they stand apart from words
)
```

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
