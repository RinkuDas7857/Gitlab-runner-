package volumes

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"

	docker_helpers "gitlab.com/gitlab-org/gitlab-runner/helpers/docker"
)

type containerClient interface {
	docker_helpers.Client

	LabelContainer(container *container.Config, containerType string, otherLabels ...string)
	WaitForContainer(id string) error
	RemoveContainer(ctx context.Context, id string) error
}

type CacheContainersManager interface {
	FindOrCleanExisting(containerName string, containerPath string) string
	Create(containerName string, containerPath string) (string, error)
	Cleanup(ctx context.Context, ids []string) chan bool
}

type cacheContainerManager struct {
	ctx    context.Context
	logger logrus.FieldLogger

	containerClient containerClient

	helperImage        *types.ImageInspect
	failedContainerIDs []string
}

func NewCacheContainerManager(ctx context.Context, logger logrus.FieldLogger, cClient containerClient, helperImage *types.ImageInspect) CacheContainersManager {
	return &cacheContainerManager{
		ctx:             ctx,
		logger:          logger,
		containerClient: cClient,
		helperImage:     helperImage,
	}
}

func (m *cacheContainerManager) FindOrCleanExisting(containerName string, containerPath string) string {
	inspected, err := m.containerClient.ContainerInspect(m.ctx, containerName)
	logger := m.logger.WithField("ContainerName", containerName)
	if err != nil {
		logger.Debugln(fmt.Sprintf("Error while inspecting container: %v", err))
		return ""
	}

	// check if we have valid cache, if not remove the broken container
	_, ok := inspected.Config.Volumes[containerPath]
	if !ok {
		logger.Debugln(fmt.Sprintf("Removing broken cache container for %q path", containerPath))
		err = m.containerClient.RemoveContainer(m.ctx, inspected.ID)
		logger.Debugln(fmt.Sprintf("Cache container for %q path removed with %v", containerPath, err))

		return ""
	}

	return inspected.ID
}

func (m *cacheContainerManager) Create(containerName string, containerPath string) (string, error) {
	containerID, err := m.createCacheContainer(containerName, containerPath)
	if err != nil {
		return "", err
	}

	err = m.startCacheContainer(containerID)
	if err != nil {
		return "", err
	}

	return containerID, nil
}

func (m *cacheContainerManager) createCacheContainer(containerName string, containerPath string) (string, error) {
	config := &container.Config{
		Image: m.helperImage.ID,
		Cmd:   []string{"gitlab-runner-helper", "cache-init", containerPath},
		Volumes: map[string]struct{}{
			containerPath: {},
		},
	}
	m.containerClient.LabelContainer(config, "cache", "cache.dir="+containerPath)

	hostConfig := &container.HostConfig{
		LogConfig: container.LogConfig{
			Type: "json-file",
		},
	}

	resp, err := m.containerClient.ContainerCreate(m.ctx, config, hostConfig, nil, containerName)
	if err != nil {
		if resp.ID != "" {
			m.failedContainerIDs = append(m.failedContainerIDs, resp.ID)
		}

		return "", err
	}

	return resp.ID, nil
}

func (m *cacheContainerManager) startCacheContainer(containerID string) error {
	logger := m.logger.WithField("ContainerID", containerID)
	logger.Debugln("Starting cache container...")
	err := m.containerClient.ContainerStart(m.ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		m.failedContainerIDs = append(m.failedContainerIDs, containerID)

		return err
	}

	logger.Debugln(fmt.Sprintf("Waiting for cache container..."))
	err = m.containerClient.WaitForContainer(containerID)
	if err != nil {
		m.failedContainerIDs = append(m.failedContainerIDs, containerID)

		return err
	}

	return nil
}

func (m *cacheContainerManager) Cleanup(ctx context.Context, ids []string) chan bool {
	done := make(chan bool, 1)

	ids = append(m.failedContainerIDs, ids...)

	go func() {
		wg := new(sync.WaitGroup)
		wg.Add(len(ids))
		for _, id := range ids {
			m.remove(ctx, wg, id)
		}

		wg.Wait()
		done <- true
	}()

	return done
}

func (m *cacheContainerManager) remove(ctx context.Context, wg *sync.WaitGroup, id string) {
	go func() {
		err := m.containerClient.RemoveContainer(ctx, id)
		if err != nil {
			m.logger.WithField("ContainerID", id).Debugln(
				fmt.Sprintf("Error while removing the container: %v", err))
		}
		wg.Done()
	}()
}
