# 4. Emoji on any Android

> Status: approved, to build · Author's approval: proposal ☑ plan ☑ (through the organizer, 2026-10-27)

## Why

On Android the overlay reads colour emoji from the system's font files, and it can only read one
kind of them: **CBDT**, where each emoji is a PNG picture. Android 13 moved its own emoji to
**COLRv1**, where each emoji is a vector drawing with gradients, and kept a CBDT copy,
`NotoColorEmojiLegacy.ttf`, for programs like this one. On the Android 14 emulator that copy is
there, and a chat made with Go UI draws 👍 in colour.

A Pixel 6 Pro on Android 17 draws `?` instead: a recent Android, and phones whose maker ships its
own emoji font, carry no CBDT file the overlay can read. Every emoji turns into a question mark,
in every program built with go-renderer, and nothing tells the program why.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | On an Android with no CBDT emoji font, an emoji of one character (😀, 👍, 🚀) and one asked for with `U+FE0F` (❤️) is drawn in colour, at the text's size, on its baseline | emulator capture on Android 14 with `NotoColorEmojiLegacy.ttf` moved out, next to the same label with it in place |
| R2 | An emoji is measured as it is drawn, so text with emoji wraps, centres and ellipsises right | test with a fake platform source: the width of "a😀b" is the sum of its three advances |
| R3 | Where a CBDT font has the emoji (Android 14 and earlier), it is used as today, without calling Java | test: with both sources, the font file's picture wins and the platform source is never asked |
| R4 | Each emoji is made once per size; a frame showing emoji already made calls no Java and allocates nothing | test: drawing the same label twice asks the platform once |
| R5 | When the platform cannot draw it either, the text keeps its `?`, as today, and nothing fails | test with a platform source that has nothing |
| R6 | The desktop and the browser draw emoji exactly as today | the overlay's tests, and proposal 3's browser capture, unchanged |
| R7 | Only font files (by their extension) are read as colour emoji fonts | test: a file named `NotoColorEmoji.ttf.hidden` is not read |

## Out of scope

- **Sequences joined by U+200D** (a family, a skin tone, most flags): shown as their parts, as on
  every platform today. Android's canvas could draw them later; the seam below allows it.
- **Reading COLRv1 in Go**, for desktops whose Noto Color Emoji is COLRv1 too: compared below,
  and left for a later proposal if wanted.
- **Typing emoji** in Go UI's own fields on Android: `android/java/TextInput.java` is someone
  else's work in progress, and this proposal does not touch it.

## Design

### The idea: let Android draw what it can draw

Proposal 3 did this for the browser: the browser has the system's emoji, so it paints each one
into a small canvas, and Go reads the pixels back once. Android's own `Canvas` can do the same. A
`Paint` with the text size, `Canvas.drawText` into a `Bitmap`, and `Paint.getFontMetrics` and
`measureText` for where it sits. Android draws whatever its emoji font is (COLRv1, CBDT, a maker's
own), so this works on every phone, now and after the next format change.

### Where it plugs in

Proposal 3 left a single seam: `emojiFont` asks an **`emojiSource`** for a character's picture at a
size, and both the painter and the `Measurer` go through it. Nothing above it changes.

