package overlay

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/png"
)

// A colour emoji font, as Android's NotoColorEmoji, has no outlines: each emoji is a PNG picture,
// kept in the font's CBDT table and found through its CBLC table, at a few sizes (strikes). This
// reads them, so an emoji is drawn as a picture in its own colours. Only what a colour font needs
// is read: the table directory, the character map (cmap) and the two bitmap tables.

// colourFont is a parsed colour emoji font.
type colourFont struct {
	characters   characterMap
	strike       bitmapStrike // the largest, scaled down to the size drawn
	bitmapData   []byte       // the CBDT table
	bitmapLookup []byte       // the CBLC table
}

// bitmapStrike is one size of a colour font's pictures: which glyphs it has and where their
// pictures are listed, and its pixels per em.
type bitmapStrike struct {
	indexArray      []byte // the strike's index subtable array, within CBLC
	indexCount      int
	pixelsPerEm     float32
	first, last     uint16
	indexArrayStart int
}

// colourPicture is an emoji's PNG and where it sits, in ems: Left from the pen, Top above the
// baseline, its Width and Height, and how far the pen moves after it.
type colourPicture struct {
	png                               []byte
	left, top, width, height, advance float32
}

var errNoColourTables = errors.New("no colour bitmap tables (CBDT and CBLC)")

// parseColourFont reads the tables of a colour emoji font.
func parseColourFont(data []byte) (*colourFont, error) {
	tables, err := tableDirectory(data)
	if err != nil {
		return nil, err
	}
	bitmapData, bitmapLookup, characterTable := tables["CBDT"], tables["CBLC"], tables["cmap"]
	if bitmapData == nil || bitmapLookup == nil {
		return nil, errNoColourTables
	}
	characters, err := parseCharacterMap(characterTable)
	if err != nil {
		return nil, err
	}
	strike, err := largestStrike(bitmapLookup)
	if err != nil {
		return nil, err
	}
	return &colourFont{characters: characters, strike: strike, bitmapData: bitmapData, bitmapLookup: bitmapLookup}, nil
}

// tableDirectory finds a font's tables by their tags.
func tableDirectory(data []byte) (map[string][]byte, error) {
	if len(data) < 12 {
		return nil, errors.New("too short for a font")
	}
	count := int(binary.BigEndian.Uint16(data[4:]))
	tables := map[string][]byte{}
	for index := range count {
		record := 12 + 16*index
		if record+16 > len(data) {
			return nil, errors.New("the table directory runs past the end")
		}
		offset, length := int(binary.BigEndian.Uint32(data[record+8:])), int(binary.BigEndian.Uint32(data[record+12:]))
		if offset < 0 || length < 0 || offset+length > len(data) {
			continue
		}
		tables[string(data[record:record+4])] = data[offset : offset+length]
	}
	return tables, nil
}

// largestStrike is the CBLC table's strike with the most pixels per em: scaled down, it looks
// best at every size.
func largestStrike(table []byte) (bitmapStrike, error) {
	if len(table) < 8 {
		return bitmapStrike{}, errors.New("CBLC too short")
	}
	count := int(binary.BigEndian.Uint32(table[4:]))
	var best bitmapStrike
	for index := range count {
		record := 8 + 48*index
		if record+48 > len(table) {
			break
		}
		start := int(binary.BigEndian.Uint32(table[record:]))
		strike := bitmapStrike{
			indexArrayStart: start,
			indexCount:      int(binary.BigEndian.Uint32(table[record+8:])),
			first:           binary.BigEndian.Uint16(table[record+40:]),
			last:            binary.BigEndian.Uint16(table[record+42:]),
			pixelsPerEm:     float32(table[record+45]), // ppemY
		}
		if start < 0 || start+8*strike.indexCount > len(table) {
			continue
		}
		strike.indexArray = table[start : start+8*strike.indexCount]
		if strike.pixelsPerEm > best.pixelsPerEm {
			best = strike
		}
	}
	if best.pixelsPerEm == 0 {
		return bitmapStrike{}, errors.New("CBLC has no strike")
	}
	return best, nil
}

// picture finds a character's PNG and metrics, if the font has a picture for it.
func (font *colourFont) picture(character rune) (colourPicture, bool) {
	glyph := font.characters.glyph(character)
	if glyph == 0 || glyph < font.strike.first || glyph > font.strike.last {
		return colourPicture{}, false
	}
	for index := range font.strike.indexCount {
		entry := font.strike.indexArray[8*index:]
		first, last := binary.BigEndian.Uint16(entry), binary.BigEndian.Uint16(entry[2:])
		if glyph < first || glyph > last {
			continue
		}
		subtable := font.strike.indexArrayStart + int(binary.BigEndian.Uint32(entry[4:]))
		return font.pictureIn(subtable, first, glyph)
	}
	return colourPicture{}, false
}

