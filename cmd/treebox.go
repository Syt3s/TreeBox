package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/branding"
	"github.com/syt3s/TreeBox/internal/cmd"
	"github.com/syt3s/TreeBox/internal/logging"
)

func main() {
	if _, err := logging.Init(logging.Options{
		ServiceName: branding.ServiceName,
		LogDir:      "logs",
	}); err != nil {
		panic(err)
	}
	defer func() { _ = logging.Sync() }()

	app := cli.NewApp()
	app.Name = branding.ProductName
	app.Description = "Anonymous question box"
	app.Commands = []*cli.Command{
		cmd.Web,
	}

	if err := app.Run(os.Args); err != nil {
		logging.L().Fatal("failed to start application", zap.Error(err))
	}
}
