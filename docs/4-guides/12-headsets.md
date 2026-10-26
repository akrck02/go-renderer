# 🥽 Drawing for a headset

**Status:** new (October 2026), not yet seen in a headset. The pieces below are plain Go and are
checked by unit tests (the maths) and by the recording fake of `gles` (which calls are made); how
the picture looks in a real headset still needs a look.

A headset is not one screen but two: each eye gets its own picture, drawn from where that eye is,
with its own, lopsided field of view. The runtime (OpenXR, for example) tells the program every
frame where the eyes are and how wide they see, and hands it a framebuffer per eye to draw in.
This page is about the go-renderer side of that: turning what the headset reports into matrices,
and drawing one scene into several pictures without planning it several times.

## Poses and fields of view

The headset reports things in its own words, and `graphics` has a type for each:

| Type | What it is |
|---|---|
| `graphics.Quaternion` | A rotation as x, y, z, w, the way OpenXR and glTF write it. `Matrix`, `Rotate` and `Multiply` turn things with it. |
| `graphics.Pose` | Where something is (in metres) and which way it is turned: an eye, the head, a hand. `Matrix` puts the thing in the room; `ViewMatrix` is the other way round, the view matrix of an eye. |
| `graphics.FieldOfView` | How far an eye sees to the left, right, up and down, as four angles (left and down negative). |

`graphics.PerspectiveFromFieldOfView(field, near, far)` makes an eye's projection from those four
angles. A headset's eyes see more to the outside than towards the nose, so the picture is
off-centre: unlike `PerspectiveOpenGL`, the middle of the picture is not straight ahead. When the
angles happen to be symmetric, the two give the same matrix.

## The rig: where the room is in the world

The headset knows where you are *in your room*; the scene does not care about your room. A
`camera.Rig` says where the room stands in the scene:

```go
rig := camera.Rig{
	Position:   graphics.Vector{40, 0, -10, 1}, // the room's floor, in the world
	YawRadians: 0.5,                            // the room turned left a little
	Scale:      1,                              // scene units per metre (0 also means 1)
}
```

Walking in the room changes the poses the headset reports; walking with a stick, or teleporting,
moves the rig. Turning with the stick changes `YawRadians`. `Scale` above 1 makes you a giant: a
step crosses more of the scene and your eyes are further apart in it, so the world looks like a
model on a table. That is a handy way to look over a whole map.

With a rig, each eye becomes something the renderer understands:

- `rig.EyeView(eye)` is the eye's view matrix, from the world to the eye. It has no scale in it, so
  near and far stay in scene units, like a `Lens`'s.
- `rig.EyePosition(eye)` is the eye in the world, which lighting, fog and water need.
- `rig.Place(hand)` is the model matrix for a tracked thing, such as a controller. It keeps the
  rig's scale, so a hand modelled in metres looks hand-sized to a giant too.

### Walking with the sticks: RigWalk

`camera.RigWalk` moves the rig from a pad's sticks, as a headset's controllers give them. Call it
once a frame, with the head's pose as the headset reports it:

```go
walk := camera.RigWalk{Ground: ground} // optional: keep the floor on the ground

// every frame:
walk.Update(state, seconds, head, &rig)
```

- The left stick walks the way the head looks, flat on the ground: looking up does not lift you.
  A full stick is `SpeedMetres` a second (1.5 by default), multiplied by the rig's `Scale`, so a
  giant crosses the map as fast as its size suggests.
- The right stick turns in steps (`TurnRadians`, π/6 by default): one step each time it is pushed
  most of the way, the next only after it came back near the middle. The turn is around the head,
  so the player stays where they stand. A smooth turn the head does not make makes many people
  sick; a step does not.
- With `Ground`, the room's floor is put on the ground under the head at every update, where there
  is ground; elsewhere it keeps its height.

The program moves the room; only the player moves the head.

## One plan for both eyes: CoveringLens

Before go-renderer draws a frame it *plans* it: which meshes are in view, how detailed each one
must be, in what order the translucent ones go. Planning twice, once per eye, would cost twice and
could even choose different details for each eye, which the eyes would notice as shimmer.

`camera.CoveringLens(rig, eyes, fields, near, far)` returns one ordinary, symmetric `Lens`, with
the aspect of its picture, that sees everything both eyes see. The frame is planned once from it,
and drawn twice.

