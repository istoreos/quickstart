package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeShareProtocolFacade struct {
	webdavResult *models.ShareProtocolWebdavConfig
	sambaResult  *models.ShareProtocolSambaConfig
	err          error

	updatedWebdav models.ShareProtocolWebdavConfig
	updatedSamba  models.ShareProtocolSambaConfig
}

func (svc *fakeShareProtocolFacade) WebdavConfig(ctx context.Context) (*models.ShareProtocolWebdavConfig, error) {
	return svc.webdavResult, svc.err
}

func (svc *fakeShareProtocolFacade) UpdateWebdav(ctx context.Context, input models.ShareProtocolWebdavConfig) error {
	svc.updatedWebdav = input
	return svc.err
}

func (svc *fakeShareProtocolFacade) SambaConfig(ctx context.Context) (*models.ShareProtocolSambaConfig, error) {
	return svc.sambaResult, svc.err
}

func (svc *fakeShareProtocolFacade) UpdateSamba(ctx context.Context, input models.ShareProtocolSambaConfig) error {
	svc.updatedSamba = input
	return svc.err
}

func TestShareProtocolCompatibilityDelegatesWebdavAndSamba(t *testing.T) {
	original := newShareProtocol
	defer func() { newShareProtocol = original }()

	facade := &fakeShareProtocolFacade{
		webdavResult: &models.ShareProtocolWebdavConfig{Port: 6086},
		sambaResult: &models.ShareProtocolSambaConfig{
			Workgroup:             "WORKGROUP",
			Description:           "QuickStart",
			DisableNetbios:        true,
			EnableMacosCompatible: true,
			AllowLegacy:           true,
		},
	}
	newShareProtocol = func() shareProtocolFacade {
		return facade
	}

	webdavResp, err := ShareWebdavConfig(context.Background())
	if err != nil {
		t.Fatalf("ShareWebdavConfig returned error: %v", err)
	}
	if webdavResp.Result.Port != 6086 {
		t.Fatalf("webdav response = %#v", webdavResp)
	}

	if _, err := ShareWebdavConfigUpdate(context.Background(), shareTestRequest(`{"port":6087}`)); err != nil {
		t.Fatalf("ShareWebdavConfigUpdate returned error: %v", err)
	}
	if facade.updatedWebdav.Port != 6087 {
		t.Fatalf("updatedWebdav = %#v", facade.updatedWebdav)
	}

	sambaResp, err := ShareSambaConfig(context.Background())
	if err != nil {
		t.Fatalf("ShareSambaConfig returned error: %v", err)
	}
	if sambaResp.Result.Workgroup != "WORKGROUP" || !sambaResp.Result.AllowLegacy {
		t.Fatalf("samba response = %#v", sambaResp)
	}

	body := `{"workgroup":"NAS","description":"Storage","disableNetbios":true,"enableMacosCompatible":true,"allowLegacy":true}`
	if _, err := ShareSambaConfigUpdate(context.Background(), shareTestRequest(body)); err != nil {
		t.Fatalf("ShareSambaConfigUpdate returned error: %v", err)
	}
	if facade.updatedSamba.Workgroup != "NAS" || facade.updatedSamba.Description != "Storage" || !facade.updatedSamba.DisableNetbios || !facade.updatedSamba.EnableMacosCompatible || !facade.updatedSamba.AllowLegacy {
		t.Fatalf("updatedSamba = %#v", facade.updatedSamba)
	}
}

func TestShareProtocolCompatibilityPropagatesFacadeErrors(t *testing.T) {
	original := newShareProtocol
	defer func() { newShareProtocol = original }()

	expectedErr := errors.New("facade failed")
	newShareProtocol = func() shareProtocolFacade {
		return &fakeShareProtocolFacade{err: expectedErr}
	}

	if _, err := ShareWebdavConfig(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("ShareWebdavConfig error = %v, want expectedErr", err)
	}
	if _, err := ShareWebdavConfigUpdate(context.Background(), shareTestRequest(`{"port":6087}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("ShareWebdavConfigUpdate error = %v, want expectedErr", err)
	}
	if _, err := ShareSambaConfig(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("ShareSambaConfig error = %v, want expectedErr", err)
	}
	if _, err := ShareSambaConfigUpdate(context.Background(), shareTestRequest(`{"workgroup":"NAS","description":"Storage"}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("ShareSambaConfigUpdate error = %v, want expectedErr", err)
	}
}

func TestShareProtocolCompatibilityKeepsDecodeErrors(t *testing.T) {
	original := newShareProtocol
	defer func() { newShareProtocol = original }()

	newShareProtocol = func() shareProtocolFacade {
		return &fakeShareProtocolFacade{}
	}
	if _, err := ShareWebdavConfigUpdate(context.Background(), shareTestRequest(`{`)); err == nil {
		t.Fatal("ShareWebdavConfigUpdate expected decode error")
	}
	if _, err := ShareSambaConfigUpdate(context.Background(), shareTestRequest(`{`)); err == nil {
		t.Fatal("ShareSambaConfigUpdate expected decode error")
	}
}
