package nintendo3ds

// The vertex shaders in shaders/ are written in the PICA200's assembly (.v.pica) and assembled by
// devkitPro's picasso into the binaries (.shbin) the renderers embed. The binaries stay in git:
// go:embed needs them to build, and a program that imports this package should not need
// devkitPro. After changing a source, run go generate ./nintendo3ds with picasso on the PATH.
//
//go:generate picasso -o shaders/scene.shbin shaders/scene.v.pica
//go:generate picasso -o shaders/baked.shbin shaders/baked.v.pica
//go:generate picasso -o shaders/bakedtextured.shbin shaders/bakedtextured.v.pica
//go:generate picasso -o shaders/quad.shbin shaders/quad.v.pica
