package software

import (
	"errors"

	"github.com/akrck02/go-renderer/models"
)

type Software struct{}

func (software *Software) StartLoop(app *models.Application) error {
	return errors.New("Graphics API not implemented yet!")
}

func (software *Software) Draw(app *models.Application) error {
	return errors.New("Graphics API not implemented yet!")
}
