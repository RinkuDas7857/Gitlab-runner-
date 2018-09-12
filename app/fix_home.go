package app

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/homedir"
	"github.com/urfave/cli"
)

func FixHOME(cliCtx *cli.Context) error {
	// Fix home
	if key := homedir.Key(); os.Getenv(key) == "" {
		value := homedir.Get()
		if value == "" {
			return fmt.Errorf("the %q is not set", key)
		}
		os.Setenv(key, value)
	}

	return nil
}
