# 3. Emoji in the browser

> Status: building · Author's approval: proposal ☑ (2026-10-27) plan ☑ (2026-10-27)

## Why

On Android and the desktop the overlay draws emoji in colour, read from the system's colour emoji
font. In a browser, a WebAssembly program cannot read the system's fonts, so every emoji is drawn
as `?`: `drawable()` in `opengl/overlay/layout.go` swaps each character no font has for a
question mark. The same goes for Chinese, Arabic and every other script the Go fonts lack.

A program built for both platforms then shows two different texts. A chat page made with Go UI
(LLM-CP's `llmcp-chat`) shows a message sent as "emoji test 😀👍" as "emoji test ??". The text
itself arrives intact; only the drawing loses it.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | In a browser, an emoji of one character (😀, 🚀, 👍) is drawn in colour, at the text's size, on the text's baseline | headless Chromium capture of a label "North light 🌊 😀 🚀" next to the same frame on the desktop |
| R2 | An emoji is measured the same as it is drawn: text with emoji wraps, centres and ellipsises right | test: the width of "a😀b" equals the sum of the three advances; capture of a wrapped line |
| R3 | A character followed by `U+FE0F` (❤️) is drawn as an emoji, as on the other platforms | the capture of R1 holds "❤️" |
| R4 | Each emoji is made once per size and kept; a frame that shows emoji already made allocates nothing new | test: drawing the same label twice makes its emoji once |
| R5 | A browser without the means to draw one (no 2D canvas) still draws `?`, as today, and nothing fails | test with the source unavailable: the text is drawn with `?` |
| R6 | Android and the desktop draw emoji exactly as today | the overlay's tests and the Android emulator capture of the overlay guide, unchanged |

## Out of scope

- **Sequences joined by U+200D** (a family, a skin tone, most flags of more than two letters):
  they show as their parts, as on the other platforms today.
- **Other scripts** (Chinese, Arabic) in the browser. The same idea would serve them. They are a
  second proposal if wanted, so this one stays small.
- **Typing emoji** in Go UI's own text field in a browser. Go UI's pages type in a plain HTML
  field today. Typing on Android is someone else's work in progress (`android/java/TextInput.java`).

## Design

**Let the browser draw each emoji.** The browser has the system's emoji font, so it can paint an
emoji into a small 2D canvas. The program then reads that canvas's pixels back into Go as an
`image.Image`, made once per character and size. That picture enters the existing path: the
picture atlas, in the same draw call as the text, faded as the text is. Only where the picture
comes from changes.

There is a single seam. The painter (`drawing.go`) and the `Measurer` both ask
`fonts.colourFor`, which asks `emojiFont.glyph`. So measuring and drawing agree by construction
(R2), and nothing above the overlay changes: Go UI, its `Measurer` and the backends stay as they are.

- `emojiFont` keeps what it does today: the cache of glyphs by character and size, and the
  mutex. What it loses is reading the font files: it asks an **`emojiSource`** for a character's
  picture and metrics at a size, and the font-file reading moves behind that source.
- **The font files' source** (`emoji_files.go`, every build but `js && wasm`) is today's code, moved
  as it is: `colourEmojiFiles`, `parseColourFont`, the scaling.
- **The browser's source** (`emoji_browser.go`, `js && wasm`) draws on an `OffscreenCanvas` (or a
  `<canvas>` when there is none). It sets the font to the systems' emoji families (`Apple Color
  Emoji`, `Segoe UI Emoji`, `Noto Color Emoji`) at the size in pixels. `measureText` gives the
  advance and the ascent; `fillText` and `getImageData` give the pixels as an `image.NRGBA`. A
  canvas left empty means the system has no picture of the character, so no glyph. So does one
  the browser draws in the fill's colour: it is drawn twice, in two colours, and a colour emoji
  keeps its own. Where there is no canvas at all (Node), the font files are read, as on other
  systems; with none in reach, the text keeps its `?` (R5). *Changed while building: the plan had
  no source there, but Node can read files, and the overlay's tests run in it.*
- `fallback.go` picks the source when it makes the `emojiFont`: the browser's under `js && wasm`,
  the files' elsewhere. `Fonts.Folders` keeps its meaning for the files' source; in a browser it is
  ignored, and the docs say so.

**Not taken:**

- **Ship a colour emoji font with the page** (Noto Color Emoji, about 10 MB): a heavy download
  for every page, and its emoji would look unlike the system's own.
- **Draw the text with HTML over the canvas:** it would leave the overlay's single draw call, and
  break clipping, fading and layering.
- **Ask the browser for every missing character, not only emoji:** it would serve Chinese and
  Arabic too, but it changes how all fallback text is drawn; left for its own proposal.

## Tasks

- [x] T1 — Put the emoji's pictures behind an `emojiSource`; move the font-file reading into
  `emoji_files.go` unchanged · `opengl/overlay/emoji.go`, `emoji_files.go`, `fallback.go` · serves
  R4, R6 · tests: `TestAnEmojiIsMadeOncePerSize` (a fake source counts its calls), and the overlay's
  tests as they are
- [x] T2 — Measure with the source's advance: a text with emoji is as wide as its parts · serves
  R2 · test: `TestATextWithAnEmojiIsAsWideAsItsParts` (fake source, "a😀b")
- [x] T3 — The browser's source: draw on a 2D canvas, read the pixels, an empty canvas is no
  glyph · `opengl/overlay/emoji_browser.go` (`js && wasm`) · serves R1, R3, R5 · test under
  `GOOS=js` in Node, which has no canvas: `TestWithoutACanvasAnEmojiIsDrawnAsAQuestionMark`
- [ ] T4 — A page to look at: a label with "North light 🌊 😀 🚀 ❤️", captured in headless Chromium
  and on the desktop · an example under `cmd/` · serves R1, R3 · check: the two captures side
  by side in the proposal's Result
- [ ] T5 — Docs: the overlay guide's emoji paragraph says where a browser's emoji come from, and
  the reference says `Fonts.Folders` is ignored in a browser · `docs/4-guides/3-overlay.md`,
  `docs/5-reference/15-opengl-overlay.md` · serves R1–R6

Each task leaves the project working: T1 is a pure move, and T3 is the first to change what a
browser shows. `go vet ./...` and the overlay's tests run after each one, under `GOOS=js` too.
R6 still needs a person: the Android emulator capture of the overlay guide, after T1.

## Result

*Filled at the end.*
