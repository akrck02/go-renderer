package models

import "github.com/akrck02/go-renderer/graphics"

type ApplicationType int

const (
	HeadlessApplication ApplicationType = 1
	WindowApplication   ApplicationType = 2
)

type Application struct {
	Type       ApplicationType
	Background graphics.Vec4
	Width      int
	Height     int
	Title      string
	Version    string
	Renderer   Renderer
	Draw       func(*Application) error
}
