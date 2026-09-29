package scene

import "testing"

func TestLevelForScreenSizePicksTheCoarsestThatFits(t *testing.T) {
	low, far := &Mesh{Name: "low"}, &Mesh{Name: "far"}
	mesh := &Mesh{Details: []DetailLevel{{Mesh: low, ScreenSize: 0.05}, {Mesh: far, ScreenSize: 0.01}}}
	cases := map[float32]int{0.2: 0, 0.05: 0, 0.03: 1, 0.005: 2}
	for screenSize, expected := range cases {
		if level := mesh.LevelForScreenSize(screenSize); level != expected {
			t.Errorf("screen size %v: level %d, expected %d", screenSize, level, expected)
		}
	}
	if mesh.Level(2) != far || mesh.Level(0) != mesh || mesh.Level(9) != far {
		t.Errorf("Level returns the wrong mesh")
	}
}
