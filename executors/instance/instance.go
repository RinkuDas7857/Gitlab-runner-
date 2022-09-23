package instance

import (
	"errors"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/executors"
	"gitlab.com/gitlab-org/gitlab-runner/executors/internal/autoscaler"
)

type executor struct {
	executors.AbstractExecutor
	client executors.Client
}

func (e *executor) Name() string {
	return "shell+autoscaler"
}

func (e *executor) Prepare(options common.ExecutorPrepareOptions) error {
	err := e.AbstractExecutor.Prepare(options)
	if err != nil {
		return fmt.Errorf("preparing AbstractExecutor: %w", err)
	}

	if e.BuildShell.PassFile {
		return errors.New("shell autoscaler doesn't support shells that require script file")
	}

	environment, ok := e.Build.ExecutorData.(executors.Environment)
	if !ok {
		return errors.New("expected environment executor data")
	}

	e.Println("Dialing remote instance...")
	e.client, err = environment.Dial(options.Context)
	if err != nil {
		return fmt.Errorf("connecting to remote environment: %w", err)
	}

	return nil
}

func (e *executor) Run(cmd common.ExecutorCommand) error {
	return e.client.Run(executors.RunOptions{
		Command: e.BuildShell.CmdLine,
		Stdin:   strings.NewReader(cmd.Script),
		Stdout:  e.Trace,
		Stderr:  e.Trace,
	})
}

func (e *executor) Cleanup() {
	if e.client != nil {
		e.client.Close()
	}
	e.AbstractExecutor.Cleanup()
}

func init() {
	options := executors.ExecutorOptions{
		DefaultCustomBuildsDirEnabled: false,
		DefaultBuildsDir:              "builds",
		DefaultCacheDir:               "cache",
		SharedBuildsDir:               true,
		Shell: common.ShellScriptInfo{
			Shell:         "bash",
			RunnerCommand: "gitlab-runner",
		},
		ShowHostname: true,
	}

	creator := func() common.Executor {
		return &executor{
			AbstractExecutor: executors.AbstractExecutor{
				ExecutorOptions: options,
			},
		}
	}

	featuresUpdater := func(features *common.FeaturesInfo) {
		features.Variables = true
		features.Shared = true
	}

	common.RegisterExecutorProvider("shell+autoscaler", autoscaler.New(executors.DefaultExecutorProvider{
		Creator:          creator,
		FeaturesUpdater:  featuresUpdater,
		DefaultShellName: options.Shell.Shell,
	}))
}
