# 🧰 Troubleshooting

This page is for anyone whose build or window does not work. Find the message you see, and read
the fix under it.

## Building

### `fatal error: GL/gl.h`, `X11/Xlib.h` or `wayland-client.h`: No such file or directory

The window goes through GLFW, which cgo compiles from C. C needs the headers of OpenGL, X11 and
Wayland, and Go needs a C compiler. On Debian or Ubuntu:

```sh
sudo apt install gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev
```

Fedora's packages are in [get started](2-get-started.md). The first build compiles GLFW, so it
takes a minute; later ones are quick.

### `cgo: C compiler "gcc" not found`, or `build constraints exclude all Go files`

cgo is off, or there is no C compiler. Install one (`gcc`, or Xcode's command line tools on macOS
with `xcode-select --install`) and check that `go env CGO_ENABLED` prints `1`.

Packages that need no window (`scene`, `draw2d`, `input`, `internal/sceneplan`) build without cgo. Only
`opengl`, `windowloop`, `viewer`, `metal` and `cmd/visor` need it.

### Warnings on macOS: `'glClear' is deprecated`, `MTLStorageModeManaged is deprecated`

They are warnings, not errors: the program still builds and runs.

- **OpenGL** has been deprecated on macOS since 2018. Every OpenGL call warns when GLFW and
  go-gl compile. On a Mac, the viewer draws through Metal by default; see [Metal](4-guides/7-metal.md).
- **Metal**: newer SDKs (macOS 27) mark `MTLStorageModeManaged` as deprecated. `metal/flat_darwin.m`
  and `metal/scene_darwin.m` use it for textures the processor writes. Replacing it is part of the
  [proposal towards 1.0](proposals/1-towards-version-1.md).

## Opening a window

### `VersionUnavailable` or `GLX: Failed to create context`

The renderer asks for **OpenGL 4.1, core profile**. The graphics driver offers less. Check what it
offers:

```sh
glxinfo -B | grep "core profile version"
```

Update the driver, or use Mesa's software one (`LIBGL_ALWAYS_SOFTWARE=1`), which gives 4.5 on
any computer, slowly. In a virtual machine, turn on 3D acceleration.

### A blank or black window, or `GLX extension not found`, under Xvfb

Xvfb is a display with no screen, for servers and CI. It draws OpenGL only through Mesa's GLX.
Install Mesa's drivers, then try again:

```sh
sudo apt install xvfb libgl1-mesa-dri
xvfb-run go run ./cmd/visor -sample island -capture island.png
```

`-capture` draws one frame into a picture and exits: no need to look at the window.

### On Wayland: the window has no title bar, or opens on X11 instead

GLFW 3.4 can talk to Wayland directly: it prefers Wayland when `WAYLAND_DISPLAY` is set and its
Wayland support was built, and X11 otherwise. Some Wayland desktops (GNOME among them) draw no title bar for such windows. To run under X11 through
XWayland instead, unset the variable for one run:

```sh
env -u WAYLAND_DISPLAY go run ./cmd/visor -sample island
```

### `StartLoop must run on the main thread`

Windows only receive their events on the program's first thread (macOS insists). Lock it in
package `main`:

```go
func init() { runtime.LockOSThread() }
```

## Other platforms

### In a browser: `build constraints exclude all Go files in …/opengl`

`opengl`, `windowloop` and `viewer` need GLFW, which does not exist in a browser. A web page uses
`browser` and `gles` instead; see [in a web browser](4-guides/5-web.md). The CI vets every package a
page can use with `GOOS=js GOARCH=wasm`.

### Android: `ld: cannot find -llog`, or the app closes at once

An Android program is a shared library built with the NDK's compiler for each architecture, with
`CGO_ENABLED=1 GOOS=android` and `-buildmode=c-shared`. The whole command, and the 16 KB alignment
recent phones need, are in [on Android](4-guides/6-android.md#building).

### The Nintendo 3DS: `unknown target nintendo3ds`

The 3DS build needs a TinyGo with the `nintendo3ds` target (a separate fork, not upstream TinyGo)
and devkitPro's libctru and citro3d. See [on the Nintendo 3DS](4-guides/8-3ds.md#building).

## Still stuck?

Open an issue with the command you ran, the whole message, your system, and what
`go version` and `glxinfo -B` print.
