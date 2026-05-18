package webgpu

import (
	"errors"

	"github.com/akrck02/go-renderer/models"
)

type WebGPU struct{}

func (webgpu *WebGPU) StartLoop(app *models.Application) error {
	return errors.New("Graphics API not implemented yet!")
}

func (webgpu *WebGPU) Draw(app *models.Application) error {
	return errors.New("Graphics API not implemented yet!")
}
