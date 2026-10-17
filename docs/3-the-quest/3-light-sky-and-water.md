# Lesson 3: light, sky and water

So far the island floats in grey. In this lesson you fill `scene.Environment`: the sea around it,
fog in the distance, wind in the trees, clouds, and the night, with a lamp that glows in it.

| Mission | Pieces | Time |
|---|---|---|
| The sea, the wind, clouds and the night | `scene.Environment`, `Water`, `Sky`, `KindGlow` | 15 min |

#### The idea

The scene's **weather report**: one struct says where the sun is, how far the fog reaches, how high the sea comes and which way the wind blows. There is no clock in it: **the sun's height is the time of day**, as a sundial reads it.

## The environment

Everything that is not a node lives in `world.Environment`: the sun, the sky, fog, water, wind,
snow and the season's tint on leaves. `scene.New()` starts from `scene.DefaultEnvironment()`, a
clear afternoon; you change only what you need.

### The sea

`Water` is an endless animated plane at a height. It is coloured by how deep the water is over the
ground below it, so the shore shines turquoise and the open sea turns dark blue on its own:

```go
world.Environment.Water = &scene.Water{
	Level:   1,    // one metre above zero: the hill's edges go under
	Size:    2000, // how far the plane reaches
	Shallow: graphics.Vec4{0.1, 0.5, 0.55, 1},
	Deep:    graphics.Vec4{0.02, 0.12, 0.25, 1},
}
```

### Fog and wind

```go
world.Environment.FogColor = graphics.Vec4{0.75, 0.8, 0.85, 1}
world.Environment.FogNear, world.Environment.FogFar = 80, 400 // FogFar 0 means no fog
world.Environment.Wind = scene.Wind{Direction: graphics.Vec4{1, 0, 0.3, 0}, Strength: 0.5}
```

The wind moves the clouds and bends every material with a `Sway`. Plants are modelled with their
base at the origin and a height of one; the higher a vertex, the more it bends:

```go
forest.Mesh.Material.Sway = 0.3
```

### Clouds

`scene.Sky` describes the sky over a whole day: stars, a moon with phases, clouds, constellations,
or pictures (a skybox). Start from Earth's:

```go
sky := scene.DefaultSky()
sky.Clouds = scene.Clouds{Coverage: 0.4, Color: graphics.Vec4{1, 1, 1, 1}, Speed: 0.01, Scale: 1}
world.Environment.Sky = sky
```

![The island in the sea in the evening light, the forest's shadows long](../images/tutorial/tutorial-3.jpg)

## Day and night

There is no clock in the renderer: **the sun's height is the time of day**. Above the horizon it
is day, near it twilight, below it night, when the moon lights the scene and the stars turn about
the pole. `Environment.SunDirection` points towards the sun; the viewer's keys `I` `K` `J` `L` move
it and `N` jumps between day and night, and the flags `-night`, `-sun-elevation` and `-sun-azimuth`
start there.

A lighthouse should shine at night. A **glowing** material is lit like any other by day and shows
its own colour in the dark:

```go
lamp.Mesh.Material.Kind = scene.KindGlow
```

```bash
go run . -night
```

![The island at night, the lamp glowing yellow on the dark tower](../images/tutorial/tutorial-3-night.jpg)

### Material kinds

| Kind | Shading |
|---|---|
| `KindLit` | Sun or moon, sky light, shadows, fog: most things |
| `KindUnlit` | Its colour as it is (plus fog): flags, markers, things that should not darken |
| `KindGlow` | Lit by day, shining in the dark: lamps, windows, glowing flowers |
| `KindWater` | An animated water surface: a river or a lake above the sea's level |
| `KindWaterfall` | Translucent falling water |

#### Checkpoint

The hill rises out of a sea, turquoise at the shore and dark further out; the trees sway; with `-night`, the lamp glows yellow over the dark island.

![Achievement unlocked: Weatherman](../images/tutorial/quest/achievement-weatherman.svg)

## What you learnt

- `Environment` holds the sun, the sky, fog, water, wind and snow.
- The sun's height is the time of day; the night comes with the moon and the stars.
- `Sway` bends plants in the wind; `KindGlow` shines at night.

## Bonus (optional)

Give the island a winter: snow above two and a half metres, fading in over half a metre, and
foliage turned a little brown.

<details>
<summary>🔓 Show the solution</summary>

```go
world.Environment.Snow = &scene.Snow{Level: 2.5, Blend: 0.5, Color: graphics.Vec4{0.95, 0.97, 1, 1}}
forest.Mesh.Material.Foliage = true // takes the season's tint
world.Environment.FoliageTint = graphics.Vec4{0.55, 0.45, 0.3, 0.5} // the colour, and how much
```

Snow settles on surfaces facing up, so the tower's walls stay clear.

</details>

---

[← Lesson 2](2-making-things-move.md) · [Lesson 4: labels and the overlay →](4-labels-and-the-overlay.md)
