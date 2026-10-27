package overlay

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/akrck02/go-renderer/draw2d"
)

const wave = '🌊'

// colourTestFont makes a small colour emoji font, as Android's is made: a cmap of format 12 with
// one character, a CBLC strike of 8 pixels per em listing one glyph (index format 1), and its
// picture in CBDT (image format 17): a red PNG with small metrics.
func colourTestFont(test *testing.T) []byte {
	test.Helper()
	return colourTestFontOf(test, wave)
}

// colourTestFontOf makes the font with one character's picture.
func colourTestFontOf(test *testing.T, character rune) []byte {
	test.Helper()
	picture := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			picture.Set(x, y, color.NRGBA{255, 0, 0, 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		test.Fatal(err)
	}
	big := binary.BigEndian
	// cmap: one record for Unicode's full repertoire, one group
	characterMap := []byte{0, 0, 0, 1, 0, 3, 0, 10, 0, 0, 0, 12}
	group := make([]byte, 28)
	big.PutUint16(group, 12)
	big.PutUint32(group[4:], 28)
	big.PutUint32(group[12:], 1)
	big.PutUint32(group[16:], uint32(character))
	big.PutUint32(group[20:], uint32(character))
	big.PutUint32(group[24:], 1)
	characterMap = append(characterMap, group...)
	// CBDT: its version, then the picture: height, width, bearing x, bearing y, advance, length
	record := []byte{8, 8, 0, 7, 9, 0, 0, 0, 0}
	big.PutUint32(record[5:], uint32(encoded.Len()))
	record = append(record, encoded.Bytes()...)
	bitmapData := append([]byte{0, 3, 0, 0}, record...)
	// CBLC: its version and one strike, then the strike's index array and its one subtable
	lookup := make([]byte, 8+48+8+16)
	big.PutUint16(lookup, 3)
	big.PutUint32(lookup[4:], 1)
	strike := lookup[8:]
	big.PutUint32(strike, 56)
	big.PutUint32(strike[4:], 24)
	big.PutUint32(strike[8:], 1)
	big.PutUint16(strike[40:], 1)
	big.PutUint16(strike[42:], 1)
	strike[44], strike[45], strike[46] = 8, 8, 32
	array := lookup[56:]
	big.PutUint16(array, 1)
	big.PutUint16(array[2:], 1)
	big.PutUint32(array[4:], 8)
	subtable := lookup[64:]
	big.PutUint16(subtable, 1)
	big.PutUint16(subtable[2:], 17)
	big.PutUint32(subtable[4:], 4)
	big.PutUint32(subtable[8:], 0)
	big.PutUint32(subtable[12:], uint32(len(record)))
	return fontFileOf(map[string][]byte{"CBDT": bitmapData, "CBLC": lookup, "cmap": characterMap})
}

// fontFileOf puts tables in a font file, after its directory.
func fontFileOf(tables map[string][]byte) []byte {
	tags := []string{"CBDT", "CBLC", "cmap"}
	data := make([]byte, 12+16*len(tags))
	binary.BigEndian.PutUint32(data, 0x00010000)
	binary.BigEndian.PutUint16(data[4:], uint16(len(tags)))
	for index, tag := range tags {
		record := data[12+16*index:]
		copy(record, tag)
		binary.BigEndian.PutUint32(record[8:], uint32(len(data)))
		binary.BigEndian.PutUint32(record[12:], uint32(len(tables[tag])))
		data = append(data, tables[tag]...)
	}
	return data
}

func TestAColourFontsPictureIsFoundWithItsMetrics(test *testing.T) {
	font, err := parseColourFont(colourTestFont(test))
	if err != nil {
		test.Fatal(err)
	}
	picture, found := font.picture(wave)
	if !found || picture.width != 1 || picture.height != 1 || picture.top != 7.0/8 || picture.advance != 9.0/8 {
		test.Fatalf("the wave, an em square from 7/8 above the baseline: %+v", picture)
	}
	decoded, err := picture.decode()
	if err != nil {
		test.Fatal(err)
	}
	if red, _, _, _ := decoded.At(1, 1).RGBA(); red != 0xffff {
		test.Fatal("its own colours")
	}
	if _, found := font.picture('😀'); found {
		test.Fatal("a character the font lacks has no picture")
	}
	if _, err := parseColourFont(fontFileOf(map[string][]byte{"cmap": nil})); err == nil {
		test.Fatal("a font without bitmap tables is no colour font")
	}
}

func TestAnEmojiIsMeasuredAndDrawnInItsColours(test *testing.T) {
	fonts := withFontFolder(test, map[string][]byte{"NotoColorEmoji.ttf": colourTestFont(test)})
	measurer, err := NewMeasurer(fonts)
	if err != nil {
		test.Fatal(err)
	}
	letter := measurer.MeasureText("a", 16, draw2d.Regular, 1).Width
	// the variation selector asking for the picture is not drawn
	if width := measurer.MeasureText("a🌊️", 16, draw2d.Regular, 1).Width; width != letter+18 {
		test.Fatalf("the wave moves the pen 9/8 of an em: %v, want %v", width, letter+18)
	}
	var list draw2d.List
	list.Text("🌊", 10, 30, 16, draw2d.Regular, white)
	quads := paintListIn(test, fonts, &list, 1)
	var pictured []Quad
	for _, quad := range quads {
		if quad.Pictured {
			pictured = append(pictured, quad)
		}
	}
	if bounds := covered(pictured); len(pictured) == 0 || bounds != [4]float32{10, 16, 26, 32} {
		test.Fatalf("a 16-pixel picture from 14 pixels above the baseline: %v", bounds)
	}
}

func TestEveryColourFontIsLookedIn(test *testing.T) {
	flags := colourTestFontOf(test, '🇦') // a font that reads, with a flag's letter only
	fonts := withFontFolder(test, map[string][]byte{"NotoColorEmojiFlags.ttf": flags, "NotoColorEmojiLegacy.ttf": colourTestFont(test)})
	_, systemEmoji := systemFontsIn(fonts.Folders)
	if _, found := systemEmoji.glyph(wave, 16); !found {
		test.Fatal("the wave is in the second colour font")
	}
}

// Hiding a font by renaming it, as one does to see how a phone without it behaves, must hide it.
func TestARenamedEmojiFontIsNotRead(test *testing.T) {
	fonts := withFontFolder(test, map[string][]byte{"NotoColorEmoji.ttf.hidden": colourTestFont(test)})
	_, systemEmoji := systemFontsIn(fonts.Folders)
	if _, found := systemEmoji.glyph(wave, 16); found {
		test.Fatal("a file that is not named as a font was read as one")
	}
}

func TestASelectorAsksForThePictureOverAnOutline(test *testing.T) {
	// a character the Go fonts draw, with a picture in the colour font too
	fonts := withFontFolder(test, map[string][]byte{"NotoColorEmoji.ttf": colourTestFontOf(test, '©')})
	measurer, err := NewMeasurer(fonts)
	if err != nil {
		test.Fatal(err)
	}
	outline := measurer.MeasureText("©", 16, draw2d.Regular, 1).Width
	if picture := measurer.MeasureText("©️", 16, draw2d.Regular, 1).Width; picture == outline || picture != 18 {
		test.Fatalf("followed by U+FE0F it is the picture: %v, the outline %v", picture, outline)
	}
}