```text
   \                 /     the covering lens's view
    \  \         /  /
     \  \       /  /       each eye's outer edge
      \  \     /  /
       \  L-+-R  /          the eyes; + is the middle between them
        \       /
         \     /
           C               the covering lens, a little behind
```

It looks the way the head looks, with angles as wide as the widest each eye sees. It stands on the
line through the middle of the eyes, but a little *behind* them: a camera right between the eyes
would need a much wider angle to take in what each eye sees right in front of its nose. A few
centimetres back (times the rig's scale), both eyes' views fit in the angles they already have, so
culling stays as tight as for a single eye. The tests check that every corner of each eye's view,
from the near plane to the far one, is inside the covering lens, also with the head turned, tilted
or looking up.

## Drawing the views: DrawViews

`gles.SceneRenderer.DrawViews` draws the planned frame in every view:

```go
covering, aspect := camera.CoveringLens(rig, eyePoses, eyeFields, 0.05, 2000)
views := make([]gles.View, len(eyePoses))
for index, eye := range eyePoses {
	views[index] = gles.View{
		Target: eyeFramebuffers[index], // with a depth buffer
		Width:  eyeWidth, Height: eyeHeight,
		View:       rig.EyeView(eye),
		Projection: graphics.PerspectiveFromFieldOfView(eyeFields[index], 0.05, 2000),
		Eye:        rig.EyePosition(eye),
	}
}
renderer.DrawViews(world, covering, aspect, views)
```

For each view it binds the target, sets the viewport, clears, and draws the same commands with the
view's own matrices; lighting, fog and the water's sky reflection are worked out from the view's
`Eye`. Two views may share one target side by side, each in its own viewport: the target is then
cleared only before the first of them, so the second does not wipe the first out.

There is no overlay in `DrawViews`: no labels, header, notices or draw list. They are pinned to
the corners of a screen, and a headset has no screen to pin them to. Panels in a headset belong in
the scene, at a distance from the eyes. `Draw`, for screens, is unchanged.

## Colour: LinearOutput

The scene shader works in linear colour and, at the very end, raises it to the power 1/2.2 to
suit a screen. Headset runtimes usually hand out sRGB framebuffers, which do that step themselves
when the colour is written. Doing it twice washes the picture out: everything looks pale and grey.

Set `renderer.LinearOutput = true` when the target encodes sRGB itself. The shader then leaves the
colour linear (the exponent is a uniform, so it is the same program either way) and the clear
colour, the horizon, is linear too. Leave it false for a screen, a phone's window or an `RGBA8`
framebuffer.

`gles.FlatRenderer` has the same switch for an interface drawn into such a target (a headset's
panel, below): its colours are a screen's, so with `LinearOutput` the quads' shader raises them to
the power 2.2, and the background with them, before the target encodes them back.

## OpenXR

**Status:** new (October 2026), built and unit-tested, **never run on a headset**: no headset was
connected while it was written. The session's states and the controllers' mapping are tested on a
computer; everything that talks to a runtime has only been compiled for Android.

Package `openxr` is the other half: it asks the headset's runtime where the eyes are, and gives it
the pictures back. It speaks OpenXR, the Khronos standard that a Meta Quest's runtime (and most
others) understands.

### Opening a session

```go
android.RunHeadset(android.HeadsetOptions{}, func(headset android.Headset, functions android.GL) bool {
	if session == nil {
		opened, err := openxr.Open(headset, openxr.Options{ApplicationName: "Example"})
		if err != nil {
			log.Print(err) // openxr.LoaderNotFound, openxr.HeadsetNotFound, …
			return false
		}
		session = opened
		renderer.LinearOutput = session.SRGB()
	}
	session.PollEvents()
	if session.Finished() {
		session.Close()
		return false
	}
	frame, err := session.BeginFrame()
	if err != nil {
		session.Close() // see "Leaving" below for a gentler way out
		return false
	}
	session.ReadControls(frame).Apply(state)
	if frame.ShouldRender {
		drawTheEyes(frame) // gles.View per eye: frame.Eyes[i].Target, Width, Height, Pose, Field
	}
	if wantsToLeave {
		session.RequestExit() // then keep polling and ending frames until Finished
	}
	return session.EndFrame(frame) == nil
})
```

