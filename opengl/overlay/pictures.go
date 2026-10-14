package overlay

import (
	"image"
	"image/draw"

	"github.com/akrck02/go-renderer/draw2d"
	scaling "golang.org/x/image/draw"
)

// Pictures is the second atlas: the colours of the images a draw list shows (photos, a video's
// frames), side by side in one RGBA image. A picture's quads read their colours here and their
// coverage (the rounded corners) in the grey Atlas, so images are drawn in the same single draw
// call as everything else.
//
// A picture is copied in the first time it is drawn, and copied again into the same place when it
// changes (draw2d.Image.Changed). When no room is left, the image grows; at its largest, it is
// emptied and holds only the pictures of the frame being drawn. A backend uploads what changed:
// the whole image after it grew or was emptied, else only the areas pictures were copied into.
// The image is made with the first picture: an interface without any needs none.
type Pictures struct {
	Image *image.NRGBA

	entries                     map[uint64]*pictureEntry
	cursorX, cursorY, rowHeight int
	frame                       int
	whole                       bool              // the whole image is to be uploaded
	changed                     []image.Rectangle // areas to upload, when not the whole image
}

// pictureEntry is where a picture is, and which of its versions is there.
type pictureEntry struct {
	bounds    image.Rectangle // its pixels, without the border around them
	version   uint64
	lastFrame int
}

const (
	initialPicturesSize = 1024
	maximumPicturesSize = 4096
	// maximumPictureSide is the longest side a picture keeps: larger photos are scaled down to it,
	// which still fills a phone's screen across.
	maximumPictureSide = 2048
	// pictureBorder is copied from each picture's own edge around it, so linear filtering at the
	// edge reads the picture and not its neighbour.
	pictureBorder = 1
)

// NewPictures returns an empty picture atlas, with no image yet.
func NewPictures() *Pictures {
	return &Pictures{entries: map[uint64]*pictureEntry{}}
}

// Size returns the side of the square image, in pixels: 0 before the first picture.
func (pictures *Pictures) Size() int {
	if pictures.Image == nil {
		return 0
	}
	return pictures.Image.Rect.Dx()
}

// Pending tells what to upload: the whole image, or only some areas of it (none when nothing
// changed). The backend calls Uploaded once it has.
func (pictures *Pictures) Pending() (whole bool, areas []image.Rectangle) {
	return pictures.whole, pictures.changed
}

// Uploaded forgets what Pending told.
func (pictures *Pictures) Uploaded() {
	pictures.whole, pictures.changed = false, pictures.changed[:0]
}

// prepare makes sure every picture of a frame is in the atlas, in its latest version.
func (pictures *Pictures) prepare(frame []*draw2d.Image) {
	pictures.frame++
	for _, picture := range frame {
		if !pictures.keep(picture) {
			pictures.emptyFor(frame)
			return
		}
	}
}

// keep copies a picture in when it is new or changed; false when there is no room left.
func (pictures *Pictures) keep(picture *draw2d.Image) bool {
	entry, found := pictures.entries[picture.ID()]
	if found {
		entry.lastFrame = pictures.frame
		if entry.version != picture.Version() {
			pictures.copyInto(entry.bounds, picture)
			entry.version = picture.Version()
		}
		return true
	}
	width, height := keptSize(picture)
	bounds, placed := pictures.place(width, height)
	if !placed {
		return false
	}
	pictures.copyInto(bounds, picture)
	pictures.entries[picture.ID()] = &pictureEntry{bounds: bounds, version: picture.Version(), lastFrame: pictures.frame}
	return true
}

// emptyFor starts the image again with only the pictures of this frame.
func (pictures *Pictures) emptyFor(frame []*draw2d.Image) {
	clear(pictures.entries)
	pictures.cursorX, pictures.cursorY, pictures.rowHeight = 0, 0, 0
	pictures.whole, pictures.changed = true, pictures.changed[:0]
	for _, picture := range frame {
		if _, found := pictures.entries[picture.ID()]; !found {
			pictures.keep(picture) // what does not fit even now is not drawn
		}
	}
}

