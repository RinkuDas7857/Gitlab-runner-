package custom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/executors"
	"gitlab.com/gitlab-org/gitlab-runner/executors/custom/api"
	"gitlab.com/gitlab-org/gitlab-runner/executors/custom/command"
)

const (
	executorVariableEnvPrefix = "CUSTOM_ENV"

	ciJobImageEnv   = "CUSTOM_ENV_CI_JOB_IMAGE"
	ciJobPayloadEnv = "CUSTOM_ENV_CI_JOB_PAYLOAD"
)

type commandOutputs struct {
	stdout io.Writer
	stderr io.Writer
}

type prepareCommandOpts struct {
	executable string
	args       []string
	out        commandOutputs
}

type ConfigExecOutput struct {
	api.ConfigExecOutput
}

func (c *ConfigExecOutput) InjectInto(executor *executor) {
	if c.Hostname != nil {
		executor.Build.Hostname = *c.Hostname
	}

	if c.BuildsDir != nil {
		executor.Config.BuildsDir = *c.BuildsDir
	}

	if c.CacheDir != nil {
		executor.Config.CacheDir = *c.CacheDir
	}

	if c.BuildsDirIsShared != nil {
		executor.SharedBuildsDir = *c.BuildsDirIsShared
	}

	executor.driverInfo = c.Driver
}

type executor struct {
	executors.AbstractExecutor

	config  *config
	tempDir string

	driverInfo *api.DriverInfo
}

func (e *executor) Prepare(options common.ExecutorPrepareOptions) error {
	e.AbstractExecutor.PrepareConfiguration(options)

	err := e.prepareConfig()
	if err != nil {
		return err
	}

	e.tempDir, err = ioutil.TempDir("", "custom-executor")
	if err != nil {
		return err
	}

	err = e.dynamicConfig()
	if err != nil {
		return err
	}

	e.logStartupMessage()

	err = e.AbstractExecutor.PrepareBuildAndShell()
	if err != nil {
		return err
	}

	// nothing to do, as there's no prepare_script
	if e.config.PrepareExec == "" {
		return nil
	}

	ctx, cancelFunc := context.WithTimeout(e.Context, e.config.GetPrepareExecTimeout())
	defer cancelFunc()

	opts := prepareCommandOpts{
		executable: e.config.PrepareExec,
		args:       e.config.PrepareArgs,
		out:        e.defaultCommandOutputs(),
	}

	cmd, err := e.prepareCommand(ctx, opts)
	if err != nil {
		return err
	}

	return cmd.Run()
}

func (e *executor) prepareConfig() error {
	if e.Config.Custom == nil {
		return common.MakeBuildError("custom executor not configured")
	}

	e.config = &config{
		CustomConfig: e.Config.Custom,
	}

	if e.config.RunExec == "" {
		return common.MakeBuildError("custom executor is missing RunExec")
	}

	return nil
}

func (e *executor) dynamicConfig() error {
	if e.config.ConfigExec == "" {
		return nil
	}

	ctx, cancelFunc := context.WithTimeout(e.Context, e.config.GetConfigExecTimeout())
	defer cancelFunc()

	buf := bytes.NewBuffer(nil)
	outputs := commandOutputs{
		stdout: buf,
		stderr: e.Trace,
	}

	opts := prepareCommandOpts{
		executable: e.config.ConfigExec,
		args:       e.config.ConfigArgs,
		out:        outputs,
	}

	cmd, err := e.prepareCommand(ctx, opts)
	if err != nil {
		return err
	}

	err = cmd.Run()
	if err != nil {
		return err
	}

	jsonConfig := buf.Bytes()
	if len(jsonConfig) < 1 {
		return nil
	}

	config := new(ConfigExecOutput)

	err = json.Unmarshal(jsonConfig, config)
	if err != nil {
		return fmt.Errorf("error while parsing JSON output: %v", err)
	}

	config.InjectInto(e)

	return nil
}

func (e *executor) logStartupMessage() {
	const usageLine = "Using Custom executor"

	info := e.driverInfo
	if info == nil || info.Name == nil {
		e.Println(fmt.Sprintf("%s...", usageLine))
		return
	}

	if info.Version == nil {
		e.Println(fmt.Sprintf("%s with driver %s...", usageLine, *info.Name))
		return
	}

	e.Println(fmt.Sprintf("%s with driver %s %s...", usageLine, *info.Name, *info.Version))
}

