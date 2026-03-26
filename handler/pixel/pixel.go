// Copyright 2024 E99p1ant. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package pixel

import (
	"io"
	"net/http"
	"path"
	"strconv"

	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/branding"
	"github.com/syt3s/TreeBox/internal/conf"
	"github.com/syt3s/TreeBox/internal/context"
	"github.com/syt3s/TreeBox/internal/logging"
)

func Proxy(ctx context.Context) error {
	uri := ctx.Param("**")
	method := ctx.Request().Method
	userID := strconv.Itoa(int(ctx.User.ID))
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "pixel.proxy"),
		zap.String("method", method),
		zap.String("uri", uri),
		zap.Uint("user_id", ctx.User.ID),
	)

	var body io.Reader
	if method == http.MethodPost || method == http.MethodPut {
		body = ctx.Request().Body
	}

	req, err := http.NewRequest(method, "http://pixel/", body)
	if err != nil {
		logger.Error("failed to create pixel request", zap.Error(err))
		return ctx.ServerError()
	}
	req.URL.Host = conf.Pixel.Host
	req.URL.Path = path.Join("/api/", uri)
	req.Header.Set(branding.PixelUserHeader, userID)
	req.Header.Set(branding.LegacyPixelUserHeader, userID)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("failed to send pixel request", zap.Error(err))
		return ctx.ServerError()
	}
	defer func() { _ = resp.Body.Close() }()

	for k, v := range resp.Header {
		ctx.ResponseWriter().Header()[k] = v
	}
	ctx.ResponseWriter().WriteHeader(resp.StatusCode)

	_, err = io.Copy(ctx.ResponseWriter(), resp.Body)
	if err != nil {
		logger.Error("failed to copy pixel response", zap.Error(err))
		return ctx.ServerError()
	}

	return nil
}
