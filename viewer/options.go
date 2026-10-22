package viewer

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/akrck02/go-renderer/failure"
)

// Options configure a viewing session, in three groups: where the view starts, how it is drawn,
// and what is measured or captured.
type Options struct {
	// Width and Height are the window's size.
	Width, Height int
	Start         Start
	Rendering     Rendering
	Measure       Measure
	// Backend is what draws: Automatic (Metal on macOS, OpenGL elsewhere), Metal or OpenGL.
	Backend Backend
}

// Start is where the view starts looking from, and the light it starts in.
type Start struct {
	Walking   bool // on the ground, walking
	Map       bool // on the map: the ground from straight above, without perspective
	Isometric bool // on the map's isometric view
	Night     bool
	// LookAt is the point (x, z) the view starts centred on, and where walking starts; nil is the
	// scene's centre.
	LookAt *[2]float32
	// Zoom is the starting orbit distance (or map span) as a fraction of the default; 0 is 1.
	Zoom float64
	// SunElevationDegrees and SunAzimuthDegrees (clockwise from north) place the sun; NaN keeps
	// the scene's.
	SunElevationDegrees float64
	SunAzimuthDegrees   float64
	// HeadingDegrees, clockwise from north (-z), is the way the walker starts facing; NaN keeps
	// the default. PitchDegrees is how far it starts looking up (negative is down).
	HeadingDegrees float64
	PitchDegrees   float64
	// EyeHeight (scene units) and WalkSpeed (scene units a second) of the walker; 0 keeps the
	// default, a share of the scene's size.
	EyeHeight float64
	WalkSpeed float64
}

// Rendering are switches of how frames are drawn.
type Rendering struct {
	NoShadows bool
	// NoVsync draws as fast as possible instead of waiting for the display.
	NoVsync bool
	// Continuous draws every frame, so water, clouds and plants in the wind keep moving while
	// nothing else happens. By default the viewer draws only when something changes (input, the
	// camera, the sun, notices), unless it runs a program's update function (a simulation).
	Continuous bool
	// FixedSeconds, when not NaN, draws every frame as if this many seconds had passed (water,
	// clouds and waterfalls stand still): captures that must come out the same, to compare them.
	FixedSeconds float64
}

// Measure says what a session captures or measures instead of being explored.
type Measure struct {
	// CapturePath renders one frame to this PNG (in a hidden window) instead of exploring.
	CapturePath string
	// BenchmarkFrames measures this many frames, writes their times to Output and ends; 0
	// explores.
	BenchmarkFrames int
	// LoadingTime is how long the scene took to load, shown with the benchmark's times.
	LoadingTime time.Duration
	// Output is where a benchmark writes its report; nil writes it nowhere.
	Output io.Writer
}

// DefaultOptions returns a 1280×800 window with the scene's own sun, reporting benchmarks on the
// standard output.
func DefaultOptions() Options {
	return Options{
		Width: 1280, Height: 800,
		Start: Start{
			SunElevationDegrees: math.NaN(), SunAzimuthDegrees: math.NaN(), Zoom: 1, HeadingDegrees: math.NaN(),
		},
		Rendering: Rendering{FixedSeconds: math.NaN()},
		Measure:   Measure{Output: os.Stdout},
	}
}

// RegisterFlags declares the viewer options as command-line flags, starting from the defaults.
// flags.PrintDefaults lists them for a usage message.
func (options *Options) RegisterFlags(flags *flag.FlagSet) {
	defaults := DefaultOptions()
	flags.IntVar(&options.Width, "width", defaults.Width, "window width")
	flags.IntVar(&options.Height, "height", defaults.Height, "window height")
	flags.Var(&options.Backend, "backend", "what draws the scene: automatic, metal or opengl")
	options.Start.registerFlags(flags, defaults.Start)
	options.Rendering.registerFlags(flags, defaults.Rendering)
	flags.StringVar(&options.Measure.CapturePath, "capture", "", "render one frame to this PNG and exit")
	flags.IntVar(&options.Measure.BenchmarkFrames, "benchmark", 0, "measure this many frames, print CPU and GPU times per pass and exit")
}

