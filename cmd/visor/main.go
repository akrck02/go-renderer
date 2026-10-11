// Command visor opens a glTF 2.0 scene (.glb or .gltf) and lets you explore it (see package viewer
// for the controls).
//
//	go run ./cmd/visor scene.glb
//	go run ./cmd/visor -capture frame.png scene.glb   (renders one frame to a PNG, no visible window)
//	go run ./cmd/visor -benchmark 120 scene.glb        (prints CPU and GPU times per pass)
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/akrck02/go-renderer/loaders"
	"github.com/akrck02/go-renderer/viewer"
)

func main() {
	options := viewer.DefaultOptions()
	options.RegisterFlags(flag.CommandLine)
	cpuProfile := flag.String("cpu-profile", "", "write a CPU profile of the session to this file (go tool pprof)")
	flag.Parse()
	if *cpuProfile != "" {
		stop, err := startProfile(*cpuProfile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer stop()
	}
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: visor", viewer.Usage, "scene.glb")
		os.Exit(2)
	}
	scenePath := flag.Arg(0)
	loadingStarted := time.Now()
	world, err := loaders.LoadScene(scenePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot load scene:", err)
		os.Exit(1)
	}
	if err := viewer.Run(world, options, "Visor · "+scenePath, nil, time.Since(loadingStarted)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
