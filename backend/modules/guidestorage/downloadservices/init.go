package downloadservices

import (
	"context"
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Aria2InitInput struct {
	BtTracker    string
	ConfigPath   string
	DownloadPath string
	RPCToken     string
}

type QbittorrentInitInput struct {
	ConfigPath   string
	DownloadPath string
}

type TransmissionInitInput struct {
	ConfigPath   string
	DownloadPath string
}

type Writer interface {
	ValidateDownloadPath(path string) error
	EnsureDownloadDir(ctx context.Context, path string) error
	CanAccessPath(path string) bool
	WriteAria2Config(ctx context.Context, input Aria2InitInput) error
	WriteAria2Trackers(ctx context.Context, trackers []string) error
	RestartAria2(ctx context.Context) error
	WriteQbittorrentConfig(ctx context.Context, input QbittorrentInitInput) error
	RestartQbittorrent(ctx context.Context) error
	WriteTransmissionConfig(ctx context.Context, input TransmissionInitInput) error
	RestartTransmission(ctx context.Context) error
}

type Runtime interface {
	ResolveAria2Trackers(ctx context.Context, rawTrackers string) ([]string, error)
}

type Aria2InitService struct {
	writer  Writer
	runtime Runtime
}

type QbittorrentInitService struct {
	writer Writer
}

type TransmissionInitService struct {
	writer Writer
}

func NewAria2InitService(writer Writer, runtime Runtime) *Aria2InitService {
	return &Aria2InitService{
		writer:  writer,
		runtime: runtime,
	}
}

func NewQbittorrentInitService(writer Writer) *QbittorrentInitService {
	return &QbittorrentInitService{writer: writer}
}

func NewTransmissionInitService(writer Writer) *TransmissionInitService {
	return &TransmissionInitService{writer: writer}
}

func (service *Aria2InitService) InitAria2(ctx context.Context, input Aria2InitInput) (*models.SDKNormalResponse, error) {
	if err := service.writer.ValidateDownloadPath(input.DownloadPath); err != nil {
		return nil, err
	}
	if err := service.writer.EnsureDownloadDir(ctx, input.DownloadPath); err != nil {
		return nil, errors.New(input.DownloadPath + " 文件夹创建失败，请检查文件系统是否只读，或者已经存在同名文件")
	}
	if !service.writer.CanAccessPath(input.DownloadPath) {
		return nil, errors.New("无法访问下载路径")
	}
	if err := service.writer.WriteAria2Config(ctx, input); err != nil {
		return nil, err
	}
	trackers, err := service.runtime.ResolveAria2Trackers(ctx, input.BtTracker)
	if err != nil {
		return nil, err
	}
	if err := service.writer.WriteAria2Trackers(ctx, trackers); err != nil {
		return nil, err
	}
	if err := service.writer.RestartAria2(ctx); err != nil {
		return nil, errors.New("aria2启动失败")
	}
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (service *QbittorrentInitService) InitQbittorrent(ctx context.Context, input QbittorrentInitInput) (*models.SDKNormalResponse, error) {
	if err := service.writer.ValidateDownloadPath(input.DownloadPath); err != nil {
		return nil, err
	}
	if err := service.writer.EnsureDownloadDir(ctx, input.DownloadPath); err != nil {
		return nil, errors.New(input.DownloadPath + " 文件夹创建失败，请检查文件系统是否只读，或者已经存在同名文件")
	}
	if !service.writer.CanAccessPath(input.DownloadPath) {
		return nil, errors.New("无法访问下载路径")
	}
	if err := service.writer.WriteQbittorrentConfig(ctx, input); err != nil {
		return nil, errors.New("设置失败" + input.DownloadPath)
	}
	if err := service.writer.RestartQbittorrent(ctx); err != nil {
		return nil, errors.New("启动失败")
	}
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (service *TransmissionInitService) InitTransmission(ctx context.Context, input TransmissionInitInput) (*models.SDKNormalResponse, error) {
	if err := service.writer.ValidateDownloadPath(input.DownloadPath); err != nil {
		return nil, err
	}
	if err := service.writer.EnsureDownloadDir(ctx, input.DownloadPath); err != nil {
		return nil, errors.New(input.DownloadPath + " 文件夹创建失败，请检查文件系统是否只读，或者已经存在同名文件")
	}
	if !service.writer.CanAccessPath(input.DownloadPath) {
		return nil, errors.New("无法访问下载路径")
	}
	if err := service.writer.WriteTransmissionConfig(ctx, input); err != nil {
		return nil, errors.New("设置失败" + input.DownloadPath)
	}
	if err := service.writer.RestartTransmission(ctx); err != nil {
		return nil, errors.New("启动失败")
	}
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}
