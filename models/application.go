package models

type Application struct {
	Width    int
	Height   int
	Title    string
	Version  string
	Renderer Renderer
	Draw     func(*Application)
}
