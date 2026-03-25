package form

import (
	"encoding/json"
	"reflect"

	"github.com/flamego/flamego"

	"github.com/syt3s/TreeBox/internal/context"
)

func JSONBind(model interface{}) flamego.Handler {
	if reflect.TypeOf(model).Kind() == reflect.Ptr {
		panic("form: pointer can not be accepted as binding model")
	}

	return func(c context.Context) {
		obj := reflect.New(reflect.TypeOf(model))
		defer func() { c.Map(obj.Elem().Interface()) }()

		body, err := c.Request().Body().Bytes()
		if err != nil {
			c.Map(Error{Category: ErrorCategoryDeserialization, Error: err})
			return
		}

		if err := json.Unmarshal(body, obj.Interface()); err != nil {
			c.Map(Error{Category: ErrorCategoryDeserialization, Error: err})
			return
		}
	}
}
