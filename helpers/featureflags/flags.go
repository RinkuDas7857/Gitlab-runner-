package featureflags

import (
	"strconv"
)

const (
	CmdDisableDelayedErrorLevelExpansion string = "FF_CMD_DISABLE_DELAYED_ERROR_LEVEL_EXPANSION"
	NetworkPerBuild                      string = "FF_NETWORK_PER_BUILD"
	UseLegacyKubernetesExecutionStrategy string = "FF_USE_LEGACY_KUBERNETES_EXECUTION_STRATEGY"
	UseDirectDownload                    string = "FF_USE_DIRECT_DOWNLOAD"
	SkipNoOpBuildStages                  string = "FF_SKIP_NOOP_BUILD_STAGES"
	ShellExecutorUseLegacyProcessKill    string = "FF_SHELL_EXECUTOR_USE_LEGACY_PROCESS_KILL"
	ResetHelperImageEntrypoint           string = "FF_RESET_HELPER_IMAGE_ENTRYPOINT"
	UseLegacyDockerUmask                 string = "FF_USE_LEGACY_DOCKER_UMASK"
)

type FeatureFlag struct {
	Name            string
	DefaultValue    string
	Deprecated      bool
	ToBeRemovedWith string
	Description     string
}

// REMEMBER to update the documentation after adding or removing a feature flag
//
// Please use `make update_feature_flags_docs` to make the update automatic and
// properly formatted. It will replace the existing table with the new one, computed
// basing on the values below
var flags = []FeatureFlag{
	{
		Name:            CmdDisableDelayedErrorLevelExpansion,
		DefaultValue:    "false",
		Deprecated:      false,
		ToBeRemovedWith: "",
		Description: "Disables [EnableDelayedExpansion](https://ss64.com/nt/delayedexpansion.html) for " +
			"error checking for when using [Window Batch](../shells/index.md#windows-batch) shell",
	},
	{
		Name:            NetworkPerBuild,
		DefaultValue:    "false",
		Deprecated:      false,
		ToBeRemovedWith: "",
		Description: "Enables creation of a Docker [network per build](../executors/docker.md#networking) with " +
			"the `docker` executor",
	},
	{
		Name:            UseLegacyKubernetesExecutionStrategy,
		DefaultValue:    "true",
		Deprecated:      false,
		ToBeRemovedWith: "",
		Description: "When set to `false` disables execution of remote Kubernetes commands through `exec` in " +
			"favor of `attach` to solve problems like " +
			"[#4119](https://gitlab.com/gitlab-org/gitlab-runner/-/issues/4119)",
	},
	{
		Name:            UseDirectDownload,
		DefaultValue:    "true",
		Deprecated:      false,
		ToBeRemovedWith: "",
		Description: "When set to `true` Runner tries to direct-download all artifacts instead of proxying " +
			"through GitLab on a first try. Enabling might result in a download failures due to problem validating " +
			"TLS certificate of Object Storage if it is enabled by GitLab",
	},
	{
		Name:            SkipNoOpBuildStages,
		DefaultValue:    "true",
		Deprecated:      false,
		ToBeRemovedWith: "",
		Description:     "When set to `false` all build stages are executed even if running them has no effect",
	},
	{
		Name:            ShellExecutorUseLegacyProcessKill,
		DefaultValue:    "false",
		Deprecated:      true,
		ToBeRemovedWith: "14.0",
		Description: "Use the old process termination that was used prior to GitLab 13.1 where only `SIGKILL`" +
			" was sent",
	},
	{
		Name:            ResetHelperImageEntrypoint,
		DefaultValue:    "true",
		Deprecated:      true,
		ToBeRemovedWith: "14.0",
		Description: "Enables adding an ENTRYPOINT layer for Helper images imported from local Docker archives " +
			"by the `docker` executor, in order to enable [importing of user certificate roots]" +
			"(./tls-self-signed.md#trusting-the-certificate-for-the-other-cicd-stages)",
	},
	{
		Name:            UseLegacyDockerUmask,
		DefaultValue:    "true",
		Deprecated:      true,
		ToBeRemovedWith: "",
		Description: "When set to 'true', 'umask 0000' is used for all predefined steps (fetching sources, " +
			"artifacts and caches). This results in any files the steps produce being world writable. " +
			"When set to 'false', a uid/gid lookup is performed on the build image " +
			"and 'chown -RP -- %d:%d' is executed if the user is not root. This sets the files to be owned by " +
			"the same user as the build image, allowing a build user to write to the files without them having " +
			"to be explicitly world writable.",
	},
}

func GetAll() []FeatureFlag {
	return flags
}

func IsOn(value string) (bool, error) {
	if value == "" {
		return false, nil
	}

	on, err := strconv.ParseBool(value)
	if err != nil {
		return false, err
	}

	return on, nil
}
