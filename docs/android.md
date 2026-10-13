# On Android

**Status:** a first step (October 2026). Package `android` runs a go-renderer program as an Android
application: go-ui's interfaces, and the 3D scene under them, draw on an Android 16 emulator and on a
Pixel 6 Pro. There is no Java and no golang.org/x/mobile: the program is a shared library that
Android's own `NativeActivity` loads. go-ui's `docs/android.md` has the screenshots, the build and
what was tested.

## How it fits together

```text
your program            the same code as on the desktop and in a browser
go-ui window            window.Run: an interface, and a scene under it
android                 the activity, its window and touches, a thread that draws (this package)
gles                    FlatRenderer and SceneRenderer, over gles.Functions (shared with browsers)
Android                 NativeActivity loads the program, a shared library built with the NDK
```

| Piece | Where |
|---|---|
| The library's entry point and the activity's callbacks, EGL, the screen's density | `android/activity.c` |
| The callbacks handed to the drawing thread; starting main | `android/activity.go`, `android/callmain` |
| The drawing thread: waiting, the frame cap, steps and frames | `android/loop.go` |
| Touches and a hardware keyboard | `android/input.go`; fingers as the pointer in `input.TouchPointer` |
| OpenGL ES, called directly | `android/gl.go` |
| The system's bars and theme, through JNI | `android/system.c` |

**Starting.** Android loads the library and calls `ANativeActivity_onCreate`. Go does not run
`main.main` in a shared library by itself, so the first time the library finds it by name and calls
it (`callmain`, a few lines of assembly: a package with cgo cannot hold Go assembly). It also hands Go
the application's own folder, which becomes `os.TempDir`: the Go runtime read the environment when
the library loaded, before C could set it.

**One drawing thread.** Android calls the activity's callbacks on its own interface thread; they are
handed to a thread of the program's own, which holds the OpenGL ES context and an Android looper. The
looper wakes it for touches (the input queue is attached to it, so the looper is prepared to accept a
queue with no callback), for the activity's news, for a frame a control asked for (`RequestFrame`)
and for `Wake`; in between it sleeps. When Android takes a window or an input queue away, its
callback waits until the drawing thread has let it go, as Android requires.

**Frames.** Like the desktop and the browser, it draws only when something happens, and a frame that
would look like the one shown is not drawn. It draws at most `Options.MaxFramesPerSecond` a second,
60 by default: a phone refreshes at 90 or 120 Hz, and a finger dragging a 3D view sends touch events
far more often. When the context is new (the first frame, or back from the background), the drawing
function is told, and makes its renderers again.

**When it closes.** The Go runtime writes a crash's report (with the failing goroutine's stack) to the
application's folder; `LastCrash` gives it to the next run, and `Graphics` names the phone's OpenGL ES
and GPU. A phone shows a person nothing when an application closes. What the program writes to its
standard output and error (a `log.Fatal`, a panic) goes to Android's log a line at a time, under the
tag `Go` (`adb logcat -s Go`), with a line when the runtime starts and one when it starts `main`.

**The window's news can come first.** Android may hand the activity its window before the drawing
thread has its looper: telling it then wakes nothing. The thread wakes its own looper once as it
makes it, so the first wait reads what is already there. Without that, a OnePlus 7 Pro (Android 10)
stayed on a blank window, its drawing thread asleep with the window's news unread.

## Why its own activity

The first step used `golang.org/x/mobile` (gomobile), which gives a Go program an activity, an EGL
surface and its events with no Java. It worked, with two costs:

- Its OpenGL bindings are compiled against the OpenGL ES **2** headers, and it asks for a version 2
  context. go-renderer's `gles` keeps to OpenGL ES 2 for that reason (see [in a web browser](web.md)),
  which WebGL2 also accepts.
- Each OpenGL call goes to another thread through a channel, three at a time. A CPU profile taken on
  the phone found it was about **two thirds** of what a frame cost: `runtime.futex` 34 %, `libc` 17 %,
  `cgocall` 13 %, the scheduler 11 %.

With direct calls, on a Pixel 6 Pro:

| | With x/mobile | Its own activity |
|---|---|---|
| Dragging the sample island for 10 s (go-ui's example) | 82 % of a core | **41 %** |
| A 3D scene of a few hundred moving models, playing | 84 % | **31 %** |
| Idle | 0 | 0 |

The package asks for an OpenGL ES 3 context when the phone has one (both phones tried give 3.x), so
instanced drawing can come later behind the same `gles.Functions`.

## Building

```bash
CC=$ANDROID_NDK_HOME/toolchains/llvm/prebuilt/darwin-x86_64/bin/aarch64-linux-android26-clang \
CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
go build -buildmode=c-shared -ldflags=-extldflags=-Wl,-z,max-page-size=16384 -o libprogram.so .
```

A package then holds `lib/arm64-v8a/libprogram.so` and a manifest whose activity is
`android.app.NativeActivity`, with `android.app.lib_name` set to `program` and `android:hasCode="false"`.
go-ui's `cmd/mobile/build-apk.sh` does it all with the Android SDK's tools. The 16 KB alignment is what
phones with 16 KB memory pages (Android 15 and later) need.

## Not yet

- The on-screen keyboard, and the characters of keys other than letters, digits and space (a key's
  character needs a call into Java).
- Long presses, and the back gesture inside the program (it closes the activity today).
- Instanced drawing with OpenGL ES 3, and the full scene renderer (shadows, sky, waves) on phones.
