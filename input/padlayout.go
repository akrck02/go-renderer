package input

import "strings"

// A pad's face buttons are read by position (PadA is always the bottom one), but each maker prints
// other names on them: the bottom button is A on an Xbox pad, ✕ on a PlayStation pad and B on a
// Nintendo pad. The layout tells a game which names the pad in the player's hands shows. It is
// worked out from the USB vendor of the pad, which the system gives on every platform, while a
// pad's name varies by driver.

// PadLayout is the naming of a pad's buttons, after the maker of the pad.
type PadLayout int

const (
	// XboxLayout is the default: an Xbox pad's names, the layout most pads copy.
	XboxLayout PadLayout = iota
	// PlayStationLayout names the face buttons ✕ ○ □ △.
	PlayStationLayout
	// NintendoLayout swaps A with B and X with Y, as a Nintendo pad prints them.
	NintendoLayout
)

// The USB vendors whose pads have a layout of their own.
const (
	sonyVendor     = 0x054c
	nintendoVendor = 0x057e
)

// guidVendorStart is where the vendor starts in an SDL-style GUID, as hexadecimal digits: after
// the bus (2 bytes) and the checksum (2 bytes), as 2 bytes little-endian.
const guidVendorStart = 8

// browserVendorLabel comes before the vendor in Chrome's name of a pad.
const browserVendorLabel = "Vendor: "

// PadLayout is the layout of the pad being read; XboxLayout until the backend sets another.
func (state *State) PadLayout() PadLayout { return state.padLayout }

// SetPadLayout records the layout of the pad being read; the backend sets it when a pad connects.
func (state *State) SetPadLayout(layout PadLayout) { state.padLayout = layout }

// PadLayoutFromVendor is the layout of a pad made by a USB vendor.
func PadLayoutFromVendor(vendor uint16) PadLayout {
	switch vendor {
	case sonyVendor:
		return PlayStationLayout
	case nintendoVendor:
		return NintendoLayout
	}
	return XboxLayout
}

// PadLayoutFromGUID is the layout of a pad from its SDL-style GUID (as GLFW gives it), whose bytes
// 4 and 5 are the vendor, little-endian: "030000004c05…" is a Sony pad. A GUID it cannot read
// gives XboxLayout.
func PadLayoutFromGUID(guid string) PadLayout {
	if len(guid) < guidVendorStart+4 {
		return XboxLayout
	}
	low, lowFound := parseHexadecimal(guid[guidVendorStart : guidVendorStart+2])
	high, highFound := parseHexadecimal(guid[guidVendorStart+2 : guidVendorStart+4])
	if !lowFound || !highFound {
		return XboxLayout
	}
	return PadLayoutFromVendor(uint16(high<<8 | low))
}

// PadLayoutFromName is the layout of a pad from the name a browser gives it (the Gamepad API's
// id), which holds the vendor: Chrome writes "… Vendor: 054c Product: 09cc)", Firefox starts with
// "054c-09cc-…". A name without a vendor gives XboxLayout.
func PadLayoutFromName(name string) PadLayout {
	if start := strings.Index(name, browserVendorLabel); start >= 0 {
		return layoutFromLeadingVendor(name[start+len(browserVendorLabel):], " ")
	}
	return layoutFromLeadingVendor(name, "-")
}

// layoutFromLeadingVendor reads the vendor as the hexadecimal digits text starts with, up to end.
func layoutFromLeadingVendor(text, end string) PadLayout {
	digits, _, _ := strings.Cut(text, end)
	vendor, isNumber := parseHexadecimal(digits)
	if !isNumber || vendor > 0xffff {
		return XboxLayout
	}
	return PadLayoutFromVendor(uint16(vendor))
}

// parseHexadecimal reads up to 4 hexadecimal digits without allocating; found is false for
// anything else.
func parseHexadecimal(digits string) (value uint32, found bool) {
	if len(digits) == 0 || len(digits) > 4 {
		return 0, false
	}
	for index := 0; index < len(digits); index++ {
		digit, isDigit := hexadecimalDigit(digits[index])
		if !isDigit {
			return 0, false
		}
		value = value<<4 | digit
	}
	return value, true
}

// hexadecimalDigit is the value of one hexadecimal digit, in either case.
func hexadecimalDigit(character byte) (uint32, bool) {
	switch {
	case character >= '0' && character <= '9':
		return uint32(character - '0'), true
	case character >= 'a' && character <= 'f':
		return uint32(character-'a') + 10, true
	case character >= 'A' && character <= 'F':
		return uint32(character-'A') + 10, true
	}
	return 0, false
}
