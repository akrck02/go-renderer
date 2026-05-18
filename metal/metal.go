package metal

import (
	"errors"

	"github.com/akrck02/go-renderer/models"
)

type Metal struct{}

func (metal *Metal) StartLoop(app *models.Application) error {
	return errors.New("Graphics API not implemented yet!")
}

func (metal *Metal) Draw(app *models.Application) error {
	return errors.New("Graphics API not implemented yet!")
}
