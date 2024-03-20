package docker

import (
	"errors"
	"fmt"

	"github.com/magefile/mage/sh"
	"gopkg.in/yaml.v2"
)

type ManifestToolContext struct {
}

func NewManifestTool() *ManifestToolContext {
	return &ManifestToolContext{}
}

type PlatformSpec struct {
	Os           string
	Architecture string
	Variant      string `yaml:",omitempty"`
}

func (p PlatformSpec) String() string {
	if p.Variant != "" {
		return fmt.Sprintf("%s/%s/%s", p.Os, p.Architecture, p.Variant)
	} else {
		return fmt.Sprintf("%s/%s", p.Os, p.Architecture)
	}
}

func (mt *ManifestToolContext) ManifestTool(args ...string) error {
	return sh.RunWithV(
		map[string]string{},
		"manifest-tool",
		args...,
	)
}

func (mt *ManifestToolContext) Push(args ...string) error {
	return mt.ManifestTool(append([]string{"push"}, args...)...)
}

type ManifestImage struct {
	Image    string
	Platform PlatformSpec
}

type ManifestToolSpec struct {
	Image     string
	Manifests []ManifestImage
}

func (mts *ManifestToolSpec) Render() (string, error) {
	if mts.Image == "" {
		return "", errors.New("no image name provided for manifest list")
	}
	if mts.Manifests == nil || len(mts.Manifests) == 0 {
		return "", errors.New("no component images provided for manifest list")
	}
	src, err := yaml.Marshal(mts)
	if err != nil {
		return "", err
	}
	return string(src), nil
}
