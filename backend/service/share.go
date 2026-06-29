package service

import (
	"context"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
	shareservice "github.com/linkease/quick-start/istore-backend/modules/share/service"
	shareuser "github.com/linkease/quick-start/istore-backend/modules/share/user"
)

func ShareUserList(ctx context.Context) (*models.ShareUserListResponse, error) {
	users, err := newShareUserService().List(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.ShareUserListResponse{
		Result: &models.ShareUserListResponseResult{Users: users},
	}
	return &resp, nil
}

func ShareUserCreate(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.ShareUserCreateRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return ShareUserCreateTyped(ctx, req)
}

func ShareUserCreateTyped(ctx context.Context, req models.ShareUserCreateRequest) (*models.SDKNormalResponse, error) {
	if err := newShareUserService().Create(ctx, shareuser.CreateInput{UserName: req.UserName, Password: req.Password}); err != nil {
		return nil, err
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func ShareUserUpdate(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.ShareUserCreateRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return ShareUserUpdateTyped(ctx, req)
}

func ShareUserUpdateTyped(ctx context.Context, req models.ShareUserCreateRequest) (*models.SDKNormalResponse, error) {
	if err := newShareUserService().Update(ctx, shareuser.UpdateInput{UserName: req.UserName, Password: req.Password}); err != nil {
		return nil, err
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func ShareUserDelete(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.ShareUserDeleteRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return ShareUserDeleteTyped(ctx, req)
}

func ShareUserDeleteTyped(ctx context.Context, req models.ShareUserDeleteRequest) (*models.SDKNormalResponse, error) {
	if err := newShareUserService().Delete(ctx, shareuser.DeleteInput{UserName: req.UserName}); err != nil {
		return nil, err
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func ShareServiceList(ctx context.Context) (*models.ShareServiceListResponse, error) {
	services, err := newShareService().List(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.ShareServiceListResponse{
		Result: &models.ShareServiceListResponseResult{Services: services},
	}
	return &resp, nil
}

func ShareServiceCreate(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.ShareServiceCreateRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return ShareServiceCreateTyped(ctx, req)
}

func ShareServiceCreateTyped(ctx context.Context, req models.ShareServiceCreateRequest) (*models.SDKNormalResponse, error) {
	if err := newShareService().Create(ctx, shareservice.CreateInput{
		Name:   req.Name,
		Path:   req.Path,
		Samba:  req.Samba,
		Webdav: req.Webdav,
		Users:  req.Users,
	}); err != nil {
		return nil, err
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func ShareServiceUpdate(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.ShareServiceCreateRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return ShareServiceUpdateTyped(ctx, req)
}

func ShareServiceUpdateTyped(ctx context.Context, req models.ShareServiceCreateRequest) (*models.SDKNormalResponse, error) {
	if err := newShareService().Update(ctx, shareservice.UpdateInput{
		Name:   req.Name,
		Path:   req.Path,
		Samba:  req.Samba,
		Webdav: req.Webdav,
		Users:  req.Users,
	}); err != nil {
		return nil, err
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func ShareServiceDelete(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.ShareServicDeleteRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return ShareServiceDeleteTyped(ctx, req)
}

func ShareServiceDeleteTyped(ctx context.Context, req models.ShareServicDeleteRequest) (*models.SDKNormalResponse, error) {
	if err := newShareService().Delete(ctx, shareservice.DeleteInput{Name: req.Name}); err != nil {
		return nil, err
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func ShareWebdavConfig(ctx context.Context) (*models.ShareProtocolWebdavResponse, error) {
	config, err := newShareProtocol().WebdavConfig(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.ShareProtocolWebdavResponse{
		Result: config,
	}
	return &resp, nil
}

func ShareWebdavConfigUpdate(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.ShareProtocolWebdavConfig{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return ShareWebdavConfigUpdateTyped(ctx, req)
}

func ShareWebdavConfigUpdateTyped(ctx context.Context, req models.ShareProtocolWebdavConfig) (*models.SDKNormalResponse, error) {
	if err := newShareProtocol().UpdateWebdav(ctx, req); err != nil {
		return nil, err
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func ShareSambaConfig(ctx context.Context) (*models.ShareProtocolSambaResponse, error) {
	config, err := newShareProtocol().SambaConfig(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.ShareProtocolSambaResponse{
		Result: config,
	}
	return &resp, nil
}

func ShareSambaConfigUpdate(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.ShareProtocolSambaConfig{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return ShareSambaConfigUpdateTyped(ctx, req)
}

func ShareSambaConfigUpdateTyped(ctx context.Context, req models.ShareProtocolSambaConfig) (*models.SDKNormalResponse, error) {
	if err := newShareProtocol().UpdateSamba(ctx, req); err != nil {
		return nil, err
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}