// keptSize is the size a picture has in the atlas: its own, or scaled down to the longest side.
func keptSize(picture *draw2d.Image) (width, height int) {
	width, height = picture.Size()
	if longest := max(width, height); longest > maximumPictureSide {
		width, height = max(width*maximumPictureSide/longest, 1), max(height*maximumPictureSide/longest, 1)
	}
	return width, height
}

// place finds room for a picture and its border (rows of pictures, left to right), growing the
// image when needed.
func (pictures *Pictures) place(width, height int) (image.Rectangle, bool) {
	outerWidth, outerHeight := width+2*pictureBorder, height+2*pictureBorder
	for !pictures.fits(outerWidth, outerHeight) {
		if !pictures.grow() {
			return image.Rectangle{}, false
		}
	}
	if pictures.cursorX+outerWidth > pictures.Size() {
		pictures.cursorX, pictures.cursorY, pictures.rowHeight = 0, pictures.cursorY+pictures.rowHeight, 0
	}
	origin := image.Pt(pictures.cursorX+pictureBorder, pictures.cursorY+pictureBorder)
	pictures.cursorX += outerWidth
	pictures.rowHeight = max(pictures.rowHeight, outerHeight)
	return image.Rectangle{Min: origin, Max: origin.Add(image.Pt(width, height))}, true
}

func (pictures *Pictures) fits(width, height int) bool {
	size := pictures.Size()
	if width > size {
		return false
	}
	if pictures.cursorX+width <= size {
		return pictures.cursorY+height <= size
	}
	return pictures.cursorY+pictures.rowHeight+height <= size
}

// grow makes the image, or doubles it, keeping every picture where it was.
func (pictures *Pictures) grow() bool {
	size := max(pictures.Size()*2, initialPicturesSize)
	if size > maximumPicturesSize {
		return false
	}
	larger := image.NewNRGBA(image.Rect(0, 0, size, size))
	if pictures.Image != nil {
		draw.Draw(larger, pictures.Image.Rect, pictures.Image, image.Point{}, draw.Src)
	}
	pictures.Image, pictures.whole = larger, true
	return true
}

// copyInto copies a picture's pixels into its place, scaled to it, and repeats its edge pixels in
// the border around it.
func (pictures *Pictures) copyInto(bounds image.Rectangle, picture *draw2d.Image) {
	source := picture.Pixels
	if bounds.Size() == source.Rect.Size() {
		draw.Draw(pictures.Image, bounds, source, source.Rect.Min, draw.Src)
	} else {
		scaling.BiLinear.Scale(pictures.Image, bounds, source, source.Rect, draw.Src, nil)
	}
	pictures.repeatEdges(bounds)
	if !pictures.whole {
		pictures.changed = append(pictures.changed, bounds.Inset(-pictureBorder))
	}
}

// repeatEdges fills the border around a picture with the picture's own outermost pixels.
func (pictures *Pictures) repeatEdges(bounds image.Rectangle) {
	atlas := pictures.Image
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		atlas.SetNRGBA(x, bounds.Min.Y-1, atlas.NRGBAAt(x, bounds.Min.Y))
		atlas.SetNRGBA(x, bounds.Max.Y, atlas.NRGBAAt(x, bounds.Max.Y-1))
	}
	for y := bounds.Min.Y - 1; y <= bounds.Max.Y; y++ {
		atlas.SetNRGBA(bounds.Min.X-1, y, atlas.NRGBAAt(bounds.Min.X, y))
		atlas.SetNRGBA(bounds.Max.X, y, atlas.NRGBAAt(bounds.Max.X-1, y))
	}
}

// bounds tells where a picture of this frame is.
func (pictures *Pictures) bounds(picture *draw2d.Image) (image.Rectangle, bool) {
	entry, found := pictures.entries[picture.ID()]
	if !found || entry.lastFrame != pictures.frame {
		return image.Rectangle{}, false
	}
	return entry.bounds, true
}
