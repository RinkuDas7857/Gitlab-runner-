package api

// ConfigExecOutput defines the output structure of the config_exec call.
//
// This should be used to pass the configuration values from Custom Executor
// driver to the Runner.
type ConfigExecOutput struct {
	Driver *DriverInfo `json:"driver,omitempty"`

	Hostname  *string `json:"hostname,omitempty"`
	BuildsDir *string `json:"builds_dir,omitempty"`
	CacheDir  *string `json:"cache_dir,omitempty"`

	BuildsDirIsShared *bool `json:"builds_dir_is_shared,omitempty"`

	JobEnv *map[string]string `json:"job_env,omitempty"`

	// DEPRECATED
	// TODO: Remove this in 15.0 - https://gitlab.com/gitlab-org/gitlab-runner/-/issues/27959
	StepScriptSupported *bool `json:"step_script_supported,omitempty"`
}

// DriverInfo wraps the information about Custom Executor driver details
// like the name or version
type DriverInfo struct {
	Name    *string `json:"name,omitempty"`
	Version *string `json:"version,omitempty"`
}