// pictureIn reads a glyph's picture through an index subtable, in one of its five formats.
func (font *colourFont) pictureIn(subtable int, first, glyph uint16) (colourPicture, bool) {
	lookup := font.bitmapLookup
	if subtable < 0 || subtable+8 > len(lookup) {
		return colourPicture{}, false
	}
	indexFormat := binary.BigEndian.Uint16(lookup[subtable:])
	imageFormat := binary.BigEndian.Uint16(lookup[subtable+2:])
	imageData := int(binary.BigEndian.Uint32(lookup[subtable+4:]))
	body := lookup[subtable+8:]
	position := int(glyph - first)
	var offset int
	var metrics []byte // big metrics shared by every glyph of the subtable (formats 2 and 5)
	switch indexFormat {
	case 1:
		if 4*position+4 > len(body) {
			return colourPicture{}, false
		}
		offset = int(binary.BigEndian.Uint32(body[4*position:]))
	case 3:
		if 2*position+2 > len(body) {
			return colourPicture{}, false
		}
		offset = int(binary.BigEndian.Uint16(body[2*position:]))
	case 2:
		if len(body) < 12 {
			return colourPicture{}, false
		}
		offset, metrics = int(binary.BigEndian.Uint32(body))*position, body[4:12]
	case 4:
		found := false
		offset, found = sparseOffset(body, glyph)
		if !found {
			return colourPicture{}, false
		}
	case 5:
		found := false
		offset, metrics, found = sparseConstantOffset(body, glyph)
		if !found {
			return colourPicture{}, false
		}
	default:
		return colourPicture{}, false
	}
	return font.readPicture(imageData+offset, imageFormat, metrics)
}

// sparseOffset finds a glyph in an index subtable of format 4: pairs of a glyph and its offset.
func sparseOffset(body []byte, glyph uint16) (int, bool) {
	if len(body) < 4 {
		return 0, false
	}
	count := int(binary.BigEndian.Uint32(body))
	for index := range count {
		pair := 4 + 4*index
		if pair+4 > len(body) {
			return 0, false
		}
		if binary.BigEndian.Uint16(body[pair:]) == glyph {
			return int(binary.BigEndian.Uint16(body[pair+2:])), true
		}
	}
	return 0, false
}

// sparseConstantOffset finds a glyph in an index subtable of format 5: pictures of one size, for
// a list of glyphs.
func sparseConstantOffset(body []byte, glyph uint16) (int, []byte, bool) {
	if len(body) < 16 {
		return 0, nil, false
	}
	size, metrics := int(binary.BigEndian.Uint32(body)), body[4:12]
	count := int(binary.BigEndian.Uint32(body[12:]))
	for index := range count {
		at := 16 + 2*index
		if at+2 > len(body) {
			break
		}
		if binary.BigEndian.Uint16(body[at:]) == glyph {
			return size * index, metrics, true
		}
	}
	return 0, nil, false
}

// readPicture reads a picture in CBDT: small or big metrics, then the PNG's length and the PNG.
func (font *colourFont) readPicture(offset int, format uint16, shared []byte) (colourPicture, bool) {
	data := font.bitmapData
	if offset < 0 || offset >= len(data) {
		return colourPicture{}, false
	}
	record := data[offset:]
	var picture colourPicture
	var rest []byte
	switch format {
	case 17: // small metrics: height, width, bearing x, bearing y, advance
		if len(record) < 9 {
			return colourPicture{}, false
		}
		picture = font.inEms(record[1], record[0], int8(record[2]), int8(record[3]), record[4])
		rest = record[5:]
	case 18: // big metrics: height, width, then the horizontal bearings and advance
		if len(record) < 12 {
			return colourPicture{}, false
		}
		picture = font.inEms(record[1], record[0], int8(record[2]), int8(record[3]), record[4])
		rest = record[8:]
	case 19: // the subtable's metrics
		if len(shared) < 8 {
			return colourPicture{}, false
		}
		picture = font.inEms(shared[1], shared[0], int8(shared[2]), int8(shared[3]), shared[4])
		rest = record
	default:
		return colourPicture{}, false
	}
	if len(rest) < 4 {
		return colourPicture{}, false
	}
	length := int(binary.BigEndian.Uint32(rest))
	if length <= 0 || 4+length > len(rest) {
		return colourPicture{}, false
	}
	picture.png = rest[4 : 4+length]
	return picture, true
}

