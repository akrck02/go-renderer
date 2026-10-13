# On Android

**Status:** a first step (October 2026). go-ui's interfaces, and the 3D scene under them, draw on
an Android emulator through this repository's `gles` renderers; the Android window itself lives in go-ui
(its `docs/android.md` has the screenshots, the build and what was tested). This page is about
go-renderer's part, and about the choices behind it.

## go-renderer's part

Almost nothing is Android-specific here, which was the point:

- **`gles`** draws flat interfaces and simple scenes through OpenGL ES, against the
  `gles.Functions` interface (see [in a web browser](web.md#gles-renderers-written-once-for-opengl-es)).
  The same code draws in browsers and on phones.
- **`frameloop`** is the fixed-step clock both the desktop and the Android loop use.
- **`input`** takes touches as a pointer: go-ui's window turns the first finger into the cursor and
  the left button, so every control works unchanged.

The implementation of `gles.Functions` over `golang.org/x/mobile/gl` lives in go-ui's window, not
here, so that go-renderer (which other programs build on) does not depend on
`golang.org/x/mobile`.

## Why OpenGL ES 2 and not 3

The first plan was OpenGL ES 3. On the emulator it failed with "GLES3 function is missing":
`golang.org/x/mobile` compiles its OpenGL bindings against the OpenGL ES **2** headers on Android
and asks EGL for a version 2 context, so vertex array objects, `R8` textures and GLSL ES 3.00 are
not there. Rather than fork x/mobile, the renderers were narrowed to what OpenGL ES 2 has (GLSL ES
1.00, no vertex arrays, a `LUMINANCE` atlas), which WebGL2 also accepts: one code path for both.

## How the route was chosen

| Option | For | Against | Choice |
|---|---|---|---|
| gomobile with `golang.org/x/mobile/app` and OpenGL ES | No Java or Kotlin; the activity, the EGL surface, lifecycle and touch come ready; gomobile compiles and packs | It changes slowly; OpenGL ES 2 only; its manifest targets an Android too old to install (worked around, see go-ui) | **Chosen** for the first step |
| A native activity of our own (cgo, EGL, `android_native_app_glue`) | Full control: OpenGL ES 3, instancing, insets, the keyboard | More C to write and keep | The fallback if x/mobile gets in the way |
| A Kotlin activity with `GLSurfaceView` and Go as a shared library | The most "Android" integration | A Gradle project and JNI to keep | Later, if needed |
| Gio, Ebitengine | Already run on Android | Another way of drawing or of building interfaces: go-ui would be rewritten | No |

## Not yet

- OpenGL ES 3 and instancing, which need a native activity of our own (see above).
- Shadows, the sky dome and waves, on phones as in browsers (see [in a web browser](web.md#not-yet)).

The 3D scene does draw on Android: go-ui's example shows the sample island under its interface, at
any time of day, on the emulator.
