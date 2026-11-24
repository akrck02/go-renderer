//go:build linux && !android

package overlay

import "testing"

func TestAGnomeFontSettingIsReadAsItsFamily(test *testing.T) {
	for setting, family := range map[string]string{
		"'Adwaita Sans 11'\n": "Adwaita Sans",
		"'Cantarell 10.5'":    "Cantarell",
		"'Noto Sans Mono'":    "Noto Sans Mono",
		"'Inter'":             "Inter",
	} {
		if got := familyOf(setting); got != family {
			test.Errorf("the setting %q is read as %q, not %q", setting, got, family)
		}
	}
}