// inEms turns a picture's metrics, in the strike's pixels, into ems.
func (font *colourFont) inEms(width, height uint8, left, top int8, advance uint8) colourPicture {
	perEm := font.strike.pixelsPerEm
	return colourPicture{left: float32(left) / perEm, top: float32(top) / perEm, width: float32(width) / perEm,
		height: float32(height) / perEm, advance: float32(advance) / perEm}
}

// decode reads a picture's PNG.
func (picture colourPicture) decode() (image.Image, error) {
	return png.Decode(bytes.NewReader(picture.png))
}

// characterMap finds a character's glyph: a font's cmap subtable of format 12 (every plane) or 4
// (the first plane).
type characterMap struct {
	groups   []byte // format 12: groups of 12 bytes, sorted
	segments []byte // format 4: the whole subtable
}

func parseCharacterMap(table []byte) (characterMap, error) {
	if len(table) < 4 {
		return characterMap{}, errors.New("no cmap")
	}
	count := int(binary.BigEndian.Uint16(table[2:]))
	var found characterMap
	for index := range count {
		record := 4 + 8*index
		if record+8 > len(table) {
			break
		}
		platform, encoding := binary.BigEndian.Uint16(table[record:]), binary.BigEndian.Uint16(table[record+2:])
		offset := int(binary.BigEndian.Uint32(table[record+4:]))
		if offset+4 > len(table) || !(platform == 3 || platform == 0) {
			continue
		}
		subtable := table[offset:]
		switch binary.BigEndian.Uint16(subtable) {
		case 12:
			if len(subtable) < 16 {
				continue
			}
			groups := int(binary.BigEndian.Uint32(subtable[12:]))
			if 16+12*groups <= len(subtable) {
				found.groups = subtable[16 : 16+12*groups]
			}
		case 4:
			if encoding == 1 || platform == 0 {
				found.segments = subtable
			}
		}
	}
	if found.groups == nil && found.segments == nil {
		return characterMap{}, errors.New("no Unicode cmap of format 12 or 4")
	}
	return found, nil
}

// glyph is a character's glyph, or 0 when the font has none.
func (characters characterMap) glyph(character rune) uint16 {
	if characters.groups != nil {
		return groupGlyph(characters.groups, uint32(character))
	}
	if character > 0xffff {
		return 0
	}
	return segmentGlyph(characters.segments, uint16(character))
}

// groupGlyph searches format 12's sorted groups: a run of characters and its first glyph.
func groupGlyph(groups []byte, character uint32) uint16 {
	low, high := 0, len(groups)/12-1
	for low <= high {
		middle := (low + high) / 2
		group := groups[12*middle:]
		start, end := binary.BigEndian.Uint32(group), binary.BigEndian.Uint32(group[4:])
		switch {
		case character < start:
			high = middle - 1
		case character > end:
			low = middle + 1
		default:
			return uint16(binary.BigEndian.Uint32(group[8:]) + character - start)
		}
	}
	return 0
}

// segmentGlyph reads format 4: segments of characters, each with a delta or a range of glyphs.
func segmentGlyph(subtable []byte, character uint16) uint16 {
	if len(subtable) < 14 {
		return 0
	}
	segments := int(binary.BigEndian.Uint16(subtable[6:])) / 2
	ends := 14
	starts := ends + 2*segments + 2
	deltas := starts + 2*segments
	ranges := deltas + 2*segments
	if ranges+2*segments > len(subtable) {
		return 0
	}
	for segment := range segments {
		end := binary.BigEndian.Uint16(subtable[ends+2*segment:])
		if character > end {
			continue
		}
		start := binary.BigEndian.Uint16(subtable[starts+2*segment:])
		if character < start {
			return 0
		}
		delta := binary.BigEndian.Uint16(subtable[deltas+2*segment:])
		rangeOffset := int(binary.BigEndian.Uint16(subtable[ranges+2*segment:]))
		if rangeOffset == 0 {
			return character + delta
		}
		at := ranges + 2*segment + rangeOffset + 2*int(character-start)
		if at+2 > len(subtable) {
			return 0
		}
		glyph := binary.BigEndian.Uint16(subtable[at:])
		if glyph == 0 {
			return 0
		}
		return glyph + delta
	}
	return 0
}
