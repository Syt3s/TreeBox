package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/syt3s/TreeBox/internal/cmd"
)

func main() {
	app := cli.NewApp()
	app.Name = "NekoBox"
	app.Description = "Anonymous question box"

	app.Commands = []*cli.Command{
		cmd.Web,
		cmd.Censor,
		cmd.Uid,
	}

	if err := app.Run(os.Args); err != nil {
		logrus.WithError(err).Fatal("Failed to start application")
	}
}
