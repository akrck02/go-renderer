# opengl/overlay

```go
import "github.com/akrck02/go-renderer/opengl/overlay"
```

Package overlay lays out the scene's overlay (labels over the world, the header
and the notices, see scene.Overlay) as textured rectangles in screen pixels,
ready for a backend to draw with one texture: the Atlas, which holds the glyphs
of the text, the Lucide icons and the corners of the rounded pills. It does no
drawing itself and needs no graphics context, so it can be tested anywhere;
the OpenGL scene renderer uploads its atlas and draws its quads.

### Functions

| Function | |
|---|---|
| `func DefaultFontFolders() []string` | DefaultFontFolders are where the system's fonts are looked for by default, for characters the program's fonts lack: Android's own, or the usual folders of the desktop's system. |

### Constants and variables

```go
var FontNotRead = failure.Define("overlay.font-not-read", "a font could not be read")
```

FontNotRead is returned by NewPainter and NewMeasurer when one of the fonts (the default ones or those given in Fonts) is not a TrueType or OpenType font.

### Atlas

Atlas is one grey image holding everything the overlay draws: the glyphs of the text, the icons and the rounded corners of the pills. Every picture is a coverage mask (0 = empty, 255 = covered) that the backend tints when it draws it, so a single texture and a single draw call paint the whole overlay. Pictures are added when first needed and never move; when the image is full it doubles in size (keeping what it had in the same place). Version changes whenever pixels change, so the backend knows to upload the image again.

```go
type Atlas struct {
	Image   *image.Alpha
	Version int

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewAtlas() *Atlas` | NewAtlas returns an empty atlas. |
| `func (atlas *Atlas) Size() int` | Size returns the side of the square image, in pixels. |

### Entry

Entry is a picture in the atlas: where its pixels are, and where to draw them relative to a pen position (for glyphs, the dot on the baseline).

```go
type Entry struct {
	Bounds  image.Rectangle // pixels in the atlas
	Offset  image.Point     // top-left corner of the picture relative to the pen
	Advance float32         // how far the pen moves after a glyph, in pixels
}
```

### Fonts

Fonts are what a Painter or a Measurer draws text with: the TrueType or OpenType fonts, one per weight, and where the system's fonts are, for the characters they lack. The zero value is the default: the Go fonts and the system's usual folders.

```go
type Fonts struct {
	// Regular, Medium, Bold and Mono are a font file's bytes each; an empty one keeps the Go font
	// of that weight.
	Regular, Medium, Bold, Mono []byte
	// Folders are where the system's fonts and colour emoji are looked for, for characters the
	// fonts lack. nil is DefaultFontFolders(); an empty, non-nil list looks nowhere, and such
	// characters are drawn as a question mark. In a browser the emoji come from the browser
	// itself, and Folders is not looked in for them.
	Folders []string
}
```

### Frame

Frame is what the layout needs to know about the frame being drawn.

```go
type Frame struct {
	ViewProjection graphics.Matrix // the camera's projection times its view
	Width, Height  int             // framebuffer size in pixels
	Scale          float32         // framebuffer pixels per layout pixel: 2 on a high-density screen; 0 = 1
	VerticalScale  float32         // the scene's vertical exaggeration; 0 = 1
}
```

### LightQuad

