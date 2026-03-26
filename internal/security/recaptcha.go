package security

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/syt3s/TreeBox/internal/conf"
)

type RecaptchaVerifyResponse struct {
	Success bool `json:"success"`
}

func VerifyRecaptcha(ctx context.Context, token string, remoteAddr string) (*RecaptchaVerifyResponse, error) {
	verifyURL := "https://www.google.com/recaptcha/api/siteverify"
	if conf.Recaptcha.TurnstileStyle {
		verifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	}

	form := url.Values{}
	form.Set("secret", conf.Recaptcha.ServerKey)
	form.Set("response", token)
	if strings.TrimSpace(remoteAddr) != "" {
		form.Set("remoteip", remoteAddr)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.Wrap(err, "create recaptcha request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "send recaptcha request")
	}
	defer func() { _ = resp.Body.Close() }()

	var result RecaptchaVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "decode recaptcha response")
	}

	return &result, nil
}
