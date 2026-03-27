package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/syt3s/TreeBox/internal/config"
)

func TestRouterHomeRedirect(t *testing.T) {
	engine := newTestRouter(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusFound, recorder.Code)
	require.Equal(t, "http://frontend.local", recorder.Header().Get("Location"))
}

func TestRouterRobotsTxt(t *testing.T) {
	engine := newTestRouter(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "text/plain; charset=utf-8", recorder.Header().Get("Content-Type"))
	require.Equal(t, "User-agent: *\nDisallow: /_/", recorder.Body.String())
}

func TestRouterNoRouteRedirect(t *testing.T) {
	engine := newTestRouter(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusFound, recorder.Code)
	require.Equal(t, "/", recorder.Header().Get("Location"))
}

func TestRouterAuthRequiredAPI(t *testing.T) {
	engine := newTestRouter(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v2/auth/me", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, float64(40100), response["code"])
	require.NotEmpty(t, response["message"])
	require.Contains(t, response, "trace_id")
}

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	oldApp := config.App
	t.Cleanup(func() {
		config.App = oldApp
	})

	config.App.Production = false
	config.App.MaintenanceMode = false
	config.App.ExternalURL = "http://frontend.local"

	return New()
}
