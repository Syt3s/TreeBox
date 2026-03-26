package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/uptrace/uptrace-go/uptrace"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/branding"
	"github.com/syt3s/TreeBox/internal/conf"
	"github.com/syt3s/TreeBox/internal/db"
	"github.com/syt3s/TreeBox/internal/logging"
	"github.com/syt3s/TreeBox/internal/router"
	"github.com/syt3s/TreeBox/internal/tracing"
)

var Web = &cli.Command{
	Name:   "web",
	Usage:  "Start web server",
	Action: runWeb,
}

func runWeb(ctx *cli.Context) error {
	if err := conf.Init(); err != nil {
		return errors.Wrap(err, "load configuration")
	}

	if _, err := logging.Init(logging.Options{
		ServiceName: branding.ServiceName,
		Production:  conf.App.Production,
		LogDir:      "logs",
	}); err != nil {
		return errors.Wrap(err, "init logger")
	}

	if conf.App.UptraceDSN != "" {
		uptrace.ConfigureOpentelemetry(
			uptrace.WithDSN(conf.App.UptraceDSN),
			uptrace.WithServiceName(branding.ServiceName),
			uptrace.WithServiceVersion(conf.BuildCommit),
		)
		logging.FromContext(ctx.Context).Info("tracing enabled")
	}

	dbType := conf.Database.Type

	var dsn string
	switch dbType {
	case "mysql", "":
		dsn = conf.MySQLDsn()
	case "postgres":
		dsn = conf.PostgresDsn()
	default:
		return errors.Errorf("unknown database type: %q", dbType)
	}
	conf.Database.DSN = dsn

	if _, err := db.Init(dbType, dsn); err != nil {
		return errors.Wrap(err, "connect to database")
	}

	logging.FromContext(ctx.Context).Info("starting web server",
		zap.String("external_url", conf.App.ExternalURL),
		zap.Int("port", conf.Server.Port),
		zap.String("db_type", dbType),
	)

	r := router.New(tracing.Middleware(branding.ProductName))
	if err := r.Run(fmt.Sprintf(":%d", conf.Server.Port)); err != nil {
		return errors.Wrap(err, "run gin server")
	}

	return nil
}
