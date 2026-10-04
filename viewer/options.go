package viewer

import (
	"flag"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Options configure a viewing session.
type Options struct {
	Width, Height int
	CapturePath   string // render one frame to this PNG (hidden window) instead of exploring
	StartWalking  bool
	NoShadows     bool
	Night         bool
	SunElevation  float64     // degrees; NaN keeps the scene's sun
	SunAzimuth    float64     // degrees clockwise from north; NaN keeps the scene's sun
	Zoom          float64     // initial orbit distance as a fraction of the default (0 = 1)
	Benchmark     int         // frames to measure before printing times and exiting; 0 = explore
	NoVsync       bool        // draw as fast as possible instead of waiting for the display
	LookAt        *[2]float32 // point (x, z) the view starts centred on, and where walking starts; nil = the scene centre
	Heading       float64     // walking: degrees clockwise from north (-z) the walker starts facing; NaN keeps the default
	Pitch         float64     // walking: degrees the walker starts looking up (negative = down)
	EyeHeight     float64     // walking: eye height in scene units; 0 keeps the default (a share of the scene size)
}

// DefaultOptions returns a 1280×800 window with the scene's own sun.
func DefaultOptions() Options {
	return Options{Width: 1280, Height: 800, SunElevation: math.NaN(), SunAzimuth: math.NaN(), Zoom: 1, Heading: math.NaN()}
}

// RegisterFlags declares the viewer options as command-line flags, starting from the defaults.
func (options *Options) RegisterFlags(flags *flag.FlagSet) {
	defaults := DefaultOptions()
	flags.IntVar(&options.Width, "width", defaults.Width, "window width")
	flags.IntVar(&options.Height, "height", defaults.Height, "window height")
	flags.StringVar(&options.CapturePath, "capture", "", "render one frame to this PNG and exit")
	flags.BoolVar(&options.StartWalking, "walk", false, "start walking on the ground")
	flags.BoolVar(&options.Night, "night", false, "start at night")
	flags.Float64Var(&options.SunElevation, "sun-elevation", defaults.SunElevation, "sun elevation in degrees (negative = below the horizon)")
	flags.Float64Var(&options.SunAzimuth, "sun-azimuth", defaults.SunAzimuth, "sun azimuth in degrees, clockwise from north")
	flags.IntVar(&options.Benchmark, "benchmark", 0, "measure this many frames, print CPU and GPU times per pass and exit")
	flags.BoolVar(&options.NoShadows, "no-shadows", false, "start without sun shadows")
	flags.BoolVar(&options.NoVsync, "no-vsync", false, "do not wait for the display: the title shows the real frames per second")
	flags.Var(pointFlag{&options.LookAt}, "look-at", "start centred on this point, \"x,z\" in scene units (with -walk, start walking there)")
	flags.Float64Var(&options.Zoom, "zoom", defaults.Zoom, "initial orbit distance as a fraction of the default (0.1 = ten times closer)")
	flags.Float64Var(&options.Heading, "heading", defaults.Heading, "with -walk, start facing this way: degrees clockwise from north (-z)")
	flags.Float64Var(&options.Pitch, "pitch", defaults.Pitch, "with -walk, start looking up this many degrees (negative = down)")
	flags.Float64Var(&options.EyeHeight, "eye-height", defaults.EyeHeight, "with -walk, eye height in scene units (0 = a share of the scene size)")
}

// Usage lists the viewer flags for a usage line.
const Usage = "[-width W] [-height H] [-capture frame.png] [-walk] [-no-shadows] [-night] [-sun-elevation D] [-sun-azimuth D] [-zoom F] [-benchmark N] [-no-vsync] [-look-at x,z] [-heading D] [-pitch D] [-eye-height H]"

// pointFlag reads a point "x,z" from the command line.
type pointFlag struct {
	point **[2]float32
}

func (value pointFlag) String() string {
	if value.point == nil || *value.point == nil {
		return ""
	}
	return fmt.Sprintf("%g,%g", (*value.point)[0], (*value.point)[1])
}

func (value pointFlag) Set(text string) error {
	parts := strings.Split(text, ",")
	if len(parts) != 2 {
		return fmt.Errorf("expected x,z, got %q", text)
	}
	var point [2]float32
	for index, part := range parts {
		number, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return fmt.Errorf("expected x,z, got %q", text)
		}
		point[index] = float32(number)
	}
	*value.point = &point
	return nil
}
