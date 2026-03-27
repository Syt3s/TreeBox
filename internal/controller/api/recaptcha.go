package api

import (
	"github.com/syt3s/TreeBox/internal/config"
	"github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/security"
	"go.uber.org/zap"
)

func verifyRecaptchaIfNeeded(ctx appctx.Context, logger *zap.Logger, token string) error {
	if !config.App.Production {
		return nil
	}

	resp, err := security.VerifyRecaptcha(ctx.Request().Context(), token, ctx.Request().RemoteAddr)
	if err != nil {
		logger.Error("failed to verify recaptcha", zap.Error(err))
		return ctx.JSONError(50000, "验证码校验失败")
	}
	if !resp.Success {
		return ctx.JSONError(40000, "验证码错误")
	}

	return nil
}
