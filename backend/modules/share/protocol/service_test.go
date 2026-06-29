package protocol

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStore struct {
	webdavRecord WebdavRecord
	sambaRecord  SambaRecord

	readWebdavErr   error
	readSambaErr    error
	updateWebdavErr error
	updateSambaErr  error

	updatedWebdav models.ShareProtocolWebdavConfig
	updatedSamba  models.ShareProtocolSambaConfig
}

func (store *fakeStore) ReadWebdav(ctx context.Context) (WebdavRecord, error) {
	return store.webdavRecord, store.readWebdavErr
}

func (store *fakeStore) UpdateWebdav(ctx context.Context, input models.ShareProtocolWebdavConfig) error {
	store.updatedWebdav = input
	return store.updateWebdavErr
}

func (store *fakeStore) ReadSamba(ctx context.Context) (SambaRecord, error) {
	return store.sambaRecord, store.readSambaErr
}

func (store *fakeStore) UpdateSamba(ctx context.Context, input models.ShareProtocolSambaConfig) error {
	store.updatedSamba = input
	return store.updateSambaErr
}

func TestWebdavConfigParsesPort(t *testing.T) {
	svc := NewService(&fakeStore{webdavRecord: WebdavRecord{Port: "6086"}})

	config, err := svc.WebdavConfig(context.Background())
	if err != nil {
		t.Fatalf("WebdavConfig returned error: %v", err)
	}
	if config.Port != 6086 {
		t.Fatalf("Port = %d, want 6086", config.Port)
	}
}

func TestWebdavConfigKeepsMissingPortAsZeroAndReturnsParseErrors(t *testing.T) {
	svc := NewService(&fakeStore{})
	config, err := svc.WebdavConfig(context.Background())
	if err != nil {
		t.Fatalf("WebdavConfig returned error: %v", err)
	}
	if config.Port != 0 {
		t.Fatalf("Port = %d, want 0", config.Port)
	}

	svc = NewService(&fakeStore{webdavRecord: WebdavRecord{Port: "invalid"}})
	if _, err := svc.WebdavConfig(context.Background()); err == nil {
		t.Fatal("WebdavConfig expected parse error")
	}
}

func TestWebdavUpdateValidatesPortAndDelegates(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)

	err := svc.UpdateWebdav(context.Background(), models.ShareProtocolWebdavConfig{})
	if err == nil || err.Error() != "invalid port" {
		t.Fatalf("UpdateWebdav error = %v, want invalid port", err)
	}

	input := models.ShareProtocolWebdavConfig{Port: 6086}
	if err := svc.UpdateWebdav(context.Background(), input); err != nil {
		t.Fatalf("UpdateWebdav returned error: %v", err)
	}
	if store.updatedWebdav != input {
		t.Fatalf("updatedWebdav = %#v, want %#v", store.updatedWebdav, input)
	}
}

func TestSambaConfigParsesBooleanFlags(t *testing.T) {
	svc := NewService(&fakeStore{
		sambaRecord: SambaRecord{
			Workgroup:            "WORKGROUP",
			Description:          "QuickStart",
			DisableNetbios:       "1",
			Macos:                "0",
			AllowLegacyProtocols: "1",
		},
	})

	config, err := svc.SambaConfig(context.Background())
	if err != nil {
		t.Fatalf("SambaConfig returned error: %v", err)
	}
	if config.Workgroup != "WORKGROUP" || config.Description != "QuickStart" {
		t.Fatalf("SambaConfig strings = %#v", config)
	}
	if !config.DisableNetbios {
		t.Fatal("DisableNetbios = false, want true")
	}
	if config.EnableMacosCompatible {
		t.Fatal("EnableMacosCompatible = true, want false")
	}
	if !config.AllowLegacy {
		t.Fatal("AllowLegacy = false, want true")
	}
}

func TestSambaUpdateValidatesRequiredFieldsAndDelegates(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store)

	err := svc.UpdateSamba(context.Background(), models.ShareProtocolSambaConfig{Description: "desc"})
	if err == nil || err.Error() != "invalid workgroup" {
		t.Fatalf("UpdateSamba workgroup error = %v, want invalid workgroup", err)
	}

	err = svc.UpdateSamba(context.Background(), models.ShareProtocolSambaConfig{Workgroup: "WORKGROUP"})
	if err == nil || err.Error() != "invalid description" {
		t.Fatalf("UpdateSamba description error = %v, want invalid description", err)
	}

	input := models.ShareProtocolSambaConfig{
		Workgroup:             "WORKGROUP",
		Description:           "QuickStart",
		DisableNetbios:        true,
		EnableMacosCompatible: true,
		AllowLegacy:           true,
	}
	if err := svc.UpdateSamba(context.Background(), input); err != nil {
		t.Fatalf("UpdateSamba returned error: %v", err)
	}
	if store.updatedSamba != input {
		t.Fatalf("updatedSamba = %#v, want %#v", store.updatedSamba, input)
	}
}

func TestProtocolStoreErrorsArePropagated(t *testing.T) {
	readErr := errors.New("read failed")
	writeErr := errors.New("write failed")

	svc := NewService(&fakeStore{readWebdavErr: readErr, readSambaErr: readErr})
	if _, err := svc.WebdavConfig(context.Background()); !errors.Is(err, readErr) {
		t.Fatalf("WebdavConfig error = %v, want readErr", err)
	}
	if _, err := svc.SambaConfig(context.Background()); !errors.Is(err, readErr) {
		t.Fatalf("SambaConfig error = %v, want readErr", err)
	}

	svc = NewService(&fakeStore{updateWebdavErr: writeErr, updateSambaErr: writeErr})
	if err := svc.UpdateWebdav(context.Background(), models.ShareProtocolWebdavConfig{Port: 6086}); !errors.Is(err, writeErr) {
		t.Fatalf("UpdateWebdav error = %v, want writeErr", err)
	}
	if err := svc.UpdateSamba(context.Background(), models.ShareProtocolSambaConfig{Workgroup: "WORKGROUP", Description: "desc"}); !errors.Is(err, writeErr) {
		t.Fatalf("UpdateSamba error = %v, want writeErr", err)
	}
}
