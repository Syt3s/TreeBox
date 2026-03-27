// Copyright 2022 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package controller

import (
	"github.com/syt3s/TreeBox/internal/config"
	"github.com/syt3s/TreeBox/internal/http/appctx"
)

func Home(ctx appctx.Context) error {
	ctx.Redirect(config.App.ExternalURL)
	return nil
}
