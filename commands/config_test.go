//go:build !integration

package commands

import (
	"fmt"
	"os"
	"path/filepath"
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
		runnerSystemID string
		assertFn       func(
			t *testing.T,
			err error,
			config *common.Config,
			systemIDState *common.SystemIDState,
			systemIDStateFile string,
		)
	}{
		"generates and saves missing system IDs": {
			runnerSystemID: "",
			assertFn: func(
				t *testing.T,
				err error,
				_ *common.Config,
				systemIDState *common.SystemIDState,
				systemIDFile string,
			) {
				assert.NoError(t, err)
				assert.NotEmpty(t, systemIDState.GetSystemID())
				content, err := os.ReadFile(systemIDFile)
				require.NoError(t, err)
				assert.Contains(t, string(content), systemIDState.GetSystemID())
			},
		},
		"preserves existing unique system IDs": {
			runnerSystemID: "s_c2d22f638c25",
			assertFn: func(
				t *testing.T,
				err error,
				_ *common.Config,
				systemIDState *common.SystemIDState,
				_ string,
			) {
				assert.NoError(t, err)
				assert.Equal(t, "s_c2d22f638c25", systemIDState.GetSystemID())
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			dir := t.TempDir()
			cfgName := filepath.Join(dir, "config.toml")
			systemIDFile := filepath.Join(dir, ".runner_system_id")

			require.NoError(t, os.WriteFile(cfgName, []byte("[[runners]]\n name = \"runner\""), 0777))
			require.NoError(t, os.WriteFile(systemIDFile, []byte(tc.runnerSystemID), 0777))

			c := configOptions{ConfigFile: cfgName}
			err := c.loadConfig()

			require.Equal(t, 1, len(c.config.Runners))
			tc.assertFn(t, err, c.config, c.config.Runners[0].SystemStateID, systemIDFile)
		})
	}
}
