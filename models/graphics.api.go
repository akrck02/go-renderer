package models

type GraphicsApi interface {
	Start(app *Application)
	Draw(app *Application)
}
