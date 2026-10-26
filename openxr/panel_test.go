package openxr

import (
	"testing"

	"github.com/akrck02/go-renderer/gles"
)

func TestFittingAPanelsWidthKeepsItsPixelsSquare(test *testing.T) {
	panel := &Panel{width: 800, height: 600}
	panel.FitWidth(1.2)
	if panel.WidthMetres != 1.2 || panel.HeightMetres < 0.8999 || panel.HeightMetres > 0.9001 {
		test.Fatalf("an 800 × 600 panel 1.2 m wide is %v × %v m, want 1.2 × 0.9", panel.WidthMetres, panel.HeightMetres)
	}
}

func TestAPanelShowsOnlyWhenShownWithASize(test *testing.T) {
	panel := &Panel{width: 100, height: 100}
	if panel.shows() {
		test.Fatal("a panel nobody showed goes in the frame")
	}
	panel.Shown = true
	if panel.shows() {
		test.Fatal("a panel of no size goes in the frame")
	}
	panel.FitWidth(1)
	if !panel.shows() {
		test.Fatal("a shown panel with a size is left out of the frame")
	}
}

func TestAPanelWithoutASessionCannotBeDrawn(test *testing.T) {
	if _, err := DetachedPanel(10, 10).Target(); err != PanelNotDrawable {
		test.Fatalf("drawing a panel no session made gave %v, want PanelNotDrawable", err)
	}
	drawn := &Panel{acquire: func() (gles.Framebuffer, error) { return 7, nil }}
	if target, err := drawn.Target(); err != nil || target != 7 {
		test.Fatalf("the panel's target is %v (%v), want the session's 7", target, err)
	}
}

func TestAPanelWithItsOwnTargetIsDrawnWhereItSays(test *testing.T) {
	asked := 0
	panel := PanelWithTarget(640, 320, func() (gles.Framebuffer, error) { asked++; return 9, nil })
	if panel.Width() != 640 || panel.Height() != 320 {
		test.Fatalf("the panel is %d × %d pixels, want 640 × 320", panel.Width(), panel.Height())
	}
	if target, err := panel.Target(); err != nil || target != 9 || asked != 1 {
		test.Fatalf("the panel's target is %v (%v, asked %d times), want 9 from its own", target, err, asked)
	}
}
