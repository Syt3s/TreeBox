package api

import (
	"context"

	"github.com/pkg/errors"

	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/repository"
)

func resolveQuestionWorkspace(ctx context.Context, pageUser *model.User, bootstrap *repository.TenantBootstrapResult) (*model.Workspace, error) {
	if bootstrap == nil || bootstrap.Workspace == nil {
		return nil, nil
	}

	if pageUser == nil || pageUser.RoutingWorkspaceID == 0 || pageUser.RoutingWorkspaceID == bootstrap.Workspace.ID {
		return bootstrap.Workspace, nil
	}

	workspace, err := repository.Workspaces.GetByID(ctx, pageUser.RoutingWorkspaceID)
	if err != nil {
		if err == repository.ErrWorkspaceNotExist {
			return bootstrap.Workspace, nil
		}
		return nil, err
	}

	membership, err := repository.Tenants.GetMembership(ctx, workspace.TenantID, pageUser.ID)
	if err != nil {
		if errors.Is(err, repository.ErrTenantMembershipNotExists) {
			return bootstrap.Workspace, nil
		}
		return nil, err
	}
	if !membership.Role.CanManageWorkspace() {
		return bootstrap.Workspace, nil
	}

	return workspace, nil
}

func questionBelongsToPage(question *model.Question, pageUser *model.User) bool {
	return question != nil && pageUser != nil && question.UserID == pageUser.ID
}
