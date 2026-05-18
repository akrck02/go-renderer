package models

type ApplicationType int

const (
	HeadlessApplication ApplicationType = 1
	WindowApplication   ApplicationType = 2
)

type Application struct {
	Type     ApplicationType
	Width    int
	Height   int
	Title    string
	Version  string
	Renderer Renderer
	Draw     func(*Application) error
}
