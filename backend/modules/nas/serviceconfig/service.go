package serviceconfig

import (
	"context"
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
)

type SambaCreateService struct {
	statusReader   StatusReader
	runtimeReader  RuntimeReader
	configWriter   ConfigWriter
	templateWriter SambaTemplateWriter
}

func NewSambaCreateService(
	statusReader StatusReader,
	runtimeReader RuntimeReader,
	configWriter ConfigWriter,
	templateWriter SambaTemplateWriter,
) *SambaCreateService {
	return &SambaCreateService{
		statusReader:   statusReader,
		runtimeReader:  runtimeReader,
		configWriter:   configWriter,
		templateWriter: templateWriter,
	}
}

func (s SambaCreateService) Create(ctx context.Context, input SambaCreateInput) (*models.NasSambaCreateResponseResult, error) {
	if input.ShareName == "" || input.RootPath == "" || input.Username == "" || input.Password == "" {
		return nil, errors.New("param missing")
	}

	for _, share := range s.statusReader.ReadSambaShares() {
		if share != nil && input.ShareName == share.ShareName {
			return nil, errors.New("已存在同名samba共享")
		}
	}

	if err := s.configWriter.PrepareSamba(ctx); err != nil {
		return nil, err
	}
	if err := s.templateWriter.EnableRoot(); err != nil {
		return nil, err
	}
	if err := s.configWriter.CreateSambaUser(ctx, input.Username, input.Password); err != nil {
		return nil, errors.New("添加samba用户失败，请修改用户名再试，注意不能包含大写字母，并且第一位不是数字")
	}
	if err := s.configWriter.WriteSambaShare(ctx, input); err != nil {
		return nil, err
	}

	ipv4, err := s.runtimeReader.ReadLANIPv4(ctx)
	if err != nil {
		return nil, err
	}

	return &models.NasSambaCreateResponseResult{
		SambaURL: BuildSambaURL(ipv4, input.ShareName),
	}, nil
}

type WebdavCreateService struct {
	statusReader  StatusReader
	runtimeReader RuntimeReader
	configWriter  ConfigWriter
}

func NewWebdavCreateService(statusReader StatusReader, runtimeReader RuntimeReader, configWriter ConfigWriter) *WebdavCreateService {
	return &WebdavCreateService{
		statusReader:  statusReader,
		runtimeReader: runtimeReader,
		configWriter:  configWriter,
	}
}

func (s WebdavCreateService) Create(ctx context.Context, input WebdavCreateInput) (*models.NasWebdavCreateResponseResult, error) {
	if input.RootPath == "" || input.Username == "" || input.Password == "" {
		return nil, errors.New("param missing")
	}

	if err := s.configWriter.WriteWebdavConfig(ctx, input); err != nil {
		return nil, err
	}
	if err := s.configWriter.RestartWebdav(ctx); err != nil {
		return nil, err
	}

	ipv4, err := s.runtimeReader.ReadLANIPv4(ctx)
	if err != nil {
		return nil, err
	}

	port, _ := s.statusReader.ReadWebdavPort()
	return &models.NasWebdavCreateResponseResult{
		Username:  input.Username,
		WebdavURL: BuildWebdavURL(ipv4, port),
	}, nil
}

type WebdavStatusService struct {
	statusReader StatusReader
}

func NewWebdavStatusService(statusReader StatusReader) *WebdavStatusService {
	return &WebdavStatusService{statusReader: statusReader}
}

func (s WebdavStatusService) Read(ctx context.Context) (*models.NasWebdavStatusResponseResult, error) {
	info := s.statusReader.ReadWebdavInfo()
	return &models.NasWebdavStatusResponseResult{
		Path:     info.Path,
		Port:     info.Port,
		Username: info.Username,
		Password: info.Password,
	}, nil
}

type StatusService struct {
	statusReader  StatusReader
	runtimeReader RuntimeReader
}

func NewStatusService(statusReader StatusReader, runtimeReader RuntimeReader) *StatusService {
	return &StatusService{
		statusReader:  statusReader,
		runtimeReader: runtimeReader,
	}
}

func (s StatusService) Read(ctx context.Context) (*models.NasServiceResponseResult, error) {
	model := &models.NasServiceResponseResult{}
	model.Sambas = s.statusReader.ReadSambaShares()

	webdav := s.statusReader.ReadWebdavInfo()
	model.Webdav = &webdav

	linkease := &models.NasServiceLinkeaseInfo{}
	enabledByConfig, port, err := s.statusReader.ReadLinkeaseInfo(ctx)
	if err != nil {
		return nil, err
	}
	if enabledByConfig && s.runtimeReader.HasLinkeaseBinary() {
		linkease.Enabel = true
		linkease.Port = port
	}
	model.Linkease = linkease

	return model, nil
}
