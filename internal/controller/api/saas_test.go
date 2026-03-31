package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/dbutil"
	appcontext "github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/http/middleware"
	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/repository"
)

func TestListTenantMembers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTenants := repository.Tenants
	t.Cleanup(func() {
		repository.Tenants = oldTenants
	})

	repository.Tenants = &stubTenantRepository{
		tenantByUID: map[string]*model.Tenant{
			"tenant_1": {
				Model: dbutil.Model{ID: 1, CreatedAt: time.Now().UTC()},
				UID:   "tenant_1",
				Name:  "Acme",
				Plan:  model.TenantPlanFree,
			},
		},
		membershipByTenantID: map[uint]*model.TenantMember{
			1: {TenantID: 1, UserID: 7, Role: model.TenantRoleOwner},
		},
		memberList: []*repository.TenantMemberAccess{
			{
				Membership: &model.TenantMember{
					Model:    dbutil.Model{CreatedAt: time.Now().UTC()},
					TenantID: 1,
					UserID:   8,
					Role:     model.TenantRoleMember,
				},
				User: &model.User{
					Model:  gorm.Model{ID: 8},
					Name:   "Bob",
					Email:  "bob@acme.test",
					Domain: "bob",
				},
			},
		},
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{Model: gorm.Model{ID: 7}}))
	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.GET("/tenants/:tenantUID/members", appcontext.Wrap(ListTenantMembers))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v2/tenants/tenant_1/members", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Members []TenantMemberSummary `json:"members"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Len(t, response.Data.Members, 1)
	require.Equal(t, "bob@acme.test", response.Data.Members[0].Email)
	require.Equal(t, "member", response.Data.Members[0].Role)
}

func TestAddTenantMember(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTenants := repository.Tenants
	oldUsers := repository.Users
	t.Cleanup(func() {
		repository.Tenants = oldTenants
		repository.Users = oldUsers
	})

	repository.Users = &stubUserRepository{
		userByEmail: map[string]*model.User{
			"member@acme.test": {
				Model:  gorm.Model{ID: 9},
				Name:   "Member",
				Email:  "member@acme.test",
				Domain: "member",
			},
		},
	}
	repository.Tenants = &stubTenantRepository{
		tenantByUID: map[string]*model.Tenant{
			"tenant_1": {
				Model: dbutil.Model{ID: 1, CreatedAt: time.Now().UTC()},
				UID:   "tenant_1",
				Name:  "Acme",
				Plan:  model.TenantPlanFree,
			},
		},
		membershipByTenantID: map[uint]*model.TenantMember{
			1: {TenantID: 1, UserID: 7, Role: model.TenantRoleOwner},
		},
		addMemberResult: &repository.TenantMemberAccess{
			Membership: &model.TenantMember{
				Model:    dbutil.Model{CreatedAt: time.Now().UTC()},
				TenantID: 1,
				UserID:   9,
				Role:     model.TenantRoleMember,
			},
			User: &model.User{
				Model:  gorm.Model{ID: 9},
				Name:   "Member",
				Email:  "member@acme.test",
				Domain: "member",
			},
		},
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{Model: gorm.Model{ID: 7}}))
	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.POST("/tenants/:tenantUID/members", appcontext.Wrap(AddTenantMember))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v2/tenants/tenant_1/members", strings.NewReader(`{"email":"member@acme.test","role":"member"}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Member TenantMemberSummary `json:"member"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Equal(t, "member@acme.test", response.Data.Member.Email)
	require.Equal(t, "member", response.Data.Member.Role)
}

func TestListWorkspaceQuestions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTenants := repository.Tenants
	oldWorkspaces := repository.Workspaces
	oldQuestions := repository.Questions
	t.Cleanup(func() {
		repository.Tenants = oldTenants
		repository.Workspaces = oldWorkspaces
		repository.Questions = oldQuestions
	})

	repository.Tenants = &stubTenantRepository{
		tenantByID: map[uint]*model.Tenant{
			1: {
				Model: dbutil.Model{ID: 1, CreatedAt: time.Now().UTC()},
				UID:   "tenant_1",
				Name:  "Acme",
				Plan:  model.TenantPlanFree,
			},
		},
		membershipByTenantID: map[uint]*model.TenantMember{
			1: {TenantID: 1, UserID: 99, Role: model.TenantRoleMember},
		},
	}
	repository.Workspaces = &stubWorkspaceRepository{
		workspaceByUID: map[string]*model.Workspace{
			"workspace_1": {
				Model:           dbutil.Model{ID: 5, CreatedAt: time.Now().UTC()},
				UID:             "workspace_1",
				TenantID:        1,
				Name:            "Default workspace",
				CreatedByUserID: 7,
				IsDefault:       true,
			},
		},
	}
	repository.Questions = &stubQuestionRepository{
		questionsByWorkspaceID: map[uint][]*model.Question{
			5: {
				{
					Model:       dbutil.Model{ID: 101},
					TenantID:    1,
					WorkspaceID: 5,
					UserID:      7,
					Content:     "How are we doing?",
				},
			},
		},
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{Model: gorm.Model{ID: 99}}))
	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.GET("/workspaces/:workspaceUID/questions", appcontext.Wrap(ListWorkspaceQuestions))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v2/workspaces/workspace_1/questions", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Questions []model.Question `json:"questions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Len(t, response.Data.Questions, 1)
	require.Equal(t, uint(101), response.Data.Questions[0].ID)
}

func TestTenantMemberCanAnswerQuestion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldUsers := repository.Users
	oldQuestions := repository.Questions
	oldTenants := repository.Tenants
	t.Cleanup(func() {
		repository.Users = oldUsers
		repository.Questions = oldQuestions
		repository.Tenants = oldTenants
	})

	pageUser := &model.User{
		Model:  gorm.Model{ID: 42},
		Name:   "Alice",
		Domain: "alice",
	}

	repository.Users = &stubUserRepository{userByDomain: pageUser}
	repository.Questions = &stubQuestionRepository{
		questionsByID: map[uint]*model.Question{
			1: {
				Model:       dbutil.Model{ID: 1},
				TenantID:    1,
				WorkspaceID: 1,
				UserID:      42,
				Content:     "Hello",
			},
		},
	}
	repository.Tenants = &stubTenantRepository{
		membershipByTenantID: map[uint]*model.TenantMember{
			1: {TenantID: 1, UserID: 99, Role: model.TenantRoleMember},
		},
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{Model: gorm.Model{ID: 99}}))
	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.POST("/questions/:domain/:questionID/answer", appcontext.Wrap(AnswerQuestion))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v2/questions/alice/1/answer", strings.NewReader(`{"answer":"Team reply"}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
}
