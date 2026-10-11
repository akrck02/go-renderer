package loaders

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSceneReadsTheSky(t *testing.T) {
	directory := t.TempDir()
	writeTestPanorama(t, filepath.Join(directory, "night.png"))
	document := `{"asset": {"version": "2.0"}, "scene": 0, "scenes": [{"nodes": [], "extras": {"environment": {"sky": {
		"stars": {"density": 0.6, "daytimeVisibility": 0.2},
		"moon": {"size": 2, "phase": 0.25},
		"clouds": {"coverage": 0.4},
		"constellations": [{"name": "arc", "stars": [[0, 1, 0], [0.2, 1, 0]], "lines": [[0, 1], [0, 7]]}],
		"nightImage": "night.png"}}}}]}`
	path := filepath.Join(directory, "sky.gltf")
	if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	world, err := LoadScene(path)
	if err != nil {
		t.Fatal(err)
	}
	sky := world.Environment.Sky
	if sky.Stars.Density != 0.6 || sky.Stars.DaytimeVisibility != 0.2 || sky.Clouds.Coverage != 0.4 {
		t.Fatalf("stars or clouds not read: %+v %+v", sky.Stars, sky.Clouds)
	}
	if sky.Moon == nil || sky.Moon.Phase != 0.25 || sky.Moon.AngularSize < 0.034 || sky.Moon.AngularSize > 0.036 {
		t.Fatalf("moon not read: %+v", sky.Moon)
	}
	if len(sky.Constellations) != 1 || len(sky.Constellations[0].Lines) != 2 {
		t.Fatalf("constellations not read: %+v", sky.Constellations)
	}
	if sky.NightImage == nil || sky.NightImage.Equirectangular.Bounds().Dx() != 4 {
		t.Fatalf("night image not loaded")
	}
}

func writeTestPanorama(t *testing.T, path string) {
	panorama := image.NewRGBA(image.Rect(0, 0, 4, 2))
	panorama.Set(1, 1, color.RGBA{255, 255, 255, 255})
	output, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	if err := png.Encode(output, panorama); err != nil {
		t.Fatal(err)
	}
}
