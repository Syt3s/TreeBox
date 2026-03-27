package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	appcontext "github.com/syt3s/TreeBox/internal/http/appctx"
)

func TestRecoveryAPI(t *testing.T) {
	engine := gin.New()
	engine.Use(Recovery(), appcontext.Contexter(), ErrorHandler())

	api := engine.Group("/api/v2")
	api.Use(appcontext.APIEndpoint())
	api.GET("/panic", func(c *gin.Context) {
		panic("panic in api")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v2/panic", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, float64(50000), response["code"])
	require.Equal(t, "服务器内部错误", response["message"])
	require.Contains(t, response, "trace_id")
}

func TestRecoveryWeb(t *testing.T) {
	engine := gin.New()
	engine.Use(Recovery(), appcontext.Contexter(), ErrorHandler())
	engine.GET("/panic", func(c *gin.Context) {
		panic("panic in web")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, "internal server error", recorder.Body.String())
}
