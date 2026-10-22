package input

import "testing"

func TestAPadsLayoutIsReadFromTheVendorInItsGUID(t *testing.T) {
	cases := map[string]PadLayout{
		"030000004c050000cc09000011810000": PlayStationLayout, // Sony
		"050000007e0500000920000001800000": NintendoLayout,    // Nintendo
		"030000005e0400008e02000010010000": XboxLayout,        // Microsoft
		"03000000d62000000228000001010000": XboxLayout,        // any other maker
		"4c05":                             XboxLayout,        // too short to hold a vendor
		"":                                 XboxLayout,
		"03000000zz050000cc09000011810000": XboxLayout, // not hexadecimal
	}
	for guid, want := range cases {
		if got := PadLayoutFromGUID(guid); got != want {
			t.Errorf("PadLayoutFromGUID(%q) = %v, want %v", guid, got, want)
		}
	}
}

func TestAPadsLayoutIsReadFromTheVendorInTheBrowsersName(t *testing.T) {
	cases := map[string]PadLayout{
		"Wireless Controller (STANDARD GAMEPAD Vendor: 054c Product: 09cc)": PlayStationLayout, // Chrome
		"054c-09cc-Wireless Controller":                                     PlayStationLayout, // Firefox
		"Pro Controller (STANDARD GAMEPAD Vendor: 057E Product: 2009)":      NintendoLayout,
		"57e-2009-Pro Controller":                                           NintendoLayout, // Firefox drops leading zeros
		"Xbox 360 Controller (XInput STANDARD GAMEPAD)":                     XboxLayout,
		"045e-028e-Microsoft X-Box 360 pad":                                 XboxLayout,
		"Vendor: ":                                                          XboxLayout,
		"":                                                                  XboxLayout,
	}
	for name, want := range cases {
		if got := PadLayoutFromName(name); got != want {
			t.Errorf("PadLayoutFromName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestAPadIsReadAsAnXboxPadUntilTheBackendSaysOtherwise(t *testing.T) {
	state := NewState()
	if state.PadLayout() != XboxLayout {
		t.Fatal("a new state does not read its pad as an Xbox pad")
	}
	state.SetPadLayout(NintendoLayout)
	if state.PadLayout() != NintendoLayout {
		t.Fatal("the layout set is not the layout read")
	}
}
