package commands

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"gitlab.com/gitlab-org/gitlab-runner/core/commands"
)

type ListCommand struct {
	configOptions
}

func (c *ListCommand) Execute(context *cli.Context) {
	err := c.loadConfig()
	if err != nil {
		log.Warningln(err)
		return
	}

	log.WithFields(log.Fields{
		"ConfigFile": c.ConfigFile,
	}).Println("Listing configured runners")

	for _, runner := range c.config.Runners {
		log.WithFields(log.Fields{
			"Executor": runner.RunnerSettings.Executor,
			"Token":    runner.RunnerCredentials.Token,
			"URL":      runner.RunnerCredentials.URL,
		}).Println(runner.Name)
	}
}

func init() {
	commands.RegisterCommand2("list", "List all configured runners", &ListCommand{})
}