`Open` belongs inside the frame function: the session is made for the OpenGL ES context that the
drawing thread holds, and the eyes' framebuffers are objects of that context. It does, in order:

1. Opens the loader, `libopenxr_loader.so`, **when the program runs**. Nothing links against it, so
   an application without it still starts and gets `openxr.LoaderNotFound` instead of crashing. The
   loader comes from the headset's maker and ships inside the application, not in this repository.
2. Hands the loader the Java VM and the activity (Android needs them before anything else), and
   starts the runtime with the extensions for Android and OpenGL ES, plus a floor-level space when
   the runtime offers one.
3. Finds the headset and checks that the context's OpenGL ES is new enough for the runtime.
4. Makes the session, a space for the scene and one for the head, a swapchain per eye, and the
   controllers' actions.

### The session's states

A runtime does not start showing a program straight away. It sends states, and the program
answers some of them:

```text
 IDLE ─▶ READY ─▶ SYNCHRONIZED ─▶ VISIBLE ─▶ FOCUSED
          │ begin                                │
          ▼                                      ▼
       running                        STOPPING (end) ─▶ IDLE ─▶ EXITING
```

`PollEvents` reads them once a frame: on READY it begins the session (now `Running`), on STOPPING
it ends it, and EXITING, or the runtime about to go away, makes `Exiting` true: time to close and
leave. `Focused` is true only in FOCUSED: when the system's menu is over the program, the frames
are still shown but the controllers are not the program's.

### Leaving

A program that wants to stop does not close a running session: it asks the runtime with
`RequestExit`, and the runtime answers as if it were leaving on its own, STOPPING (the session is
ended) and usually EXITING. Until then the loop keeps calling `PollEvents` and handing frames back,
empty ones will do. `Finished` turns true once the runtime asked the program to leave (`Exiting`) or
the session it asked to stop has ended; then `Close`. A session that was lost has nothing to wait
for: close it straight away. A program should also stop waiting after a moment (a couple of
seconds) in case a runtime never answers.

While the session is not running, `BeginFrame` waits a short moment and returns an empty frame, so
a loop that always calls `PollEvents`, `BeginFrame` and `EndFrame` neither spins nor breaks.

### A frame

`BeginFrame` waits for the runtime's moment (this is what paces the program to the headset's
refresh rate), finds both eyes and the head at the time the frame will be shown, and takes a
picture from each eye's swapchain. Each `Eye` has a `Pose` and a `Field` for the maths above, and a
`Target`: a `gles.Framebuffer` with colour and a depth buffer, `Width` × `Height` pixels, the size
the runtime recommends.

The head is found every frame, drawn or not. `Head` is the identity pose (at the origin, looking
down -Z) until the runtime knows where it is, never a zero rotation. `HeadTracked` is true only when
the runtime knows both where the head is and which way it looks: wait for it before placing
anything in the room by the head, or it lands where a stale or made-up position put it.

`EndFrame` gives the pictures back and shows them as one projection layer. When `ShouldRender` is
false (the headset is off the face, or the runtime does not know which way the eyes look) draw
nothing, but still call `EndFrame`: every frame begun must end.

**Space.** The scene's space is the floor where you stand when the runtime has it (`LOCAL_FLOOR`),
else the play area's floor (`STAGE`), else where the head was when the session began (`LOCAL`,
with no floor: y 0 is then eye height). Poses are in metres.

**Colour.** The swapchains are sRGB when the runtime offers it, which a Quest does. `SRGB()` says
so; then set the scene renderer's `LinearOutput` (see *Colour* above).

### The controllers

`ReadControls(frame)` reads both controllers at the frame's display time, so a hand drawn in the
frame is where it will be when the frame is shown. Each `Hand` has a `Grip` pose (where it is held)
and an `Aim` pose (where it points), `Tracked`, the `Trigger` and `Squeeze` from 0 to 1, the stick
and its click, and the two face buttons. The actions are bound to Meta's Touch controllers and, for
anything else, to the simple controller every runtime knows (poses, select as the trigger, menu).

`Controls.Apply(state)` turns them into the same `input.State` a game pad fills, so a camera written
for a pad works in a headset:

