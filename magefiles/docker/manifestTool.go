package docker

import (
	"github.com/magefile/mage/sh"
)
type ManifestToolContext struct {
}

func NewManifestTool() *ManifestToolContext {
	return &ManifestToolContext{
	}
}

func (mt *ManifestToolContext) ManifestTool(args ...string) error {
	return sh.RunWithV(
		map[string]string{
		},
		"manifest-tool",
		args...,
	)
}

func (mt *ManifestToolContext) Push(args ...string) error {
	return mt.ManifestTool(append([]string{"push"}, args...)...)
}

