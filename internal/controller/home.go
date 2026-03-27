package controller

import (
	"github.com/syt3s/TreeBox/internal/config"
	"github.com/syt3s/TreeBox/internal/http/appctx"
)

func Home(ctx appctx.Context) error {
	ctx.Redirect(config.App.ExternalURL)
	return nil
}
