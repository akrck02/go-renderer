// Command visor opens a glTF 2.0 scene (.glb or .gltf) and lets you explore it (see package viewer
// for the controls).
//
//	go run ./cmd/visor scene.glb
//	go run ./cmd/visor -capture frame.png scene.glb   (renders one frame to a PNG, no visible window)
//	go run ./cmd/visor -benchmark 120 scene.glb        (prints CPU and GPU times per pass)
//	go run ./cmd/visor -sample island                  (a built-in made-up island; also "night")
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/akrck02/go-renderer/loaders"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-renderer/scene/sample"
	"github.com/akrck02/go-renderer/viewer"
)

func main() {
	options := viewer.DefaultOptions()
	options.RegisterFlags(flag.CommandLine)
	cpuProfile := flag.String("cpu-profile", "", "write a CPU profile of the session to this file (go tool pprof)")
	sampleName := flag.String("sample", "", "open a built-in scene instead of a file: island or night")
	flag.Parse()
	if *cpuProfile != "" {
		stop, err := startProfile(*cpuProfile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer stop()
	}
	loadingStarted := time.Now()
	world, name, err := openScene(*sampleName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := viewer.Run(world, options, "Visor · "+name, nil, time.Since(loadingStarted)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// openScene builds the named sample, or loads the file given as the argument.
func openScene(sampleName string) (*scene.Scene, string, error) {
	switch {
	case sampleName == "island":
		return sample.Island(), "island", nil
	case sampleName == "night":
		return sample.Night(), "island at night", nil
	case sampleName != "":
		return nil, "", fmt.Errorf("unknown sample %q: use island or night", sampleName)
	case flag.NArg() != 1:
		return nil, "", fmt.Errorf("usage: visor %s scene.glb (or -sample island)", viewer.Usage)
	}
	path := flag.Arg(0)
	world, err := loaders.LoadScene(path)
	if err != nil {
		return nil, "", fmt.Errorf("cannot load scene: %w", err)
	}
	return world, path, nil
}

// startProfile records where the processor spends its time until the returned function is called.
func startProfile(path string) (func(), error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		file.Close()
		return nil, err
	}
	return func() { pprof.StopCPUProfile(); file.Close() }, nil
}
