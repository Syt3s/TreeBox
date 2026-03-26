// Copyright 2022 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package handler

import (
	"github.com/syt3s/TreeBox/internal/conf"
	"github.com/syt3s/TreeBox/internal/context"
)

func Home(ctx context.Context) error {
	ctx.Redirect(conf.App.ExternalURL)
	return nil
}
