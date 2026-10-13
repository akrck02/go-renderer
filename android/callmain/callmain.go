//go:build android && (arm64 || amd64)

// Package callmain starts a program's main function inside a shared library. A program built for
// Android is a shared library (-buildmode=c-shared) that the system loads into an activity, and Go
// does not run main.main there by itself: the activity's start finds it by name and calls it here.
// It is a package of its own because it holds Go assembly, which a package using cgo cannot.
package callmain

// Call calls a function that takes nothing and returns nothing, by the address of its code.
// It is meant only for main.main.
func Call(code uintptr)
