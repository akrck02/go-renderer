package loaders

import (
	"fmt"
	"image"
	_ "image/jpeg" // sky images may be JPEG
	_ "image/png"  // or PNG
	"math"
	"os"
	"path/filepath"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// applySky reads the "sky" object of the environment extras, when present:
//
//	nightZenith, nightHorizon, nightAmbient, twilightColor, celestialPole: [x, y, z(, w)]
//	stars: {density, brightness, twinkle, daytimeVisibility}
//	moon: {direction, size (degrees), color, phase, light} or false for no moon
//	clouds: {coverage, color, speed, scale}
//	constellations: [{name, stars: [[x, y, z], ...], lines: [[first, second], ...], color}]
//	dayImage, nightImage: "panorama.png" or {faces: [+X, -X, +Y, -Y, +Z, -Z]} (paths relative to the file)
func (file *gltfFile) applySky(environment *scene.Environment, extras map[string]any) error {
	definition, found := extras["sky"].(map[string]any)
	if !found {
		return nil
	}
	sky := scene.DefaultSky()
	applySkyColors(sky, definition)
	applyStars(&sky.Stars, definition)
	applyMoon(sky, definition)
	applyClouds(&sky.Clouds, definition)
	sky.Constellations = constellationsFromExtras(definition["constellations"])
	var err error
	if sky.DayImage, err = file.skyImageFromExtras(definition["dayImage"]); err != nil {
		return err
	}
	if sky.NightImage, err = file.skyImageFromExtras(definition["nightImage"]); err != nil {
		return err
	}
	environment.Sky = sky
	return nil
}

func applySkyColors(sky *scene.Sky, definition map[string]any) {
	sky.NightZenith = extraVector(definition, "nightZenith", sky.NightZenith)
	sky.NightHorizon = extraVector(definition, "nightHorizon", sky.NightHorizon)
	sky.NightAmbient = extraVector(definition, "nightAmbient", sky.NightAmbient)
	sky.TwilightColor = extraVector(definition, "twilightColor", sky.TwilightColor)
	sky.CelestialPole = extraVector(definition, "celestialPole", sky.CelestialPole).Normalize()
}

func applyStars(stars *scene.Stars, definition map[string]any) {
	values, found := definition["stars"].(map[string]any)
	if !found {
		return
	}
	stars.Density = extraNumber(values, "density", stars.Density)
	stars.Brightness = extraNumber(values, "brightness", stars.Brightness)
	stars.Twinkle = extraNumber(values, "twinkle", stars.Twinkle)
	stars.DaytimeVisibility = extraNumber(values, "daytimeVisibility", stars.DaytimeVisibility)
}

func applyMoon(sky *scene.Sky, definition map[string]any) {
	if disabled, isBool := definition["moon"].(bool); isBool && !disabled {
		sky.Moon = nil
		return
	}
	values, found := definition["moon"].(map[string]any)
	if !found {
		return
	}
	moon := sky.Moon
	moon.Direction = extraVector(values, "direction", moon.Direction)
	moon.AngularSize = extraNumber(values, "size", moon.AngularSize*180/math.Pi) * math.Pi / 180
	moon.Color = extraVector(values, "color", moon.Color)
	moon.Phase = extraNumber(values, "phase", moon.Phase)
	moon.LightIntensity = extraNumber(values, "light", moon.LightIntensity)
}

func applyClouds(clouds *scene.Clouds, definition map[string]any) {
	values, found := definition["clouds"].(map[string]any)
	if !found {
		return
	}
	clouds.Coverage = extraNumber(values, "coverage", clouds.Coverage)
	clouds.Color = extraVector(values, "color", clouds.Color)
	clouds.Speed = extraNumber(values, "speed", clouds.Speed)
	clouds.Scale = extraNumber(values, "scale", clouds.Scale)
}

func constellationsFromExtras(value any) []scene.Constellation {
	list, _ := value.([]any)
	var constellations []scene.Constellation
	for _, entry := range list {
		values, isObject := entry.(map[string]any)
		if !isObject {
			continue
		}
		name, _ := values["name"].(string)
		constellations = append(constellations, scene.Constellation{
			Name:  name,
			Stars: directionsFromExtras(values["stars"]),
			Lines: linesFromExtras(values["lines"]),
			Color: extraVector(values, "color", graphics.Vec4{0.75, 0.85, 1, 1}),
		})
	}
	return constellations
}

func directionsFromExtras(value any) []graphics.Vec4 {
	list, _ := value.([]any)
	directions := make([]graphics.Vec4, 0, len(list))
	for _, entry := range list {
		directions = append(directions, extraVector(map[string]any{"direction": entry}, "direction", graphics.Vec4{}).Normalize())
	}
	return directions
}

func linesFromExtras(value any) [][2]int {
	list, _ := value.([]any)
	var lines [][2]int
	for _, entry := range list {
		pair, isList := entry.([]any)
		if !isList || len(pair) != 2 {
			continue
		}
		first, firstIsNumber := pair[0].(float64)
		second, secondIsNumber := pair[1].(float64)
		if firstIsNumber && secondIsNumber {
			lines = append(lines, [2]int{int(first), int(second)})
		}
	}
	return lines
}

// skyImageFromExtras loads a panorama from a path, or six cube faces from {faces: [...]}.
func (file *gltfFile) skyImageFromExtras(value any) (*scene.SkyImage, error) {
	switch definition := value.(type) {
	case string:
		panorama, err := file.readImage(definition)
		return &scene.SkyImage{Equirectangular: panorama}, err
	case map[string]any:
		return file.cubeFromExtras(definition)
	}
	return nil, nil
}

func (file *gltfFile) cubeFromExtras(definition map[string]any) (*scene.SkyImage, error) {
	paths, _ := definition["faces"].([]any)
	if len(paths) != 6 {
		return nil, fmt.Errorf("sky cube needs 6 faces, found %d", len(paths))
	}
	skyImage := &scene.SkyImage{}
	for faceNumber, path := range paths {
		name, _ := path.(string)
		face, err := file.readImage(name)
		if err != nil {
			return nil, err
		}
		skyImage.Faces[faceNumber] = face
	}
	return skyImage, nil
}

func (file *gltfFile) readImage(name string) (image.Image, error) {
	path := name
	if !filepath.IsAbs(path) {
		path = filepath.Join(file.directory, name)
	}
	reader, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("sky image: %w", err)
	}
	defer reader.Close()
	decoded, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("sky image %s: %w", name, err)
	}
	return decoded, nil
}
