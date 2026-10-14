# On Android

**Status:** a first step (October 2026). Package `android` runs a go-renderer program as an Android
application: go-ui's interfaces, and the 3D scene under them, draw on an Android 16 emulator and on a
Pixel 6 Pro. There is no golang.org/x/mobile, and no Java but one small class for screen readers:
the program is a shared library that Android's own `NativeActivity` loads. go-ui's `docs/android.md` has the screenshots, the build and
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
| The system's bars and theme, and the wallpaper's palettes, through JNI | `android/system.c`, `android/palettes.go` |
| What a screen reader (TalkBack) reads, and what it asks | `android/screenreader.go`, `screenreader.c`, `android/java/ScreenReader.java` |

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
would look like the one shown is not drawn, except on a new surface (back from the background),
which is empty until painted: a screen with no scene stayed black there. After a rotation, EGL's
surface keeps the old size until a frame has been shown, so the window paints at once and once more
at the new size; before, a rotated screen kept the old frame, laid out for the other way round.

**The bars and the cutout.** `CurrentInsets` gives what the status bar, the gesture or navigation
bar and a camera's cutout cover, on every side: a phone on its side has its cutout at the left or the
right. A package whose theme sets `windowLayoutInDisplayCutoutMode` to `shortEdges` paints beside the
cutout too, and keeps its controls clear of it with those insets. It draws at most `Options.MaxFramesPerSecond` a second,
60 by default: a phone refreshes at 90 or 120 Hz, and a finger dragging a 3D view sends touch events
far more often. When the context is new (the first frame, or back from the background), the drawing
function is told, and makes its renderers again.

**When it closes.** The Go runtime writes a crash's report (with the failing goroutine's stack) to the
application's folder; `LastCrash` gives it to the next run, and `Graphics` names the phone's OpenGL ES
and GPU. A phone shows a person nothing when an application closes. What the program writes to its
standard output and error (a `log.Fatal`, a panic) goes to Android's log a line at a time, under the
tag `Go` (`adb logcat -s Go`), with a line when the runtime starts and one when it starts `main`.

