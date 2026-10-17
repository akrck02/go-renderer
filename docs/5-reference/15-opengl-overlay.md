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
| `func UseFonts(fonts Fonts)` | UseFonts makes text draw in other fonts, such as a brand's. |

### Constants and variables

```go
var FontFolders = defaultFontFolders()
```

FontFolders are where the system's fonts are looked for, for characters the program's fonts lack: Android's own, or the usual folders of the desktop's system.

```go
var PrebuiltCharacters = prebuiltCharacters()
```

PrebuiltCharacters are drawn into the atlas as soon as a face is first used, so the first frames with text do not stop to draw them one by one: ASCII, Latin-1 and a few common signs.

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

Fonts are the TrueType or OpenType fonts text is drawn with, one per weight; an empty one keeps the default (the Go fonts).

```go
type Fonts struct {
	Regular, Medium, Bold, Mono []byte
}
```

### Frame

Frame is what the layout needs to know about the frame being drawn.

```go
type Frame struct {
	ViewProjection graphics.Mat4 // the camera's projection times its view
	Width, Height  int           // framebuffer size in pixels
	Scale          float32       // framebuffer pixels per layout pixel: 2 on a high-density screen; 0 = 1
	VerticalScale  float32       // the scene's vertical exaggeration; 0 = 1
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
| `func NewMeasurer() (*Measurer, error)` | NewMeasurer parses the embedded fonts. |
| `func (measurer *Measurer) HasIcon(name string) bool` | HasIcon tells whether Lucide has an icon of this name. |
| `func (measurer *Measurer) MeasureText(text string, size float32, weight draw2d.Weight, scale float32) draw2d.TextMetrics` | MeasureText measures a line at a size in layout pixels on a screen of a density (see draw2d.Measurer). |

### Painter

Painter lays out overlays frame after frame, keeping the atlas and the fonts between frames. It is used from one goroutine, the one that draws.

```go
type Painter struct {

	// Warn reports a problem once (an unknown icon name); nil = the standard logger.
	Warn func(message string)

	// Has unexported fields.
}
```

| Function or method | |
|---|---|
| `func NewPainter() (*Painter, error)` | NewPainter parses the embedded fonts and prepares an empty atlas. |
| `func (painter *Painter) Atlas() *Atlas` | Atlas returns the image the quads point into. |
| `func (painter *Painter) Paint(overlay *scene.Overlay, anchors map[*scene.Node]graphics.Mat4, frame Frame) []Quad` | Paint lays out the whole overlay for a frame. |
| `func (painter *Painter) Pictures() *Pictures` | Pictures returns the image pictured quads read their colours from. |

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
| `func Project(point graphics.Vec4, frame Frame) Projection` | Project places a world point on screen, applying the vertical exaggeration as the scene shaders do. |

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
}
```

---

[← Reference](0-packages.md) · generated from the doc comments by `generate.py`
