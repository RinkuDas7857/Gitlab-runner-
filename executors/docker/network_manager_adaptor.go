package docker

import (
	"gitlab.com/gitlab-org/gitlab-runner/executors/docker/internal/networks"
)

var createNetworksManager = func(e *executor) (networks.Manager, error) {
	networksManager := networks.NewManager(e.Build.Log(), e.client, e.Build)

	return networksManager, nil
}

func (e *executor) createNetworksManager() error {
	nm, err := createNetworksManager(e)
	if err != nil {
		return err
	}
	e.networksManager = nm

	return nil
}