**The wallpaper's colours.** On Android 12 and later the phone makes five palettes from its
wallpaper (`system_accent1_0` … `system_neutral2_1000`, the colours its own applications use).
`SystemPalettes` returns them as last read: with the theme, when the activity comes back or its
configuration changes, which is when they change. An older phone has none, and says so.

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
`android.app.NativeActivity`, with `android.app.lib_name` set to `program`. For a screen reader it
also holds `classes.dex`, compiled from `android/java/ScreenReader.java` (`javac`, then the build
tools' `d8`), and says `android:hasCode="true"`; without it, the program runs with no screen reader.
go-ui-android's `go-ui-apk` does it all with the Android SDK's tools. The 16 KB alignment is what
phones with 16 KB memory pages (Android 15 and later) need.

## Typing and notifications

**The on-screen keyboard.** `ShowKeyboard(typing)` brings the keyboard up while a text field is typed
in and puts it away after (go-ui's window calls it with `Context.Typing`). NativeActivity's own call
for it does nothing on recent Androids, so the window's `InputMethodManager` is asked through JNI,
with the NDK's call as a fallback. A finger coming down while typing asks again, for a person who
put the keyboard away and touched the field to have it back. Every key, from either keyboard, is
given the character Android's own `KeyEvent.getUnicodeChar` says it types: any layout's letters,
accented ones, digits and symbols. A package whose activity has `windowSoftInputMode="adjustResize"`
shrinks while the keyboard is up, so what is typed in stays in sight. The window itself keeps its
size; what changes is how much of its bottom the keyboard covers, which Android tells through
`onContentRectChanged`: the insets are read again then, as the keyboard comes and goes.

**A finger lifted leaves nothing under it.** After the frame where a finger is lifted
(`input.TouchPointer`), the pointer goes off the window (`State.PointerLeaves`): with a mouse, the
control under the pointer is lit, and on a touch screen the last control touched stayed so.

**Notifications.** `Notify(id, title, text)` puts a notification in the phone's shade, as other
applications' are, on a channel of the program's (`Channel`, which the application's settings
list): touching it opens the program. Android 13 and later show them only once the person allows
it: `NotificationsAllowed` tells, and `AskForNotifications` shows Android's question (the package
names `android.permission.POST_NOTIFICATIONS`). Tried on a OnePlus 7 Pro (Android 10).

**The buzz of a long press.** `FeelLongPress()` gives the short buzz Android's own views give when a
touch becomes a long press (`View.performHapticFeedback` on the window's view): it follows the
person's setting for touch feedback and needs no permission.

**No black before the first frame.** Android takes its starting screen away as soon as the window
is there, while the program's first frame may still be on its way (the Go runtime starting, fonts
read). The window is cleared to the theme's plain white or black the moment its surface opens, so
the starting screen gives way to a plain background, never to black.

**The phone's services.** `TextScale()` is the person's text size (Settings › Display › Font
size), read with the theme. The input state's clipboard is Android's own `ClipboardManager`, so text
fields copy and paste with other applications. `OpenLink(link)` opens a link in the application
that shows it (an `ACTION_VIEW` intent: a web page, `mailto:`, `tel:`, `geo:`), and `Share(title,
text)` offers a text through the system's sheet (`ACTION_SEND` in a chooser). `Options.Hidden` is
called on the drawing thread when the activity pauses: Android may end the program after that
without a word, so it is the moment to keep what must survive.

**Films.** `OpenVideo(path)` decodes a film's pictures with the phone's own decoders (the NDK's
`MediaCodec`: MP4 and WebM, H.264, H.265, VP8, VP9, AV1), a step at a time from the goroutine that
draws: `Advance(until, pixels)` decodes up to a moment and copies the picture due by then. The
pictures come out in the decoder's own buffers, laid out as its output format says: the three
planes one after another (I420, from software decoders such as the emulator's) or the luma then the
chroma interleaved (NV12, from most hardware decoders), with rows padded to the decoder's liking.
`YUVPlanes.ToRGBA` turns them into colours, in video's limited range, with the high-definition
standard's weights from 720 lines up. Drawing into an `AImageReader` looked simpler but, on a
OnePlus 7 Pro, the decoder then wrote its pictures compressed in its maker's own layout, which no
processor reads: green noise. The sound is not played yet.

**The back gesture.** Android's back (the gesture from a side, or the button) is the program's:
the window records it (`State.PressBack`), and the part of the program the person is in answers it
(`State.TakeBack`, go-ui's `Context.Back`): a dialog closes, a screen goes back to the one before.
A back nobody answers sends the program to the background (`moveTaskToBack`), as a phone's own
applications do at their first screen; it comes back as it was. When the on-screen keyboard is up,
the back puts it away first, as everywhere on Android.

## Screen readers

A screen reader (TalkBack, or any accessibility service) reads what a view holds through the
view's `AccessibilityNodeProvider`, an abstract Java class: JNI can call Java, but cannot make a
subclass, and a proxy only stands in for interfaces. So the package carries one class,
`ScreenReader.java`, embedded in package `android/java` for the tool that compiles it. Everything
else stays in Go and C.

```text
the program's interface   the nodes of a frame: what each is, where, its state (go-ui makes them)
android (Go)              ShowAccessibleNodes: sent only when they change, in screen pixels, text as UTF-16
screenreader.c            the Java arrays, and the class's native methods
ScreenReader.java         a virtual view per node, answering TalkBack; its focus and actions back to Go
```

- **Installing.** Each time Android makes the activity (a rotation too), `ANativeActivity_onCreate`
  loads the class through the activity's class loader and puts a provider on the view NativeActivity
  shows. It listens for accessibility turning on and off: `ScreenReaderOn` tells, and while it is
  off the program makes no nodes at all.
- **A finger exploring the screen.** While TalkBack is on, Android turns a finger into hover events,
  which reach the input queue like touches. They go to Java, which finds the smallest node under the
  finger and tells TalkBack it came onto it; TalkBack reads it and puts its focus there.
- **What TalkBack asks.** A double tap is a click on the node it is on, a double tap held a long
  click, a swipe with two fingers or the volume keys on a slider a scroll. Java passes them on
  (`TakeReaderActions`) and wakes the drawing thread; the interface does them in the next frame. Where
  TalkBack's focus is (`ScreenReaderFocus`) comes the same way, for the program to ring it: the
  window is a native surface, where Android draws no ring of its own.
- **Text as UTF-16.** JNI's own UTF-8 is not quite UTF-8: an emoji in a label sent that way stops a
  program. Labels cross as Java keeps them.

## Not yet

- Characters a keyboard sends whole rather than key by key (some keyboards' suggestions, emoji):
  Android hands them as one event whose text needs the Java event, which a NativeActivity does not
  get.
- Long presses.
- Instanced drawing with OpenGL ES 3, and the full scene renderer (shadows, sky, waves) on phones.
