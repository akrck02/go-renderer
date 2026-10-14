package draw2d

import (
	"encoding/binary"
	"hash/maphash"
	"math"
)

// Kind says what a command draws.
type Kind int

const (
	RectangleKind Kind = iota // a rounded rectangle with a fill and a border
	ShadowKind                // a soft shadow under a rounded rectangle
	TextKind                  // one line of text
	IconKind                  // one icon
	ImageKind                 // a picture covering a rounded rectangle
)

// Command is one shape of the list. Only the fields of its kind are used; keeping every kind in
// one flat struct lets a list be reused frame after frame without allocating.
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

// List collects the commands of a frame. The zero value is an empty list ready to use.
type List struct {
	Commands []Command
	clips    []Rect
	layer    int
}

// Reset empties the list, keeping its memory for the next frame.
func (list *List) Reset() {
	list.Commands = list.Commands[:0]
	list.clips = list.clips[:0]
	list.layer = 0
}

// IsEmpty tells whether the list has nothing to draw.
func (list *List) IsEmpty() bool { return list == nil || len(list.Commands) == 0 }

// SetLayer makes the following commands go on a layer: for example 0 for panels, 1 for menus
// that open over them and 2 for tooltips.
func (list *List) SetLayer(layer int) { list.layer = layer }

// PushClip limits the following commands to a rectangle, inside the current clip if there is one.
func (list *List) PushClip(area Rect) {
	if clip, clipped := list.currentClip(); clipped {
		area = area.Intersect(clip)
	}
	list.clips = append(list.clips, area)
}

// PopClip returns to the clip before the last PushClip.
func (list *List) PopClip() {
	if len(list.clips) > 0 {
		list.clips = list.clips[:len(list.clips)-1]
	}
}

func (list *List) currentClip() (Rect, bool) {
	if len(list.clips) == 0 {
		return Rect{}, false
	}
	return list.clips[len(list.clips)-1], true
}

func (list *List) add(command Command) {
	command.Layer = list.layer
	command.Clip, command.HasClip = list.currentClip()
	if command.HasClip && isClippedAway(command) {
		return
	}
	list.Commands = append(list.Commands, command)
}

// isClippedAway tells whether nothing of a command can show inside its clip. A text's width is not
// known here, so a text goes only when its line misses the clip from above or below, or starts
// past its right edge: the rows of a scrolled list out of sight, which would otherwise be laid out
// glyph by glyph only to be thrown away.
func isClippedAway(command Command) bool {
	clip := command.Clip
	if command.Kind != TextKind {
		return clip.Intersect(command.Area).IsEmpty()
	}
	top, bottom := command.Baseline-command.Size, command.Baseline+command.Size/3
	return bottom <= clip.Y || top >= clip.Bottom() || command.Area.X >= clip.Right()
}

// Fingerprint sums up everything the list draws in one number: two lists with the same
// fingerprint draw the same picture (barring a hash collision). A window that draws on demand
// compares it with the last frame's to skip drawing a frame that would look the same.
func (list *List) Fingerprint(seed maphash.Seed) uint64 {
	var hash maphash.Hash
	hash.SetSeed(seed)
	var buffer [4 * 28]byte
	for index := range list.Commands {
		command := &list.Commands[index]
		values := [...]float32{
			float32(command.Kind), float32(command.Layer), float32(command.Weight), boolNumber(command.HasClip),
			command.Area.X, command.Area.Y, command.Area.Width, command.Area.Height,
			command.Clip.X, command.Clip.Y, command.Clip.Width, command.Clip.Height,
			command.Radius, command.BorderWidth, command.Blur, command.Size, command.Baseline,
			command.Fill[0], command.Fill[1], command.Fill[2], command.Fill[3],
			command.Border[0], command.Border[1], command.Border[2], command.Border[3],
		}
		for position, value := range values {
			binary.LittleEndian.PutUint32(buffer[position*4:], math.Float32bits(value))
		}
		hash.Write(buffer[:len(values)*4])
		hash.WriteString(command.Text)
		hash.WriteByte(0) // so "ab"+"c" and "a"+"bc" differ
		if command.Picture != nil {
			// the picture and its version: a video's next frame is a new picture to draw
			binary.LittleEndian.PutUint64(buffer[0:], command.Picture.id)
			binary.LittleEndian.PutUint64(buffer[8:], command.Picture.version)
			hash.Write(buffer[:16])
		}
	}
	return hash.Sum64()
}

func boolNumber(value bool) float32 {
	if value {
		return 1
	}
	return 0
}

// Rectangle draws a rounded rectangle: fill inside, and a border of a width drawn inside the area.
// A transparent fill or border is skipped.
func (list *List) Rectangle(area Rect, radius float32, fill, border Color, borderWidth float32) {
	if area.IsEmpty() || (!fill.IsVisible() && (!border.IsVisible() || borderWidth <= 0)) {
		return
	}
	list.add(Command{Kind: RectangleKind, Area: area, Radius: radius, Fill: fill, Border: border, BorderWidth: borderWidth})
}

// Shadow draws a soft shadow for a rounded rectangle, fading out over blur pixels on every side
// and falling offsetY pixels below it. Draw it before the rectangle that casts it.
func (list *List) Shadow(area Rect, radius, blur, offsetY float32, color Color) {
	if area.IsEmpty() || !color.IsVisible() {
		return
	}
	shadowed := Rect{X: area.X - blur, Y: area.Y - blur + offsetY, Width: area.Width + 2*blur, Height: area.Height + 2*blur}
	list.add(Command{Kind: ShadowKind, Area: shadowed, Radius: radius, Blur: blur, Fill: color})
}

// Text draws one line of text starting at x, with its baseline at baseline. Characters the font
// lacks are drawn as a question mark.
func (list *List) Text(text string, x, baseline, size float32, weight Weight, color Color) {
	if text == "" || size <= 0 || !color.IsVisible() {
		return
	}
	list.add(Command{Kind: TextKind, Text: text, Area: Rect{X: x, Y: baseline - size, Width: 0, Height: size},
		Baseline: baseline, Size: size, Weight: weight, Fill: color})
}

// Icon draws an icon by name (a Lucide icon, https://lucide.dev) filling a square of size pixels
// whose top-left corner is at (x, y).
func (list *List) Icon(name string, x, y, size float32, color Color) {
	if name == "" || size <= 0 || !color.IsVisible() {
		return
	}
	list.add(Command{Kind: IconKind, Text: name, Area: Rect{X: x, Y: y, Width: size, Height: size}, Size: size, Fill: color})
}

// Image draws a picture covering a rounded rectangle: scaled to fill it, and cropped evenly at the
// sides or at the top and bottom when their shapes differ, as photos fill cards and avatars. tint
// multiplies its colours; white draws it as it is, and a lower opacity fades it.
func (list *List) Image(picture *Image, area Rect, radius float32, tint Color) {
	if picture == nil || picture.Pixels == nil || area.IsEmpty() || !tint.IsVisible() {
		return
	}
	if width, height := picture.Size(); width == 0 || height == 0 {
		return
	}
	list.add(Command{Kind: ImageKind, Picture: picture, Area: area, Radius: radius, Fill: tint})
}
