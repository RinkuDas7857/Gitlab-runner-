//go:build !integration

package commands

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

type metricsServerTestExample struct {
	address         string
	setAddress      bool
	expectedAddress string
	errorIsExpected bool
}

type metricsServerConfigurationType string

const (
	configurationFromCli    metricsServerConfigurationType = "from-cli"
	configurationFromConfig metricsServerConfigurationType = "from-config"
)

func testListenAddressSetting(
	t *testing.T,
	exampleName string,
	example metricsServerTestExample,
	testType metricsServerConfigurationType,
) {
	t.Run(fmt.Sprintf("%s-%s", exampleName, testType), func(t *testing.T) {
		cfg := configOptionsWithListenAddress{}
		cfg.config = &common.Config{}
		if example.setAddress {
			if testType == configurationFromCli {
				cfg.ListenAddress = example.address
			} else {
				cfg.config.ListenAddress = example.address
			}
		}

		address, err := cfg.listenAddress()
		assert.Equal(t, example.expectedAddress, address)
		if example.errorIsExpected {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	})
}

func TestMetricsServer(t *testing.T) {
	examples := map[string]metricsServerTestExample{
		"address-set-without-port": {"localhost", true, "localhost:9252", false},
		"port-set-without-address": {":1234", true, ":1234", false},
		"address-set-with-port":    {"localhost:1234", true, "localhost:1234", false},
		"address-is-empty":         {"", true, "", false},
		"address-is-invalid":       {"localhost::1234", true, "", true},
		"address-not-set":          {"", false, "", false},
	}

	for exampleName, example := range examples {
		testListenAddressSetting(t, exampleName, example, configurationFromCli)
		testListenAddressSetting(t, exampleName, example, configurationFromConfig)
	}
}

func TestGetConfig(t *testing.T) {
	c := &configOptions{}
	assert.Nil(t, c.getConfig())

	c.config = &common.Config{}
	assert.True(t, c.config != c.getConfig())
}

func TestRunnerByName(t *testing.T) {
	examples := map[string]struct {
		runners       []*common.RunnerConfig
		runnerName    string
		expectedIndex int
		expectedError error
	}{
		"finds runner by name": {
			runners: []*common.RunnerConfig{
				{
					Name: "runner1",
				},
				{
					Name: "runner2",
				},
			},
			runnerName:    "runner2",
			expectedIndex: 1,
		},
		"does not find non-existent runner": {
			runners: []*common.RunnerConfig{
				{
					Name: "runner1",
				},
				{
					Name: "runner2",
				},
			},
			runnerName:    "runner3",
			expectedIndex: -1,
			expectedError: fmt.Errorf("could not find a runner with the name 'runner3'"),
		},
	}

	for tn, tt := range examples {
		t.Run(tn, func(t *testing.T) {
			config := configOptions{
				config: &common.Config{
					Runners: tt.runners,
				},
			}

			runner, err := config.RunnerByName(tt.runnerName)
			if tt.expectedIndex == -1 {
				assert.Nil(t, runner)
			} else {
				assert.Equal(t, tt.runners[tt.expectedIndex], runner)
			}
			assert.Equal(t, tt.expectedError, err)
		})
	}
}

func TestRunnerByURLAndID(t *testing.T) {
	examples := map[string]struct {
		runners       []*common.RunnerConfig
		runnerURL     string
		runnerID      int64
		expectedIndex int
		expectedError error
	}{
		"finds runner by name": {
			runners: []*common.RunnerConfig{
				{
					RunnerCredentials: common.RunnerCredentials{
						ID:  1,
						URL: "https://gitlab1.example.com/",
					},
				},
				{
					RunnerCredentials: common.RunnerCredentials{
						ID:  2,
						URL: "https://gitlab1.example.com/",
					},
				},
			},
			runnerURL:     "https://gitlab1.example.com/",
			runnerID:      1,
			expectedIndex: 0,
		},
		"does not find runner with wrong ID": {
			runners: []*common.RunnerConfig{
				{
					RunnerCredentials: common.RunnerCredentials{
						ID:  1,
						URL: "https://gitlab1.example.com/",
					},
				},
				{
					RunnerCredentials: common.RunnerCredentials{
						ID:  2,
						URL: "https://gitlab1.example.com/",
					},
				},
			},
			runnerURL:     "https://gitlab1.example.com/",
			runnerID:      3,
			expectedIndex: -1,
			expectedError: fmt.Errorf(`could not find a runner with the URL "https://gitlab1.example.com/" and ID 3`),
		},
		"does not find runner with wrong URL": {
			runners: []*common.RunnerConfig{
				{
					RunnerCredentials: common.RunnerCredentials{
						ID:  1,
						URL: "https://gitlab1.example.com/",
					},
				},
				{
					RunnerCredentials: common.RunnerCredentials{
						ID:  2,
						URL: "https://gitlab1.example.com/",
					},
				},
			},
			runnerURL:     "https://gitlab2.example.com/",
			runnerID:      1,
			expectedIndex: -1,
			expectedError: fmt.Errorf(`could not find a runner with the URL "https://gitlab2.example.com/" and ID 1`),
		},
	}

	for tn, tt := range examples {
		t.Run(tn, func(t *testing.T) {
			config := configOptions{
				config: &common.Config{
					Runners: tt.runners,
				},
			}

			runner, err := config.RunnerByURLAndID(tt.runnerURL, tt.runnerID)
			if tt.expectedIndex == -1 {
				assert.Nil(t, runner)
			} else {
				assert.Equal(t, tt.runners[tt.expectedIndex], runner)
			}
			assert.Equal(t, tt.expectedError, err)
		})
	}
}

func Test_loadConfig(t *testing.T) {
	testCases := map[string]struct {
		config      string
		localConfig string
		assertFn    func(
			t *testing.T,
			err error,
			config *common.Config,
			localConfig *common.LocalConfig,
			localConfigFile *os.File,
		)
	}{
		"generates and saves missing unique system IDs": {
			config:      "",
			localConfig: "",
			assertFn: func(
				t *testing.T,
				err error,
				config *common.Config,
				localConfig *common.LocalConfig,
				localConfigFile *os.File,
			) {
				assert.NoError(t, err)
				assert.NotEmpty(t, localConfig.SystemID)
				content, err := os.ReadFile(localConfigFile.Name())
				require.NoError(t, err)
				assert.Contains(t, fmt.Sprintf(`system_id = "%s"`, localConfig.SystemID), content)
			},
		},
		"preserves existing unique system IDs": {
			config: "",
			localConfig: `
			system_id = "some_system_id"
`,
			assertFn: func(
				t *testing.T,
				err error,
				config *common.Config,
				localConfig *common.LocalConfig,
				localConfigFile *os.File,
			) {
				assert.NoError(t, err)
				assert.Equal(t, "some_system_id", localConfig.SystemID)
			},
		},
	}

	configFile, err := os.CreateTemp(os.TempDir(), "config.toml")
	require.NoError(t, err)
	defer func() { _ = configFile.Close() }()
	localConfigFile, err := os.CreateTemp(os.TempDir(), "config.local.toml")
	require.NoError(t, err)
	defer func() { _ = localConfigFile.Close() }()

	defer func() {
		_ = os.Remove(configFile.Name())
		_ = os.Remove(localConfigFile.Name())
	}()

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			require.NoError(t, configFile.Truncate(0))
			require.NoError(t, localConfigFile.Truncate(0))
			_, err := configFile.WriteString(tc.config)
			require.NoError(t, err)
			_, err = localConfigFile.WriteString(tc.localConfig)
			require.NoError(t, err)

			c := configOptions{ConfigFile: configFile.Name()}
			err = c.loadConfig()
			tc.assertFn(t, err, c.config, c.localConfig, localConfigFile)
		})
	}
}
