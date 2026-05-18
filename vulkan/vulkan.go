package vulkan

import (
	"errors"

	"github.com/akrck02/go-renderer/models"
)

type Vulkan struct{}

func (vulkan *Vulkan) StartLoop(app *models.Application) error {
	return errors.New("Graphics API not implemented yet!")
}

func (vulkan *Vulkan) Draw(app *models.Application) error {
	return errors.New("Graphics API not implemented yet!")
}
