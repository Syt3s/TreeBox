package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/app/web"
	"github.com/syt3s/TreeBox/internal/branding"
	"github.com/syt3s/TreeBox/internal/logging"
)

func main() {
	if _, err := logging.Init(logging.Options{
		ServiceName: branding.ServiceName,
		LogDir:      "logs",
	}); err != nil {
		panic(err)
	}
	defer logging.Sync()

	app := cli.NewApp()
	app.Name = branding.ProductName
	app.Description = "匿名提问箱"
	app.Commands = []*cli.Command{
		web.Command,
	}

	if err := app.Run(os.Args); err != nil {
		logging.L().Fatal("启动app失败", zap.Error(err))
	}
}
