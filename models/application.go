package models

import "github.com/akrck02/go-renderer/graphics"

type ApplicationType int

const (
	HeadlessApplication ApplicationType = 1
	WindowApplication   ApplicationType = 2
)

type Camera struct {
	Position graphics.Vec4
	Target   graphics.Vec4
	Up       graphics.Vec4
}

type Application struct {
	Type       ApplicationType
	Background graphics.Vec4
	Width      int
	Height     int
	Title      string
	Version    string
	Renderer   Renderer
	Draw       func(*Application) error
	Camera     Camera
	Rotation   graphics.Vec4
}
