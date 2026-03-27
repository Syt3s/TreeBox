package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	appcontext "github.com/syt3s/TreeBox/internal/http/appctx"
)

func TestErrorHandlerAPI(t *testing.T) {
	engine := gin.New()
	engine.Use(appcontext.Contexter(), ErrorHandler())

	api := engine.Group("/api/v2")
	api.Use(appcontext.APIEndpoint())
	api.GET("/boom", func(c *gin.Context) {
		_ = c.Error(errors.New("boom"))
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v2/boom", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, float64(50000), response["code"])
	require.Equal(t, "服务器内部错误", response["message"])
	require.Contains(t, response, "trace_id")
}

func TestErrorHandlerWeb(t *testing.T) {
	engine := gin.New()
	engine.Use(appcontext.Contexter(), ErrorHandler())
	engine.GET("/boom", func(c *gin.Context) {
		_ = c.Error(errors.New("boom"))
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/boom", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, "internal server error", recorder.Body.String())
}
