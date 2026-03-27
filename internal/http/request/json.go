package request

import (
	"github.com/syt3s/TreeBox/internal/http/appctx"
)

func BindJSON(c appctx.Context, target interface{}) error {
	if err := c.BindJSON(target); err != nil {
		return c.JSONError(40000, "invalid request body")
	}
	return nil
}
