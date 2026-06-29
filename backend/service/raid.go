package service

import (
	"context"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/raid/writeflow"
)

// 参考逻辑
// https://github.com/lisaac/luci-app-diskman/blob/6ba3005ebdf1faabc7e0c4889c95caa3c153cafb/applications/luci-app-diskman/luasrc/model/diskman.lua#L489
func RaidPostCreate(ctx context.Context, r *http.Request) (*models.NasDiskPartitionFormatResponse, error) {
	req := models.RaidCreateRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}

	rname, err := newRaidWriteFlowService().Create(ctx, writeflow.CreateInput{
		Level:       req.Level,
		DevicePaths: req.DevicePaths,
	})
	if err != nil {
		return nil, err
	}
	return NasDiskPartitionFormatByDevicePath(ctx, rname)
}

// https://github.com/lisaac/luci-app-diskman/blob/6ba3005ebdf1faabc7e0c4889c95caa3c153cafb/applications/luci-app-diskman/luasrc/model/diskman.lua#L335
func RaidGetList(ctx context.Context) (*models.RaidListResponse, error) {
	model := models.RaidListResponseResult{}
	disks, err := newRaidInventoryService().List(ctx)
	if err != nil {
		return nil, err
	}
	model.Disks = disks
	resp := models.RaidListResponse{Result: &model}
	return &resp, nil
}

// https://github.com/lisaac/luci-app-diskman/blob/6ba3005ebdf1faabc7e0c4889c95caa3c153cafb/applications/luci-app-diskman/luasrc/model/cbi/diskman/partition.lua#L103
func RaidPostDelete(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.RaidDeleteRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}

	if err := newRaidWriteFlowService().Delete(ctx, writeflow.DeleteInput{
		Path:      req.Path,
		MountPath: req.MountPath,
		Members:   req.Members,
	}); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

// https://github.com/lisaac/luci-app-diskman/blob/6ba3005ebdf1faabc7e0c4889c95caa3c153cafb/applications/luci-app-diskman/luasrc/model/cbi/diskman/disks.lua#L246
func RaidGetCreateList(ctx context.Context) (*models.RaidCreateListResponse, error) {
	model := models.RaidCreateListResponseResult{}
	members, err := newRaidInventoryService().CreateList(ctx)
	if err != nil {
		return nil, err
	}
	model.Members = members
	req := models.RaidCreateListResponse{Result: &model}
	return &req, nil
}

// 扩容
func RaidPostAdd(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.RaidRecoverRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	if err := newRaidWriteFlowService().Add(ctx, writeflow.MemberInput{
		Path:       req.Path,
		MemberPath: req.MemberPath,
	}); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

// 扩容raid后，删除磁盘raid的容量就不会变化了，目前并没有设计扩容的反相操作
func RaidPostRemove(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.RaidRemoveRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	if err := newRaidWriteFlowService().Remove(ctx, writeflow.MemberInput{
		Path:       req.Path,
		MemberPath: req.MemberPath,
	}); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func RaidPostRecover(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.RaidRecoverRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	if err := newRaidWriteFlowService().Recover(ctx, writeflow.RecoverInput{
		Path:               req.Path,
		MemberPath:         req.MemberPath,
		CheckRaidPartition: req.CheckRaidPartition,
	}); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func RaidPostDetail(ctx context.Context, r *http.Request) (*models.RaidDetailResponse, error) {
	req := models.RaidDetailRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}

	model := models.RaidDetailResponseResult{}
	detail, err := newRaidInventoryService().Detail(ctx, req.Path)
	if err != nil {
		return nil, err
	}
	model.Detail = detail
	resp := models.RaidDetailResponse{Result: &model}
	return &resp, nil
}

// 自动检测raid分区重新生成mdadm config
func RaidAutoFix(ctx context.Context) (*models.SDKNormalResponse, error) {
	if err := newRaidMdadmConfigService().AutoFix(ctx); err != nil {
		return nil, err
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}
