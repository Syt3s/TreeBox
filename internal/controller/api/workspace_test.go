package api

import (
	"context"
	"encoding/json"
	"errors"
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

func TestListTenants(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTenants := repository.Tenants
	t.Cleanup(func() {
		repository.Tenants = oldTenants
	})

	repository.Tenants = &stubTenantRepository{
		listByUserID: []*repository.TenantMembership{
			{
				Tenant: &model.Tenant{
					UID:        "tenant_1",
					Name:       "Acme",
					Plan:       model.TenantPlanFree,
					IsPersonal: true,
					Model: dbutil.Model{
						ID:        1,
						CreatedAt: time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC),
					},
				},
				Role: model.TenantRoleOwner,
			},
		},
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{
		Model: gorm.Model{ID: 7},
	}))
	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.GET("/tenants", appcontext.Wrap(ListTenants))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v2/tenants", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Tenants []TenantSummary `json:"tenants"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.Len(t, response.Data.Tenants, 1)
	require.Equal(t, "tenant_1", response.Data.Tenants[0].UID)
	require.Equal(t, "owner", response.Data.Tenants[0].Role)
}

func TestCreateWorkspaceRejectsUnauthorizedRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTenants := repository.Tenants
	oldWorkspaces := repository.Workspaces
	t.Cleanup(func() {
		repository.Tenants = oldTenants
		repository.Workspaces = oldWorkspaces
	})

	repository.Tenants = &stubTenantRepository{
		tenantByUID: map[string]*model.Tenant{
			"tenant_1": {
				Model: dbutil.Model{
					ID:        1,
					CreatedAt: time.Now().UTC(),
				},
				UID:   "tenant_1",
				Name:  "Acme",
				Plan:  model.TenantPlanFree,
			},
		},
	}
	repository.Workspaces = &stubWorkspaceRepository{
		createErr: repository.ErrTenantAccessDenied,
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{
		Model: gorm.Model{ID: 11},
	}))
	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.POST("/workspaces", appcontext.Wrap(CreateWorkspace))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v2/workspaces", strings.NewReader(`{"tenant_uid":"tenant_1","name":"Ops"}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	requireAPIError(t, recorder, http.StatusForbidden, 40300, "无权创建该租户下的工作区")
}

func TestListTenantAuditLogsRejectsViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTenants := repository.Tenants
	t.Cleanup(func() {
		repository.Tenants = oldTenants
	})

	repository.Tenants = &stubTenantRepository{
		tenantByUID: map[string]*model.Tenant{
			"tenant_1": {
				Model: dbutil.Model{
					ID:        1,
					CreatedAt: time.Now().UTC(),
				},
				UID:   "tenant_1",
				Name:  "Acme",
				Plan:  model.TenantPlanFree,
			},
		},
		membershipByTenantID: map[uint]*model.TenantMember{
			1: {
				TenantID: 1,
				UserID:   42,
				Role:     model.TenantRoleViewer,
			},
		},
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{
		Model: gorm.Model{ID: 42},
	}))
	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.GET("/tenants/:tenantUID/audit-logs", appcontext.Wrap(ListTenantAuditLogs))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v2/tenants/tenant_1/audit-logs", nil)
	engine.ServeHTTP(recorder, request)

	requireAPIError(t, recorder, http.StatusForbidden, 40300, "当前角色无权查看审计日志")
}

type stubTenantRepository struct {
	tenantByID            map[uint]*model.Tenant
	tenantByUID           map[string]*model.Tenant
	membershipByTenantID  map[uint]*model.TenantMember
	listByUserID          []*repository.TenantMembership
	memberList            []*repository.TenantMemberAccess
	addMemberResult       *repository.TenantMemberAccess
	updateMemberResult    *repository.TenantMemberAccess
	addMemberErr          error
	updateMemberErr       error
	removeMemberErr       error
	membershipLookupError error
}

func (s *stubTenantRepository) GetByID(_ context.Context, id uint) (*model.Tenant, error) {
	if tenant, ok := s.tenantByID[id]; ok {
		return tenant, nil
	}
	return nil, repository.ErrTenantNotExist
}

func (s *stubTenantRepository) GetByUID(_ context.Context, uid string) (*model.Tenant, error) {
	if tenant, ok := s.tenantByUID[uid]; ok {
		return tenant, nil
	}
	return nil, repository.ErrTenantNotExist
}

func (s *stubTenantRepository) GetMembership(_ context.Context, tenantID, _ uint) (*model.TenantMember, error) {
	if s.membershipLookupError != nil {
		return nil, s.membershipLookupError
	}
	if membership, ok := s.membershipByTenantID[tenantID]; ok {
		return membership, nil
	}
	return nil, repository.ErrTenantMembershipNotExists
}

func (s *stubTenantRepository) ListByUserID(context.Context, uint) ([]*repository.TenantMembership, error) {
	return s.listByUserID, nil
}

func (s *stubTenantRepository) ListMembers(context.Context, uint) ([]*repository.TenantMemberAccess, error) {
	return s.memberList, nil
}

func (s *stubTenantRepository) AddMember(context.Context, repository.AddTenantMemberOptions) (*repository.TenantMemberAccess, error) {
	if s.addMemberErr != nil {
		return nil, s.addMemberErr
	}
	return s.addMemberResult, nil
}

func (s *stubTenantRepository) UpdateMemberRole(context.Context, repository.UpdateTenantMemberRoleOptions) (*repository.TenantMemberAccess, error) {
	if s.updateMemberErr != nil {
		return nil, s.updateMemberErr
	}
	return s.updateMemberResult, nil
}

func (s *stubTenantRepository) RemoveMember(context.Context, repository.RemoveTenantMemberOptions) error {
	return s.removeMemberErr
}

type stubWorkspaceRepository struct {
	workspaceByUID map[string]*model.Workspace
	list      []*repository.WorkspaceAccess
	createErr error
	created   *model.Workspace
}

func (s *stubWorkspaceRepository) GetByID(context.Context, uint) (*model.Workspace, error) {
	return nil, repository.ErrWorkspaceNotExist
}

func (s *stubWorkspaceRepository) GetByUID(_ context.Context, uid string) (*model.Workspace, error) {
	if workspace, ok := s.workspaceByUID[uid]; ok {
		return workspace, nil
	}
	return nil, repository.ErrWorkspaceNotExist
}

func (s *stubWorkspaceRepository) ListByUserID(context.Context, uint) ([]*repository.WorkspaceAccess, error) {
	return s.list, nil
}

func (s *stubWorkspaceRepository) CreateForTenantMember(context.Context, repository.CreateWorkspaceOptions) (*model.Workspace, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	if s.created != nil {
		return s.created, nil
	}
	return nil, errors.New("not implemented")
}

type stubAuditLogRepository struct {
	logs []*model.AuditLog
}

func (s *stubAuditLogRepository) Record(context.Context, repository.RecordAuditLogOptions) (*model.AuditLog, error) {
	return nil, errors.New("not implemented")
}

func (s *stubAuditLogRepository) ListByTenantID(context.Context, uint, int) ([]*model.AuditLog, error) {
	return s.logs, nil
}
