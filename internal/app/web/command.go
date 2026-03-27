package web

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/uptrace/uptrace-go/uptrace"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/branding"
	"github.com/syt3s/TreeBox/internal/config"
	"github.com/syt3s/TreeBox/internal/http/router"
	"github.com/syt3s/TreeBox/internal/logging"
	"github.com/syt3s/TreeBox/internal/repository"
	"github.com/syt3s/TreeBox/internal/tracing"
)

var Command = &cli.Command{
	Name:   "web",
	Usage:  "启动web服务器",
	Action: runWeb,
}

func runWeb(ctx *cli.Context) error {
	if err := config.Init(); err != nil {
		return errors.Wrap(err, "加载配置")
	}

	if _, err := logging.Init(logging.Options{
		ServiceName: branding.ServiceName,
		Production:  config.App.Production,
		LogDir:      "logs",
	}); err != nil {
		return errors.Wrap(err, "初始化日志")
	}

	if config.App.UptraceDSN != "" {
		uptrace.ConfigureOpentelemetry(
			uptrace.WithDSN(config.App.UptraceDSN),
			uptrace.WithServiceName(branding.ServiceName),
			uptrace.WithServiceVersion(config.BuildCommit),
		)
		logging.FromContext(ctx.Context).Info("开始使用Uptrace进行分布式追踪")
	}

	dbType := config.Database.Type

	var dsn string
	switch dbType {
	case "mysql", "":
		dsn = config.MySQLDsn()
	case "postgres":
		dsn = config.PostgresDsn()
	default:
		return errors.Errorf("不支持该数据类型: %q", dbType)
	}
	config.Database.DSN = dsn

	if _, err := repository.Init(dbType, dsn); err != nil {
		return errors.Wrap(err, "连接数据库")
	}

	logging.FromContext(ctx.Context).Info("启动web服务器",
		zap.String("external_url", config.App.ExternalURL),
		zap.Int("port", config.Server.Port),
		zap.String("db_type", dbType),
	)

	r := router.New(tracing.Middleware(branding.ProductName))
	if err := r.Run(fmt.Sprintf(":%d", config.Server.Port)); err != nil {
		return errors.Wrap(err, "run gin server")
	}

	return nil
}
