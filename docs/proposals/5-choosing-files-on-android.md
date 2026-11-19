# 5. Choosing files on Android

> Status: built, pending review · Author's approval: proposal ☑ ("approved", 2026-10-29) plan ☑ ("approved", 2026-10-29)

## Why

An app drawn with Go Renderer cannot let a person pick a file on an Android phone. Go UI's
`window.ChooseFiles` opens the browser's file chooser on the web, and says "none here" on
Android, because Go Renderer has no way to open the system's one.

llm-center is the first to need it: its author attaches a screenshot or a log to an answer from
the phone, and today has to leave the app for the page in the phone's browser. Any app that takes
a picture, a document or a backup from the person needs the same.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | **Open the chooser.** `android.ChooseFiles` opens Android's own document chooser, for one file or several, over the app | by hand on the emulator: the chooser opens, the app is still there when it closes |
| R2 | **Hand back names and bytes.** When the person chooses, Go gets each file's display name, its size, and its bytes; when they cancel, an empty list | by hand on the emulator with a picture and a text file; a test of the Go side with the Java answer faked |
| R3 | **Never read a file whole that is too big.** The caller says the largest size it takes; a bigger file comes back with its name and size, unread, as Go UI's `ChosenFile` already says (`Read: false`) | test: a file over the limit comes back unread |
| R4 | **One at a time, safe from any goroutine.** A second call while the chooser is open returns false; the answer comes on another goroutine, as Go UI's does on the web | test of the Go side |

## Out of scope

- Choosing a folder, or saving a file (`ACTION_CREATE_DOCUMENT`): their own change when an app
  needs them.
- Filtering by type (only pictures, say): every type is offered, as on the web today. A `types`
  argument can come when a caller needs it.
- Go UI's Android side (`window.ChooseFiles` calling this): Go UI's own change, once this is in.

## Design

Android hands back what a person chose in `onActivityResult`, a method of the activity. Go
Renderer runs inside `NativeActivity`, which does not pass that method on to native code. So the
chooser needs a piece of Java that is told the answer.

**A headless fragment.** A small Java class, `FileChooser`, joins the activity as a fragment with
no view. It starts the chooser (`ACTION_OPEN_DOCUMENT`, `CATEGORY_OPENABLE`, `*/*`,
`EXTRA_ALLOW_MULTIPLE`), receives `onActivityResult`, and for each chosen `Uri`:

1. asks the `ContentResolver` for its display name and size (`OpenableColumns`);
2. reads its bytes through `openInputStream` only when the size is within the caller's limit;
3. calls Go once, through a `native` method as `TextInput` does, with the names, sizes and bytes.

Then it removes itself. The reading runs on a background thread, never on Android's main thread,
so a large file does not freeze the app.

**On the Go side,** `android.ChooseFiles(largest int64, chosen func([]ChosenFile)) bool`:

- `ChosenFile{Name string; Size int64; Content []byte; Read bool}`, the same shape as Go UI's.
- It returns false if a chooser is already open, or the package has none. Opening itself happens
  later, on Android's main thread: if it fails there, `chosen` is called with nothing.
- `chosen` is called once, on the goroutine that received the answer from Java; the caller copes
  with that as it does with the web's, by taking its own lock.

The C between them follows `services.c`: `attach_thread`, `cleared` after each JNI call, local
references deleted.

**Not taken.**

- *A second activity in the manifest*, transparent, that only opens the chooser: it works on
  every Android, but every app would need its manifest changed, in Go UI for Android's
  `go-ui-apk` as well. The fragment needs nothing outside Go Renderer.
- *`android.app.Fragment` is deprecated* since Android 9, in favour of the support libraries'
  fragment. The support libraries are a dependency Go Renderer does not have, and the platform's
  fragment still works on every Android an app of ours runs on. If it ever stops, the second
  activity above is the way out.
- *Passing a `Uri` to Go and reading it later*: the permission to read it lasts only as long as
  the activity, and Go would need JNI for every read. Reading at once is simpler.

## Tasks

On `feature/file-chooser` (from `main`), one commit per task, no push.

- [ ] T1 — `ChosenFile` and the Go side: `android.ChooseFiles(largest, chosen)` keeps the one open request, refuses a second, and turns Java's answer (names, sizes, bytes or none) into `[]ChosenFile`, unread above `largest` · `android/files.go` · R2–R4 · tests `a second chooser while one is open is refused`, `a file over the limit comes back unread`, `a cancelled chooser hands back no files`
- [ ] T2 — `FileChooser.java`: the headless fragment that opens `ACTION_OPEN_DOCUMENT`, reads each `Uri`'s name and size through `OpenableColumns`, its bytes on a background thread when within the limit, calls Go once and removes itself; added to `java.Sources` · `android/java/` · R1–R3 · the test that the class name and package agree, as for the other Java classes
- [ ] T3 — The JNI side: the class found and its native method registered as `textinput.c` does; the call from Go attaches the thread, never throws into Go, and frees its references · `android/files.c` · R1, R4 · build for android/arm64 and android/amd64
- [ ] T4 — By hand on the emulator: a small test app opens the chooser, takes a picture and a text file, and cancels once; a capture of the chooser over the app in `docs/images/` · R1, R2 · emulator capture
- [ ] T5 — Docs: the Android guide gains "Choosing files", with the limits and why a fragment; the reference lists `ChooseFiles` and `ChosenFile`; README's AI disclosure kept true · `docs/` · the docs build

## Result

Built and tried on the Android 14 emulator. Pending the author's review.

**Tests.** `go test ./android` covers the Go side: a second chooser is refused while one is open, a
file over the limit comes back unread, a cancel hands back nothing, a chooser that could not open
lets the next one, a late answer is never mixed into the next chooser's files, and the Java class
is where its name says and is given to the tool. `go vet` and the build pass for android/arm64 and
android/amd64; the Java compiles against Android 36's `android.jar` with every warning on.

**By hand,** with a throwaway app on the emulator: one file (read, with its bytes); a picture and a
2 MB file together under a 1 MiB limit (the picture read, the other unread with its real size); a
cancel, then the chooser again; and the same with "Don't keep activities" on, so the activity is
rebuilt behind the chooser: one answer, and no second chooser.

**Fixed after the review.**

- A rebuilt activity no longer opens a second chooser: the fragment saves that it is open.
- Go is always told the end: when the activity is going, its state is saved, or the fragment
  cannot join, the choice ends empty; and when the activity ends for good before an answer.
- Files go to Go one at a time as they are read, so many photos never sit in Java's memory at
  once; running out of memory while reading leaves that file unread instead of ending the app.
- A file too big whose real size is not known says `UnknownSize`, never a made-up one; a file that
  could not be read says a size within the limit, and the docs say how to tell them apart.
- A name Java could not copy reaches Go empty instead of stopping the program.
- An answer that comes when no chooser is open is dropped.
- Go's callback never runs on Android's main thread.

**Kept as it was.** `ChooseFiles` answers with a bool, not an error code: it mirrors Go UI's
`ChooseFiles`, and its caller does the same thing for either reason (it hides its button).

**Still open.** Go UI's Android side, which calls this, is Go UI's own change; until it is made,
llm-center's paperclip stays hidden in the app. Process death while the chooser is open was not
tried.