func (start *Start) registerFlags(flags *flag.FlagSet, defaults Start) {
	flags.BoolVar(&start.Walking, "walk", false, "start walking on the ground")
	flags.BoolVar(&start.Map, "map", false, "start on the map: the ground from straight above, without perspective (M switches views)")
	flags.BoolVar(&start.Isometric, "isometric", false, "start on the map's isometric view")
	flags.BoolVar(&start.Night, "night", false, "start at night")
	flags.Float64Var(&start.SunElevationDegrees, "sun-elevation", defaults.SunElevationDegrees, "sun elevation in degrees (negative = below the horizon)")
	flags.Float64Var(&start.SunAzimuthDegrees, "sun-azimuth", defaults.SunAzimuthDegrees, "sun azimuth in degrees, clockwise from north")
	flags.Var(pointFlag{&start.LookAt}, "look-at", "start centred on this point, \"x,z\" in scene units (with -walk, start walking there)")
	flags.Float64Var(&start.Zoom, "zoom", defaults.Zoom, "initial orbit distance as a fraction of the default (0.1 = ten times closer)")
	flags.Float64Var(&start.HeadingDegrees, "heading", defaults.HeadingDegrees, "with -walk, start facing this way: degrees clockwise from north (-z)")
	flags.Float64Var(&start.PitchDegrees, "pitch", defaults.PitchDegrees, "with -walk, start looking up this many degrees (negative = down)")
	flags.Float64Var(&start.EyeHeight, "eye-height", defaults.EyeHeight, "with -walk, eye height in scene units (0 = a share of the scene size)")
	flags.Float64Var(&start.WalkSpeed, "walk-speed", defaults.WalkSpeed, "walking speed in scene units per second (0 = a share of the scene size)")
}

func (rendering *Rendering) registerFlags(flags *flag.FlagSet, defaults Rendering) {
	flags.BoolVar(&rendering.NoShadows, "no-shadows", false, "start without sun shadows")
	flags.BoolVar(&rendering.NoVsync, "no-vsync", false, "do not wait for the display: the title shows the real frames per second")
	flags.BoolVar(&rendering.Continuous, "continuous", false, "draw every frame, so water, clouds and wind keep moving while nothing else happens")
	flags.Float64Var(&rendering.FixedSeconds, "time", defaults.FixedSeconds, "draw as if this many seconds had passed, with water and clouds still (captures to compare)")
}

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
		return failure.Wrap(InvalidPoint, strconv.Quote(text), nil)
	}
	var point [2]float32
	for index, part := range parts {
		number, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return failure.Wrap(InvalidPoint, strconv.Quote(text), nil)
		}
		point[index] = float32(number)
	}
	*value.point = &point
	return nil
}

// Backend is the graphics interface a session draws through.
type Backend string

const (
	Automatic Backend = ""       // Metal on macOS, falling back to OpenGL if it cannot start; OpenGL elsewhere
	Metal     Backend = "metal"  // only on macOS
	OpenGL    Backend = "opengl" // OpenGL 4.1
)

// Set reads a backend from a flag: automatic, metal or opengl.
func (backend *Backend) Set(text string) error {
	switch text {
	case "automatic", "":
		*backend = Automatic
	case string(Metal), string(OpenGL):
		*backend = Backend(text)
	default:
		return failure.Wrap(UnknownBackend, strconv.Quote(text), nil)
	}
	return nil
}

// String is the backend as its flag writes it: automatic, metal or opengl.
func (backend *Backend) String() string {
	if *backend == Automatic {
		return "automatic"
	}
	return string(*backend)
}

// usesMetal tells whether a session should try Metal first.
func (backend Backend) usesMetal() bool {
	return backend == Metal || backend == Automatic && runtime.GOOS == "darwin"
}
