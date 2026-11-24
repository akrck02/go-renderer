package svg

import (
	"testing"
	"time"
)

// patience is how long a picture of a few pixels may take before a test calls it stuck
const patience = 5 * time.Second

// Icon themes come from anyone, so a file with numbers no icon needs must still give a picture,
// quickly, and never stop the program that draws it.
func TestNumbersOutOfRangeNeitherStopNorHangTheDrawing(test *testing.T) {
	sources := map[string]string{
		"an arc of a huge radius":                  document(`<path d="M0 0 A1e200 1e200 0 0 1 10 0"/>`),
		"an arc of a tiny radius":                  document(`<path d="M0 0 A1e-200 1e-200 0 0 1 10 0"/>`),
		"a stroke of infinite width":               stroked("M2 2 L8 8", `stroke-width="Inf"`),
		"a view box that is no number":             `<svg viewBox="NaN 0 24 24"><rect width="4" height="4"/></svg>`,
		"dashes too small to see":                  stroked("M0 0 L10 10", `stroke-dasharray="1e-300"`),
		"dashes along a line far past the picture": stroked("M0 0 L1e12 0", `stroke-dasharray="1"`),
	}
	for name, source := range sources {
		drawn := make(chan error, 1)
		go func() {
			defer func() {
				if stop := recover(); stop != nil {
					drawn <- nil
					test.Errorf("%s stops the drawing: %v", name, stop)
				}
			}()
			drawing, err := Read([]byte(source))
			if err == nil {
				drawing.Picture(16, 16)
			}
			drawn <- err
		}()
		select {
		case <-drawn:
		case <-time.After(patience):
			test.Fatalf("%s is still drawing after %v", name, patience)
		}
	}
}
