package ci

import (
	"fmt"

	"gitlab.com/gitlab-org/gitlab-runner/magefiles/build"
	"gitlab.com/gitlab-org/gitlab-runner/magefiles/env"
)

var (
	RegistryImage    = env.NewDefault("CI_REGISTRY_IMAGE", fmt.Sprintf("gitlab/%s", build.AppName))
	Registry         = env.New("CI_REGISTRY")
	RegistryUser     = env.New("CI_REGISTRY_USER")
	RegistryPassword = env.New("CI_REGISTRY_PASSWORD")

	RegistryAuthBundle = env.Variables{
		Registry,
		RegistryUser,
		RegistryPassword,
	}
)
