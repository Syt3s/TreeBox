package form

import (
	"github.com/syt3s/TreeBox/internal/context"
)

func BindJSON(c context.Context, target interface{}) error {
	if err := c.BindJSON(target); err != nil {
		return c.JSONError(40000, "invalid request body")
	}
	return nil
}