- **A chain of sources.** On Android the source becomes two in order: the font files first
  (today's code, cheap, no Java), then the platform. The second is asked only for a character the
  first has no picture of (R3).
- **A generic hook in the overlay.** The overlay cannot call Java: it knows nothing of Android,
  and `android` already imports it through `gles`. So the overlay offers a function a platform
  sets once, at start, before the first frame:

  ```go
  // UseSystemEmoji makes the overlay ask the platform to draw an emoji no font file has.
  overlay.UseSystemEmoji(func(character rune, pixels int) (picture image.Image, left, top, advance float32, found bool))
  ```

  It is named for what it does on any platform, not for Android. The desktop and the browser
  never set it (R6).
- **The Android side** (`android/`): a new Java file, `android/java/EmojiPicture.java`, compiled
  and loaded like `ScreenReader.java`, with one static method. It takes the character and the size
  and returns the ARGB pixels with the four measures. A small C function calls it through JNI and
  attaches the calling thread to the Java machine when needed, since the overlay draws on the GL
  thread. Go turns the pixels into an `image.NRGBA` (Android's are premultiplied: they are divided
  back). The `android` package sets the hook when the activity starts.
- **Telling a monochrome letter from an emoji.** Android can draw a character in a plain font
  when its emoji font lacks it. As in the browser, the character is drawn in two colours; a colour
  emoji keeps its own colours, a letter follows the fill, and then there is no picture (R5).

### Why not read COLRv1 in Go

| | Android draws it | COLRv1 read in Go |
|---|---|---|
| Works on | every Android, any emoji font, future formats | only phones whose emoji font is COLRv1 |
| Size of the work | a Java method, a C call, a Go hook; about 200 lines | a paint graph (gradients, transforms, blending) and a rasteriser for it; well over 1,500 lines |
| Same look as the phone's other apps | yes, by construction | close, if every paint is right |
| Also helps the desktop | no | yes, where the Noto Color Emoji is COLRv1 |
| Calls Java | once per emoji and size | never |

Reading COLRv1 would teach the most, but it is a large project for a narrower result. Bundling a
CBDT emoji font in each app (about 10 MB, and out of date against the phone) is not taken either.

### Reproduced without the phone

Deploy-manager reproduced the `?` on the Android 14 emulator on 2026-10-27: with
`NotoColorEmojiLegacy.ttf` moved out of `/system/fonts`, only the COLRv1 `NotoColorEmoji.ttf` is
left (tables COLR, CPAL and glyf, no CBDT), and a chat's preview shows "recibido ? te leo.". With
the file back, it shows 👍. R1's capture is taken on that emulator with the file moved out; the
author confirms on the Pixel afterwards.

That check found a small slip too: `colourEmojiFiles` takes any file whose name has "emoji" and
"color", whatever its extension, so a file renamed `.ttf.hidden` is still read. It should take
font files only (`.ttf`, `.otf`, `.ttc`, `.otc`), as `isFontFile` does.

## Tasks

On `feature/android-emoji` (from `feature/browser-emoji`), one commit per task, no push. The
`go-renderer` main tree's uncommitted `android/java/TextInput.java` is not touched: this work lives
in the `go-renderer-browser-emoji` worktree.

- [ ] T1 — Colour emoji fonts are font files only · `opengl/overlay/emoji_files.go` · R7 · test `a renamed emoji font is not read`
- [ ] T2 — A chain of emoji sources and the hook: `UseSystemEmoji(drawer)` sets a source asked after the font files; the platform source turns the drawer's picture into an `emojiGlyph` · `opengl/overlay/emoji.go`, `emoji_system.go` · R2–R5 · tests `the font file's emoji wins over the system's`, `an emoji the fonts lack is asked of the system once`, `a text with a system emoji is as wide as its parts`, `without either, the character is drawn as a question mark`; `testing.AllocsPerRun` on a second frame
- [ ] T3 — `EmojiPicture.java`: one static method that draws a character at a size in pixels with `Paint` and `Canvas` into a `Bitmap`, twice in two colours, and returns the ARGB pixels and the four measures, or null for no colour emoji; added to `java.Sources` (go-ui-apk compiles every entry) · `android/java/` · R1, R5 · a test like `screenreader_nodes_test.go`'s that the class name and package agree
- [ ] T4 — The JNI side: the class is loaded on the interface thread when the activity is made (`load_package_class`, a global reference); the call attaches the GL thread to the Java machine when it is not, and never throws into Go; Go undoes Android's premultiplied alpha into an `image.NRGBA`; the `android` package calls `overlay.UseSystemEmoji` at start · `android/emoji.c`, `android/emoji.go` · R1, R4 · build for android/arm64 and android/amd64; by hand on the emulator
- [ ] T5 — Check and docs: the emulator capture of R1 (Android 14, `NotoColorEmojiLegacy.ttf` moved out of `/system/fonts`) next to the same frame with it in place, in `docs/images/`; the overlay guide and the Android guide say where emoji come from on each platform; README's AI disclosure kept true · `docs/` · R1, R6 · captures; the docs build
- [ ] T6 — Hand-over: the chat's `go.mod` already points at this worktree; deploy-manager checks the preview and the bubble on its emulator without the CBDT font, then rebuilds the APK (arm64, x86_64) and publishes it, keeping the old one · · by hand; the author confirms on the Pixel

## Result

*Filled at the end.*
