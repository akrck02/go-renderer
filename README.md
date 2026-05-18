# Go Renderer

This is a tiny, educational graphics renderer built in Go. The core goal of this project is to explore and implement complex graphics pipelines from scratch, while exposing a clean, simple contract (API) for building applications on top.

> ⚠️ **Warning:** This software is an educational experiment and is absolutely **not suitable for production use**. APIs will break, performance is unoptimized, and errors are highly probable.

##  Project Goals
* **Understand the Pipeline:** Deep-dive into vertex processing, clipping, rasterization, and fragment shading.
* **Clean Abstraction:** Create an intuitive, idiomatic Go API that hides backend rendering complexity from the application developer.
* **Minimal depencies:** Build core math and rendering concepts with minimal dependencies to fully understand the underlying mechanics.

## Roadmap
This are the goals for the next versions of the library. The feature implementation order can change.

### v1.0.0 Prototype
* Local space, world space and viewport space coordinate transformations.
* Render of 2D objects on world space
* Render of 2D objects on viewport space
* Render of 3D objects on world space
* Render of 3D objects on viewport space
* Camera movement
* Input bindings

### v2.0.0 Support expansion
* Metal support (Apple silicon)
* Vulkan support
* WebGPU support