func (e *executor) defaultCommandOutputs() commandOutputs {
	return commandOutputs{
		stdout: e.Trace,
		stderr: e.Trace,
	}
}

var commandFactory = command.New

func (e *executor) prepareCommand(ctx context.Context, opts prepareCommandOpts) (command.Command, error) {
	cmdOpts := command.CreateOptions{
		Dir:                 e.tempDir,
		Env: make([]string, 0),
		Stdout:              opts.out.stdout,
		Stderr:              opts.out.stderr,
		Logger:              e.BuildLogger,
		GracefulKillTimeout: e.config.GetGracefulKillTimeout(),
		ForceKillTimeout:    e.config.GetForceKillTimeout(),
	}

	var err error
	cmdOpts.Env, err = e.prepareVariables(cmdOpts.Env)
	if err != nil {
		return nil, err
	}

	return commandFactory(ctx, opts.executable, opts.args, cmdOpts), nil
}

func (e *executor) prepareVariables(variables []string) ([]string, error) {
	for _, variable := range e.Build.GetAllVariables() {
		variables = append(variables, fmt.Sprintf("%s_%s=%s", executorVariableEnvPrefix, variable.Key, variable.Value))
	}

	// since the variable is unique to the custom executor
	// at the moment, we add it separately from the other build variables
	// if we decide to export only the postfix in the build, this code can be removed
	imageName := e.Build.GetAllVariables().ExpandValue(e.Build.Image.Name)
	variables = append(variables, fmt.Sprintf("%s=%s", ciJobImageEnv, imageName))

	jobResponseJSON, err := e.Build.ToJSON()
	if err != nil {
		return []string{}, err
	}

	variables = append(variables, fmt.Sprintf("%s=%s", ciJobPayloadEnv, jobResponseJSON))

	return variables, nil
}

func (e *executor) Run(cmd common.ExecutorCommand) error {
	scriptDir, err := ioutil.TempDir(e.tempDir, "script")
	if err != nil {
		return err
	}

	scriptFile := filepath.Join(scriptDir, "script."+e.BuildShell.Extension)
	err = ioutil.WriteFile(scriptFile, []byte(cmd.Script), 0700)
	if err != nil {
		return err
	}

	args := append(e.config.RunArgs, scriptFile, string(cmd.Stage))

	opts := prepareCommandOpts{
		executable: e.config.RunExec,
		args:       args,
		out:        e.defaultCommandOutputs(),
	}

	execCmd, err := e.prepareCommand(cmd.Context, opts)
	if err != nil {
		return err
	}

	return execCmd.Run()
}

func (e *executor) Cleanup() {
	e.AbstractExecutor.Cleanup()

	err := e.prepareConfig()
	if err != nil {
		e.Warningln(err)

		// at this moment we don't care about the errors
		return
	}

	// nothing to do, as there's no cleanup_script
	if e.config.CleanupExec == "" {
		return
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), e.config.GetCleanupScriptTimeout())
	defer cancelFunc()

	stdoutLogger := e.BuildLogger.WithFields(logrus.Fields{"cleanup_std": "out"})
	stderrLogger := e.BuildLogger.WithFields(logrus.Fields{"cleanup_std": "err"})

	outputs := commandOutputs{
		stdout: stdoutLogger.WriterLevel(logrus.DebugLevel),
		stderr: stderrLogger.WriterLevel(logrus.WarnLevel),
	}

	opts := prepareCommandOpts{
		executable: e.config.CleanupExec,
		args:       e.config.CleanupArgs,
		out:        outputs,
	}

	cmd, err := e.prepareCommand(ctx, opts)
	if err != nil {
		e.Warningln("Cleanup script command preparation failed:", err)
	}

	err = cmd.Run()
	if err != nil {
		e.Warningln("Cleanup script failed:", err)
	}
}

func init() {
	options := executors.ExecutorOptions{
		DefaultCustomBuildsDirEnabled: false,
		Shell: common.ShellScriptInfo{
			Shell:         common.GetDefaultShell(),
			Type:          common.NormalShell,
			RunnerCommand: "gitlab-runner",
		},
		ShowHostname: false,
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

	common.RegisterExecutor("custom", executors.DefaultExecutorProvider{
		Creator:          creator,
		FeaturesUpdater:  featuresUpdater,
		DefaultShellName: options.Shell.Shell,
	})
}
