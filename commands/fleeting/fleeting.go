package fleeting

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"gitlab.com/gitlab-org/fleeting/fleeting-artifact/pkg/installer"

	"gitlab.com/gitlab-org/gitlab-runner/commands"
	"gitlab.com/gitlab-org/gitlab-runner/common"
)

type runnerFleetingPlugin struct {
	RunnerName string
	Plugin     string
}

func getPlugins(context *cli.Context) []runnerFleetingPlugin {
	config := common.NewConfig()

	pathname := context.Parent().String("config")
	if pathname == "" {
		pathname = commands.GetDefaultConfigFile()
	}

	err := config.LoadConfig(pathname)
	if err != nil {
		logrus.Fatalln(err)
	}

	var results []runnerFleetingPlugin
	for _, runnerCfg := range config.Runners {
		if runnerCfg.Autoscaler == nil {
			continue
		}

		results = append(results, runnerFleetingPlugin{
			RunnerName: runnerCfg.ShortDescription(),
			Plugin:     runnerCfg.Autoscaler.Plugin,
		})
	}

	return results
}

func init() {
	common.RegisterCommand(cli.Command{
		Name:  "fleeting",
		Usage: "manage fleeting plugins",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "config, c",
				EnvVar: "CONFIG_FILE",
			},
		},
		Subcommands: []cli.Command{
			{
				Name:  "install",
				Usage: "install or update fleeting plugins",
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name: "upgrade",
					},
				},
				Action: func(cliCtx *cli.Context) {
					for _, plugin := range getPlugins(cliCtx) {
						_, err := installer.LookPath(plugin.Plugin, "")
						install := errors.Is(err, installer.ErrPluginNotFound) || cliCtx.Bool("upgrade")

						if install {
							if err := installer.Install(context.Background(), plugin.Plugin); err != nil {
								fmt.Printf("runner: %v, plugin: %v, update error: %v\n", plugin.RunnerName, plugin.Plugin, err)
								continue
							}
							path, _ := installer.LookPath(plugin.Plugin, "")
							fmt.Printf("runner: %v, plugin: %v, path: %v\n", plugin.RunnerName, plugin.Plugin, path)
						}
					}
				},
			},
			{
				Name:  "list",
				Usage: "list installed plugins",
				Action: func(cliCtx *cli.Context) {
					for _, plugin := range getPlugins(cliCtx) {
						path, err := installer.LookPath(plugin.Plugin, "")
						if err != nil {
							fmt.Printf("runner: %v, plugin: %v, error: %v\n", plugin.RunnerName, plugin.Plugin, err)
						} else {
							fmt.Printf("runner: %v, plugin: %v, path: %v\n", plugin.RunnerName, plugin.Plugin, path)
						}
					}
				},
			},
		},
	})
}
