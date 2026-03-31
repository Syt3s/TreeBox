package api

import (
	"github.com/pkg/errors"

	"github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/repository"
)

func canManagePageQuestions(ctx appctx.Context, pageUser *model.User) (bool, *repository.TenantBootstrapResult, error) {
	if !ctx.IsLogged {
		return false, nil, nil
	}

	bootstrap, err := repository.Users.EnsureTenantBootstrap(ctx.Request().Context(), pageUser.ID)
	if err != nil {
		return false, nil, err
	}

	if ctx.User.ID == pageUser.ID {
		return true, bootstrap, nil
	}

	membership, err := repository.Tenants.GetMembership(ctx.Request().Context(), bootstrap.Tenant.ID, ctx.User.ID)
	if err != nil {
		if errors.Is(err, repository.ErrTenantMembershipNotExists) {
			return false, bootstrap, nil
		}
		return false, nil, err
	}

	return membership.Role.CanManageQuestions(), bootstrap, nil
}

func canManageQuestion(ctx appctx.Context, pageUser *model.User, question *model.Question) (bool, *repository.TenantBootstrapResult, error) {
	if !ctx.IsLogged {
		return false, nil, nil
	}

	bootstrap, err := repository.Users.EnsureTenantBootstrap(ctx.Request().Context(), pageUser.ID)
	if err != nil {
		return false, nil, err
	}

	if ctx.User.ID == pageUser.ID {
		return true, bootstrap, nil
	}

	tenantID := question.TenantID
	if tenantID == 0 && bootstrap != nil && bootstrap.Tenant != nil {
		tenantID = bootstrap.Tenant.ID
	}
	if tenantID == 0 {
		return false, bootstrap, nil
	}

	membership, err := repository.Tenants.GetMembership(ctx.Request().Context(), tenantID, ctx.User.ID)
	if err != nil {
		if errors.Is(err, repository.ErrTenantMembershipNotExists) {
			return false, bootstrap, nil
		}
		return false, nil, err
	}

	return membership.Role.CanManageQuestions(), bootstrap, nil
}
