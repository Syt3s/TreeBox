package form

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	appcontext "github.com/syt3s/TreeBox/internal/context"
	"github.com/syt3s/TreeBox/internal/middleware"
)

func TestBindJSONInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler())
	engine.POST("/api/v2/test", appcontext.APIEndpoint(), appcontext.Wrap(func(ctx appcontext.Context) error {
		var request struct {
			Name string `json:"name"`
		}
		if err := BindJSON(ctx, &request); err != nil {
			return err
		}
		return ctx.JSON(request)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v2/test", strings.NewReader("{"))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, float64(40000), response["code"])
	require.Equal(t, "invalid request body", response["message"])
}
