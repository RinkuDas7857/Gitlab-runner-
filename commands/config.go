package commands

import (
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/network"
)

func GetDefaultConfigFile() string {
	return filepath.Join(getDefaultConfigDirectory(), "config.toml")
}

func getDefaultCertificateDirectory() string {
	return filepath.Join(getDefaultConfigDirectory(), "certs")
}

type configOptions struct {
	configMutex sync.Mutex
	config      *common.Config
	localConfig *common.LocalConfig

	ConfigFile string `short:"c" long:"config" env:"CONFIG_FILE" description:"Config file"`
}

// getConfig returns a copy of the config as it was during the function call.
// This makes sure the properties of the config won't change after the mutex
// in this function has been unlocked. We don't change the inner objects,
// which makes a shallow copy OK in this case, if that ever changes we should
// look into deep copying or other alternatives.
// All writes of the config property should be protected by a mutex while
// all reads should use this function.
func (c *configOptions) getConfig() *common.Config {
	c.configMutex.Lock()
	defer c.configMutex.Unlock()

	if c.config == nil {
		return nil
	}

	config := *c.config
	return &config
}

func (c *configOptions) saveConfig() error {
	return c.config.SaveConfig(c.ConfigFile)
}

func (c *configOptions) loadConfig() error {
	c.configMutex.Lock()
	defer c.configMutex.Unlock()

	config := common.NewConfig()
	err := config.LoadConfig(c.ConfigFile)
	if err != nil {
		return err
	}

	localConfig, err := c.loadLocalConfig(c.ConfigFile)
	if err != nil {
		return fmt.Errorf("loading local config file: %w", err)
	}

	for _, runner := range config.Runners {
		runner.SystemID = localConfig.SystemID
	}

	c.config = config
	c.localConfig = localConfig
	return nil
}

func (c *configOptions) loadLocalConfig(configFile string) (*common.LocalConfig, error) {
	ext := path.Ext(configFile)
	localConfigPath := path.Join(
		path.Dir(configFile),
		strings.TrimSuffix(path.Base(configFile), ext)+".local"+ext,
	)

	localConfig := common.NewLocalConfig()
	err := localConfig.LoadConfig(localConfigPath)
	if err != nil {
		return nil, err
	}

	// ensure we have a system ID
	if localConfig.SystemID == "" {
		err = localConfig.EnsureSystemID()
		if err != nil {
			return nil, err
		}

		err = localConfig.SaveConfig(localConfigPath)
		if err != nil {
			return nil, fmt.Errorf("saving local config file: %w", err)
		}
	}

	return localConfig, nil
}

func (c *configOptions) RunnerByName(name string) (*common.RunnerConfig, error) {
	config := c.getConfig()
	if config == nil {
		return nil, fmt.Errorf("config has not been loaded")
	}

	for _, runner := range config.Runners {
		if runner.Name == name {
			return runner, nil
		}
	}

	return nil, fmt.Errorf("could not find a runner with the name '%s'", name)
}

func (c *configOptions) RunnerByURLAndID(url string, id int64) (*common.RunnerConfig, error) {
	if c.config == nil {
		return nil, fmt.Errorf("config has not been loaded")
	}

	for _, runner := range c.config.Runners {
		if runner.URL == url && runner.ID == id {
			return runner, nil
		}
	}

	return nil, fmt.Errorf("could not find a runner with the URL %q and ID %d", url, id)
}

//nolint:lll
type configOptionsWithListenAddress struct {
	configOptions

	ListenAddress string `long:"listen-address" env:"LISTEN_ADDRESS" description:"Metrics / pprof server listening address"`
}

func (c *configOptionsWithListenAddress) listenAddress() (string, error) {
	address := c.getConfig().ListenAddress
	if c.ListenAddress != "" {
		address = c.ListenAddress
	}

	if address == "" {
		return "", nil
	}

	_, port, err := net.SplitHostPort(address)
	if err != nil && !strings.Contains(err.Error(), "missing port in address") {
		return "", err
	}

	if port == "" {
		return fmt.Sprintf("%s:%d", address, common.DefaultMetricsServerPort), nil
	}
	return address, nil
}

func init() {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		err := os.Setenv("CONFIG_FILE", GetDefaultConfigFile())
		if err != nil {
			logrus.WithError(err).Fatal("Couldn't set CONFIG_FILE environment variable")
		}
	}

	network.CertificateDirectory = getDefaultCertificateDirectory()
}
