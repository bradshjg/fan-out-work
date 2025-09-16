package services

import (
	"os"
)

type FanoutService interface {
	GetPatches() ([]string, error)
}

func NewFanoutService() *FanoutServiceImpl {
	return &FanoutServiceImpl{}
}

type FanoutServiceImpl struct{}

func (*FanoutServiceImpl) GetPatches() ([]string, error) {
	var patches []string
	entries, err := os.ReadDir("./patches")
	if err != nil {
		return []string{}, err
	}
	for _, e := range entries {
		if e.IsDir() {
			patches = append(patches, e.Name())
		}
	}
	return patches, nil
}
