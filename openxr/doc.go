// Package openxr shows a scene in a headset through OpenXR, the Khronos standard that headsets'
// runtimes speak (a Meta Quest's among them).
//
// On Android it opens the runtime's loader, libopenxr_loader.so, when the program runs (there is no
// link-time dependency), and makes a session for the OpenGL ES context that android.RunHeadset
// holds. Every frame it says where each eye is and hands out a framebuffer for each, with a depth
// buffer; the program draws the eyes and gives the frame back. The controllers come back as poses,
// triggers, sticks and buttons, and Controls.Apply turns them into an input.State.
//
// What the runtime asks, its session's states and the controllers' mapping to input are plain Go
// and tested anywhere; talking to the runtime is built only for Android (files tagged android).
package openxr
