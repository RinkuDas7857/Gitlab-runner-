package docker

import (
	"crypto/md5"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

type ContainerManager interface {
	LabelContainer(container *container.Config, containerType string, otherLabels ...string)
	CreateContainer(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.ContainerCreateCreatedBody, error)
	StartContainer(containerID string, options types.ContainerStartOptions) error
	InspectContainer(containerName string) (types.ContainerJSON, error)
	WaitForContainer(id string) error
	RemoveContainer(id string) error
}

type HelperImageResolver interface {
	ResolveHelperImage() (*types.ImageInspect, error)
}

type VolumesManager interface {
	CreateUserVolumes(volumes []string) error
	CreateBuildVolume(volumes []string) error
	VolumeBindings() []string
	CacheContainerIDs() []string
	TmpContainerIDs() []string
}

type DefaultVolumesManagerConfig struct {
	cacheDir                string
	jobsRootDir             string
	fullProjectDir          string
	projectUniqName         string
	gitStrategy             common.GitStrategy
	disableCache            bool
	outdatedHelperImageUsed bool
}

type DefaultVolumesManager struct {
	DefaultVolumesManagerConfig

	logger common.BuildLogger

	containerManager    ContainerManager
	helperImageResolver HelperImageResolver

	volumeBindings    []string
	cacheContainerIDs []string
	tmpContainerIDs   []string
}

func NewDefaultVolumesManager(logger common.BuildLogger, cManager ContainerManager, hiResolver HelperImageResolver, config DefaultVolumesManagerConfig) VolumesManager {
	return &DefaultVolumesManager{
		DefaultVolumesManagerConfig: config,
		logger:              logger,
		containerManager:    cManager,
		helperImageResolver: hiResolver,
	}
}

func (m *DefaultVolumesManager) CreateUserVolumes(volumes []string) error {
	for _, volume := range volumes {
		err := m.addVolume(volume)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *DefaultVolumesManager) addVolume(volume string) error {
	hostVolume := strings.SplitN(volume, ":", 2)

	var err error
	switch len(hostVolume) {
	case 2:
		err = m.addHostVolume(hostVolume[0], hostVolume[1])
	case 1:
		// disable cache disables
		err = m.addCacheVolume(hostVolume[0])
	}

	if err != nil {
		m.logger.Errorln("Failed to create container volume for", volume, err)
	}

	return err
}

func (m *DefaultVolumesManager) addHostVolume(hostPath string, containerPath string) error {
	containerPath = m.getAbsoluteContainerPath(containerPath)
	m.appendVolumeBind(hostPath, containerPath)

	return nil
}

func (m *DefaultVolumesManager) getAbsoluteContainerPath(dir string) string {
	if path.IsAbs(dir) {
		return dir
	}

	return path.Join(m.fullProjectDir, dir)
}

func (m *DefaultVolumesManager) appendVolumeBind(hostPath string, containerPath string) {
	m.logger.Debugln(fmt.Sprintf("Using host-based %q for %q...", hostPath, containerPath))

	bindDefinition := fmt.Sprintf("%v:%v", filepath.ToSlash(hostPath), containerPath)
	m.volumeBindings = append(m.volumeBindings, bindDefinition)
}

func (m *DefaultVolumesManager) addCacheVolume(containerPath string) error {
	containerPath = m.getAbsoluteContainerPath(containerPath)

	// disable cache for automatic container cache,
	// but leave it for host volumes (they are shared on purpose)
	if m.disableCache {
		m.logger.Debugln(fmt.Sprintf("Container cache for %q is disabled", containerPath))

		return nil
	}

	hash := md5.Sum([]byte(containerPath))
	if m.cacheDir != "" {
		return m.createHostBasedCacheVolume(containerPath, hash)
	}

	return m.createContainerBasedCacheVolume(containerPath, hash)
}

func (m *DefaultVolumesManager) createHostBasedCacheVolume(containerPath string, hash [md5.Size]byte) error {
	hostPath := fmt.Sprintf("%s/%s/%x", m.cacheDir, m.projectUniqName, hash)
	hostPath, err := filepath.Abs(hostPath)
	if err != nil {
		return err
	}

	m.appendVolumeBind(hostPath, containerPath)

	return nil
}

func (m *DefaultVolumesManager) createContainerBasedCacheVolume(containerPath string, hash [md5.Size]byte) error {
	containerName := fmt.Sprintf("%s-cache-%x", m.projectUniqName, hash)

	containerID := m.findExistingCacheContainer(containerName, containerPath)

	// create new cache container for that project
	if containerID == "" {
		var err error

		containerID, err = m.createCacheVolume(containerName, containerPath)
		if err != nil {
			return err
		}
	}

	m.logger.Debugln(fmt.Sprintf("Using container %q as cache %q...", containerID, containerPath))
	m.cacheContainerIDs = append(m.cacheContainerIDs, containerID)

	return nil
}

func (m *DefaultVolumesManager) findExistingCacheContainer(containerName string, containerPath string) string {
	inspected, err := m.containerManager.InspectContainer(containerName)
	if err != nil {
		return ""
	}

	// check if we have valid cache,if not remove the broken container
	_, ok := inspected.Config.Volumes[containerPath]
	if !ok {
		m.logger.Debugln(fmt.Sprintf("Removing broken cache container for %q path", containerPath))
		err = m.containerManager.RemoveContainer(inspected.ID)
		m.logger.Debugln(fmt.Sprintf("Cache container for %q path removed with %v", containerPath, err))

		return ""
	}

	return inspected.ID
}

func (m *DefaultVolumesManager) createCacheVolume(containerName string, containerPath string) (string, error) {
	cacheImage, err := m.helperImageResolver.ResolveHelperImage()
	if err != nil {
		return "", err
	}

	config := &container.Config{
		Image: cacheImage.ID,
		Cmd:   m.getCacheCommand(containerPath),
		Volumes: map[string]struct{}{
			containerPath: {},
		},
	}
	m.containerManager.LabelContainer(config, "cache", "cache.dir="+containerPath)

	hostConfig := &container.HostConfig{
		LogConfig: container.LogConfig{
			Type: "json-file",
		},
	}

	resp, err := m.containerManager.CreateContainer(config, hostConfig, nil, containerName)
	if err != nil {
		if resp.ID != "" {
			m.tmpContainerIDs = append(m.tmpContainerIDs, resp.ID)
		}

		return "", err
	}

	m.logger.Debugln(fmt.Sprintf("Starting cache container %q...", resp.ID))
	err = m.containerManager.StartContainer(resp.ID, types.ContainerStartOptions{})
	if err != nil {
		m.tmpContainerIDs = append(m.tmpContainerIDs, resp.ID)

		return "", err
	}

	m.logger.Debugln(fmt.Sprintf("Waiting for cache container %q...", resp.ID))
	err = m.containerManager.WaitForContainer(resp.ID)
	if err != nil {
		m.tmpContainerIDs = append(m.tmpContainerIDs, resp.ID)

		return "", err
	}

	return resp.ID, nil
}

func (m *DefaultVolumesManager) getCacheCommand(containerPath string) []string {
	// TODO: Remove in 12.0 to start using the command from `gitlab-runner-helper`
	if m.outdatedHelperImageUsed {
		m.logger.Debugln("Falling back to old gitlab-runner-cache command")
		return []string{"gitlab-runner-cache", containerPath}
	}

	return []string{"gitlab-runner-helper", "cache-init", containerPath}
}

func (m *DefaultVolumesManager) CreateBuildVolume(volumes []string) error {
	// Cache Git sources:
	// use a `jobsRootDir`
	if !path.IsAbs(m.jobsRootDir) && m.jobsRootDir != "/" {
		return errors.New("build directory needs to be absolute and non-root path")
	}

	if IsHostMountedVolume(m.jobsRootDir, volumes...) {
		// If builds directory is within a volume mounted manually by user
		// it will be added by CreateUserVolumes(), so nothing more to do
		// here
		return nil
	}

	if m.gitStrategy == common.GitFetch && !m.disableCache {
		// create persistent cache container
		return m.addVolume(m.jobsRootDir)
	}

	// create temporary cache container
	id, err := m.createCacheVolume("", m.jobsRootDir)
	if err != nil {
		return err
	}

	m.cacheContainerIDs = append(m.cacheContainerIDs, id)
	m.tmpContainerIDs = append(m.tmpContainerIDs, id)

	return nil
}

func (m *DefaultVolumesManager) VolumeBindings() []string {
	return m.volumeBindings
}

func (m *DefaultVolumesManager) CacheContainerIDs() []string {
	return m.cacheContainerIDs
}

func (m *DefaultVolumesManager) TmpContainerIDs() []string {
	return m.tmpContainerIDs
}

func IsHostMountedVolume(dir string, volumes ...string) bool {
	for _, volume := range volumes {
		hostVolume := strings.Split(volume, ":")

		if len(hostVolume) < 2 {
			continue
		}

		if isParentOf(path.Clean(hostVolume[1]), path.Clean(dir)) {
			return true
		}
	}
	return false
}

func isParentOf(parent string, dir string) bool {
	for dir != "/" && dir != "." {
		if dir == parent {
			return true
		}
		dir = path.Dir(dir)
	}
	return false
}
