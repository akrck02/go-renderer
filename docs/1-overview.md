<p align="center"><img alt="" src="images/logo.svg" width="120"></p>

# Welcome to go-renderer 👋

**go-renderer draws pictures with a computer's graphics card**: 3D scenes of islands, towns, forests, the sea and the sky, by day and by night, and the flat buttons and panels other programs put over them. It is written in Go, to learn how 3D graphics work from the inside.

<p align="center"><img alt="An island with a lighthouse among trees, a label over it and a few cards of text on screen" src="images/tutorial/tutorial-4.jpg" width="720"></p>

You describe what is in the scene (*a hill here, a tower there, the sun this high*) and go-renderer draws it, sixty times a second, as you fly round it, walk on it or look at it as a map. It can also open whole scenes made by other tools, in the glTF format.

## Is it for me?

- ✅ You want to **look** at a 3D scene: open one with the viewer, no code needed.
- ✅ You are **learning** how 3D graphics work: triangles, light, cameras.
- ✅ You write Go and want to **show** a scene, a map or a simulation.
- ❌ You need a finished game engine or a product to ship: this is an experiment.

## Where to start

Pick the line that sounds like you.

| If you… | Read | Time |
|---|---|---|
| just want to **see** what it does | [Try it without writing code](#try-it-without-writing-code) | 2 min |
| are **new** to it and want to install it | [Get started](2-get-started.md) | 15 min |
| want to **learn** step by step | 🏴‍☠️ [The quest](3-the-quest/0-the-quest.md): an island with a lighthouse, in 6 levels | 90 min |
| want to know **how one part works** | [The guides](4-guides/1-architecture.md): the viewer, labels, glTF, web, Android | — |
| are **looking up** a type or a function | [The reference](5-reference/0-packages.md) | — |

## Try it without writing code

With Go installed and the project copied (see [Get started](2-get-started.md)), one command opens a made-up island to fly round:

```sh
go run ./cmd/visor -sample island
```

Drag to turn, scroll to come closer, **Tab** to walk, **M** for a map and **N** for the night. Any `.glb` file opens the same way: `go run ./cmd/visor scene.glb`.

## All the pages, in reading order

The menu on the left (or the files in `docs/`) follows this order.

**Start**

- [Overview](1-overview.md): this page
- [Get started](2-get-started.md): install, the sample island, your first scene

**Learn**

- [The quest](3-the-quest/0-the-quest.md): six levels that build an island with a lighthouse

**Guides: how each part works**

- [Architecture](4-guides/1-architecture.md)
- [The viewer](4-guides/2-viewer.md): every key and option
- [The overlay](4-guides/3-overlay.md): labels, header, notices, flat drawing
- [glTF extras](4-guides/4-gltf-extras.md)
- [In a web browser](4-guides/5-web.md)
- [On Android](4-guides/6-android.md)
- [Metal](4-guides/7-metal.md)

**Reference**

- [Packages](5-reference/0-packages.md): every type and function

**The project**

- [Optimization plan](6-project/1-optimization-plan.md)
- [Roadmap](6-project/2-roadmap.md)

## Words you will meet

| Word | What it means |
|---|---|
| Go | A programming language. To try these projects you install it once, from go.dev, and type `go run …` in a terminal. |
| Terminal | The window where you type commands: *Terminal* on macOS and Linux, *PowerShell* on Windows. |
| Repository | A project's folder of files, kept on GitHub. `git clone <address>` copies it to your computer. |
| Scene | Everything a 3D picture holds: ground, houses, trees, the sea, the sky and the sun. |
| `.glb` file | A 3D scene in one file, in the open glTF format: many programs (Blender among them) open and write it. |
| Quest | The tutorial of these docs: short levels, each ending with something that works and an achievement. |

## The family

go-renderer is one of a family of small projects that work together:

| Project | What it does |
|---|---|
| **go-renderer** (this one) | draws 3D scenes and flat interfaces |
| [go-ui](https://github.com/akrck02/go-ui) | buttons, panels and whole applications |
| [go-ui-android](https://github.com/akrck02/go-ui-android) | Android apps that look like Android's own |
| [go-sound](https://github.com/akrck02/go-sound) | sound and music |
| [go-scene-builder](https://github.com/akrck02/go-scene-builder) | 3D scenes from a flat map |
| [go-simulator](https://github.com/akrck02/go-simulator) | simulations that change a scene |
| [go-simulator-terrain](https://github.com/akrck02/go-simulator-terrain) | ground shaped by rivers |
| [go-simulator-climate](https://github.com/akrck02/go-simulator-climate) | weather and biomes |
| [go-simulator-flora](https://github.com/akrck02/go-simulator-flora) | forests that grow |
| [go-simulator-urban](https://github.com/akrck02/go-simulator-urban) | towns that grow |
| [go-simulator-motion](https://github.com/akrck02/go-simulator-motion) | a scene that moves |
| [go-3ds](https://github.com/akrck02/go-3ds) | Go on the Nintendo 3DS |

> ⚠️ All of them are **educational experiments**, written with an AI assistant and pending review by their author: good for learning and playing, not for production.
