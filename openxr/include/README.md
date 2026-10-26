# OpenXR headers

`openxr/openxr.h`, `openxr/openxr_platform.h` and `openxr/openxr_platform_defines.h` are copied
unchanged from the Khronos Group's [OpenXR-SDK](https://github.com/KhronosGroup/OpenXR-SDK), tag
`release-1.1.65` (OpenXR 1.1.65), folder `include/openxr/`. They are © The Khronos Group Inc., under
Apache-2.0 OR MIT, as the header at the top of each file says.

Only the headers are here: the loader (`libopenxr_loader.so`) is opened when the program runs, and
ships inside the application, not in this repository.
