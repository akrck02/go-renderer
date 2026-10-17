# Tutorial: an island with a lighthouse

![The quest map: Mason, Clockmaker, Weatherman, Herald, Gardener, Librarian, then the final challenge, Lighthouse keeper](../images/tutorial/quest/quest-map.svg)

In six short lessons you build one program, a step at a time: a hill rising out of the sea, a
lighthouse whose lamp turns, a forest that sways in the wind, the sky by day and by night, labels
over things, and trees you plant with a click. Each lesson adds a little to the program of the one
before it and ends with something you can run.

![The island at the end of lesson 4: a lighthouse among trees, a label over it, a header, a notice and a card of the program's own](../images/tutorial/tutorial-4.jpg)

**Before you start**, follow [Get started](../2-get-started.md): Go, a C compiler and go-renderer
cloned beside your module. Each lesson takes ten to twenty minutes.

Every lesson has the same shape, as in a quest: **the idea** in everyday words, **the code**, a **checkpoint** to know it works, a **bonus** with its solution folded, and an **achievement** to unlock. A final challenge waits at the end of lesson 6.

| Lesson | You learn | You end with |
|---|---|---|
| [1. Meshes and nodes](1-meshes-and-nodes.md) | Triangles, normals, materials, placing things with a transform | A tower on a hill |
| [2. Making things move](2-making-things-move.md) | The update function, children of a node, instancing a forest | A lamp that turns, a forest |
| [3. Light, sky and water](3-light-sky-and-water.md) | The environment: sun, fog, sea, wind, clouds, the night, glowing materials | An island in the sea, by day and by night |
| [4. Labels and the overlay](4-labels-and-the-overlay.md) | Labels over the world, the header, notices, free 2D drawing | Names and messages on screen |
| [5. Editing with the pointer](5-editing-with-the-pointer.md) | `viewer.Edit`, the input state, finding the ground under the pointer | Trees planted where you click |
| [6. Scenes from files](6-scenes-from-files.md) | glTF, the renderer's extras, embedding a scene in the program | The same island, made in another tool |

Every lesson closes with **practice**: a small change to try on your own, with a solution folded
under it. The [complete program](5-editing-with-the-pointer.md#the-whole-program) is at the end
of lesson 5.

> 💡 **Captures.** Every program built on `viewer` takes the viewer's flags. `-capture shot.png`
> draws one frame into a picture and exits, which is how the pictures in these lessons were made;
> `-time 3` draws it as if three seconds had passed, so two captures can be compared.

---

[← Get started](../2-get-started.md) · [Lesson 1 →](1-meshes-and-nodes.md)