LightQuad is a light laid out: a square of the screen it shines on, its colour (alpha: its intensity) and either a picture it shines through (Pictured), or a disc that is full inside Inner (a share of the square's half side) and fades to nothing at its edge.

```go
type LightQuad struct {
	Screen   [4]float32 // left, top, right, bottom on screen
	Color    [4]float32
	Picture  [4]float32 // where it reads its picture in the Pictures, when Pictured
	Pictured bool
	Inner    float32
}
```

### Lighting

Lighting asks the backend, before it draws the quad at At, to light everything drawn so far: the lights from First to Last (in Lights) are added over Ambient into a picture of light, which then multiplies what is on screen. Light is kept at half (0.5 leaves a colour as it is), so it can brighten up to twice.

```go
type Lighting struct {
	At, First, Last int
	Ambient         [4]float32
	// Area is what it lights on screen (left, top, right, bottom): its list's clip, or the whole
	// frame.
	Area [4]float32
}
```

### Measurer

Measurer measures text the way the painter draws it, without an atlas or a graphics context, so an interface library can lay out its controls before the frame is drawn. It implements draw2d.Measurer.

```go
type Measurer struct {
	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewMeasurer(chosen Fonts) (*Measurer, error)` | NewMeasurer reads the fonts (Fonts{} for the default ones). |
| `func (measurer *Measurer) HasIcon(name string) bool` | HasIcon tells whether Lucide has an icon of this name. |
| `func (measurer *Measurer) MeasureText(text string, size float32, weight draw2d.Weight, scale float32) draw2d.TextMetrics` | MeasureText measures a line at a size in layout pixels on a screen of a density (see draw2d.Measurer). |

### Painter

Painter lays out overlays frame after frame, keeping the atlas and the fonts between frames. It is used from one goroutine, the one that draws.

```go
type Painter struct {

	// Warn hears of a problem once (an unknown icon name, lights a GPU cannot draw); nil = the
	// default slog logger, which the program sets with slog.SetDefault (on the Nintendo 3DS, a line
	// on the console).
	Warn func(message string)

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewPainter(chosen Fonts) (*Painter, error)` | NewPainter reads the fonts (Fonts{} for the default ones) and prepares an empty atlas. |
| `func (painter *Painter) Atlas() *Atlas` | Atlas returns the image the quads point into. |
| `func (painter *Painter) Lightings() []Lighting` | Lightings list where, in the quads Paint just gave, the backend lights what it drew so far; empty when nothing is lit, which then costs nothing. |
| `func (painter *Painter) Lights() []LightQuad` | Lights are the lights Paint laid out, which the Lightings take in turn. |
| `func (painter *Painter) Paint(overlay *scene.Overlay, anchors map[*scene.Node]graphics.Matrix, frame Frame) []Quad` | Paint lays out the whole overlay for a frame. |
| `func (painter *Painter) Pictures() *Pictures` | Pictures returns the image pictured quads read their colours from. |
| `func (painter *Painter) Report(message string)` | Report tells the program of a problem a frame works around: through Warn when the program set it, otherwise through warnByDefault (the default slog logger; a line on the console on the Nintendo 3DS, where slog would cost about 42 KB of every program). |
| `func (painter *Painter) Snapshots() []Snapshot` | Snapshots lists where, in the quads Paint just gave, the backend takes and blurs a picture of the frame so far; empty when nothing on screen is frosted, which then costs nothing. |

### Pictures

Pictures is the second atlas: the colours of the images a draw list shows (photos, a video's frames), side by side in one RGBA image. A picture's quads read their colours here and their coverage (the rounded corners) in the grey Atlas, so images are drawn in the same single draw call as everything else. A picture is copied in the first time it is drawn, and copied again into the same place when it changes (draw2d.Image.Changed). When no room is left, the image grows; at its largest, it is emptied and holds only the pictures of the frame being drawn. A backend uploads what changed: the whole image after it grew or was emptied, else only the areas pictures were copied into. The image is made with the first picture: an interface without any needs none.

```go
type Pictures struct {
	Image *image.NRGBA

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewPictures() *Pictures` | NewPictures returns an empty picture atlas, with no image yet. |
| `func (pictures *Pictures) Pending() (whole bool, areas []image.Rectangle)` | Pending tells what to upload: the whole image, or only some areas of it (none when nothing changed). |
| `func (pictures *Pictures) Size() int` | Size returns the side of the square image, in pixels: 0 before the first picture. |
| `func (pictures *Pictures) Uploaded()` | Uploaded forgets what Pending told. |

### Projection

Projection is where a point of the world falls on screen.

```go
type Projection struct {
	X, Y    float32 // pixels from the top-left corner
	Depth   float32 // -1 at the near plane, 1 at the far plane: smaller is nearer
	Visible bool    // in front of the camera and inside the screen
}
```

| Function or method | |
|---|---|
| `func Project(point graphics.Vector, frame Frame) Projection` | Project places a world point on screen, applying the vertical exaggeration as the scene shaders do. |

### Quad

Quad is a rectangle of the atlas drawn on screen with a colour: the atlas gives the coverage, the colour the paint. A quad of an image also reads its colours in the Pictures, which the colour then tints. Coordinates are pixels, with the origin at the top-left corner.

```go
type Quad struct {
	Screen [4]float32 // left, top, right, bottom on screen
	Atlas  [4]float32 // left, top, right, bottom in the atlas
	Color  [4]float32 // straight RGBA; its alpha multiplies the coverage
	// Picture is where the quad reads its colours in the Pictures (left, top, right, bottom), when
	// Pictured.
	Picture  [4]float32
	Pictured bool
	Sharp    bool // the picture is read at the nearest pixel (draw2d.Image's Sharp)
	// Backdrop quads show the blurred picture of what was drawn before them (see Snapshot), cut
	// by the atlas's coverage, instead of their colour.
	Backdrop bool
}
```

### Snapshot

Snapshot asks the backend, before it draws the quad at At, to take a picture of everything drawn so far and blur it by Blur screen pixels (a standard deviation): the backdrop quads after it read that picture. There is one per layer that has backdrops, so many glass panels cost one blur.

```go
type Snapshot struct {
	At   int
	Blur float32
}
```

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