| Controller | `input.State` |
|---|---|
| Left stick | `Stick()` |
| Right stick | `RightStick()` |
| A, B (right) | `PadA`, `PadB` |
| X, Y (left) | `PadX`, `PadY` |
| Stick clicks | `PadLeftThumb`, `PadRightThumb` |
| Menu (left) | the system's back (`TakeBack`) |

A stick near its centre reads as 0, so a camera does not drift. Triggers, grips and poses have no
place in `input.State`; read them from the hands.

### Panels

A panel is a flat picture the runtime places in the world itself, over the eyes' pictures: a
*quad layer*. It is how an interface is shown in a headset. The runtime samples it straight into
each eye, so text stays sharper than the same picture drawn as a textured square in the scene.

```go
panel, err := session.NewPanel(1024, 640) // pixels; once, after Open, on the drawing thread
panel.Pose = graphics.Pose{Orientation: graphics.IdentityQuaternion(), Position: graphics.Vector{0, 1.3, -1.5, 1}}
panel.FitWidth(1.1) // metres wide, as tall as its pixels' proportions say
panel.Shown = true

// in a frame that is drawn (ShouldRender), only when the picture changed:
target, err := panel.Target()
flat.Target, flat.LinearOutput = target, session.SRGB()
flat.Draw(list, background, panel.Width(), panel.Height(), 1)
```

- **Where.** `Pose` is the panel's middle, in the session's space (the same as the head's and the
  hands'); its picture faces +Z, toward whoever looks at it from that side. A panel left in that
  space stays put in the room while a rig walks the world around it.
- **When to draw.** `Target` takes the frame's image of the panel's swapchain, and `EndFrame` gives
  it back. The runtime keeps showing the last picture given back, so draw only when it changes. A
  panel never drawn into shows nothing.
- **Order.** Shown panels go after the eyes' projection layer, in the order they were made, the
  last made on top. They are blended by their alpha, premultiplied. `FlatRenderer` clears to its
  background colour with that colour's alpha, premultiplied: an opaque background fills the
  picture, and a see-through one (alpha 0) leaves the room or the scene showing wherever nothing is
  painted, so a label can be a rounded pill rather than a rectangle.
- **How many.** `MostPanels` (8) a session.
- **Format.** The eyes' (sRGB on a Quest): draw linear colour into it (`FlatRenderer.LinearOutput`).
- **Off the headset.** `DetachedPanel(width, height)` is a panel that can be placed but not drawn,
  for working out what points at it (tests, layouts). `PanelWithTarget(width, height, target)` is
  one drawn where the program says: a stand-in for a headset on a computer composes its panels
  itself, from framebuffers of its own.

### Vibration

`session.Vibrate(openxr.RightHand, amplitude, duration)` shakes a controller, amplitude from 0 to
1, at the frequency the runtime finds best; a new vibration replaces the one still going. It is
bound to the Touch controllers' and the simple controller's haptic outputs, and does nothing while
the program does not have the controls.

On a Meta Quest 3 (October 2026): a 1024 × 640 panel shown 1.5 m ahead, drawn by `FlatRenderer` in
linear colour, reads with its own colours (not washed out) at 72 frames a second.

### The room behind the scene

On a Meta Quest the scene can stand in the player's own room. Open the session with
`openxr.Options{Passthrough: true}`: when the runtime has `XR_FB_passthrough`, the session makes a
passthrough layer, and every frame puts it under the eyes' picture. Draw the eyes with
`gles.SceneRenderer.SeeThrough` while `Session.ShowsRoom()` is true: the renderer clears to
transparent instead of the horizon, so the room shows wherever nothing is drawn.

Meta's runtime has no `ALPHA_BLEND` environment blend mode, only `OPAQUE`: the passthrough layer is
how a Quest mixes the room in. The application's manifest must also declare
`com.oculus.feature.PASSTHROUGH` (go-meta-quest's does). Without the extension the session opens as usual, with nothing
behind the scene, and `ShowsRoom` is false. A runtime that has it but will not start it fails `Open`
with `openxr.passthrough-not-started`.

### Not checked yet

No headset was connected, so none of this has run. The pure parts are unit-tested on a computer
(the session's states played with made-up events, the controllers' mapping, the pose conversion);
the C side and the Go that calls it are compiled for Android arm64 and linked into a test library,
nothing more. The first run on a headset should look at: the session reaching FOCUSED, both eyes
showing a steady picture the right way up, the floor at the right height, and every button.
