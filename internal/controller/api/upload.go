package api

import (
	"errors"

	"go.uber.org/zap"

	"github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/logging"
	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/repository"
	"github.com/syt3s/TreeBox/internal/storage"
)

type UploadUserAssetResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	URL     string      `json:"url,omitempty"`
	User    *model.User `json:"user,omitempty"`
}

func UploadAvatar(ctx appctx.Context) error {
	return uploadUserAsset(ctx, storage.UploadKindAvatar)
}

func UploadBackground(ctx appctx.Context) error {
	return uploadUserAsset(ctx, storage.UploadKindBackground)
}

func uploadUserAsset(ctx appctx.Context, kind storage.UploadKind) error {
	logger := logging.FromContext(ctx.Request().Context()).With(
		zap.String("handler", "api.upload_user_asset"),
		zap.String("kind", string(kind)),
		zap.Uint("user_id", ctx.User.ID),
	)

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.JSONError(40000, "请选择要上传的图片")
	}

	url, err := storage.UploadUserImage(ctx.Request().Context(), ctx.User.UID, kind, fileHeader)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUploadNotConfigured):
			return ctx.JSONError(50300, "图片上传服务尚未配置")
		case errors.Is(err, storage.ErrEmptyFile):
			return ctx.JSONError(40000, "上传的文件不能为空")
		case errors.Is(err, storage.ErrFileTooLarge):
			return ctx.JSONError(40000, "图片不能超过 10MB")
		case errors.Is(err, storage.ErrUnsupportedImage):
			return ctx.JSONError(40000, "仅支持 JPG、PNG、WEBP、GIF 图片")
		default:
			logger.Error("failed to upload user asset", zap.Error(err))
			return ctx.JSONError(50000, "上传图片失败")
		}
	}

	currentUser, err := repository.Users.GetByID(ctx.Request().Context(), ctx.User.ID)
	if err != nil {
		logger.Error("failed to reload current user", zap.Error(err))
		return ctx.JSONError(50000, "更新图片失败")
	}

	updateOptions := repository.UpdateUserOptions{
		Name:       currentUser.Name,
		Avatar:     currentUser.Avatar,
		Background: currentUser.Background,
		Intro:      currentUser.Intro,
		Notify:     currentUser.Notify,
	}

	message := "头像上传成功"
	if kind == storage.UploadKindAvatar {
		updateOptions.Avatar = url
	} else {
		updateOptions.Background = url
		message = "背景上传成功"
	}

	if err := repository.Users.Update(ctx.Request().Context(), ctx.User.ID, updateOptions); err != nil {
		logger.Error("failed to persist uploaded asset", zap.Error(err))
		return ctx.JSONError(50000, "更新图片失败")
	}

	user, err := repository.Users.GetByID(ctx.Request().Context(), ctx.User.ID)
	if err != nil {
		logger.Error("failed to load user after asset update", zap.Error(err))
		return ctx.JSONError(50000, "获取最新资料失败")
	}

	return ctx.JSON(UploadUserAssetResponse{
		Success: true,
		Message: message,
		URL:     url,
		User:    user,
	})
}
