package service

import (
	"context"
	"errors"
	"testing"
)

type fakeNasAutoMountFacade struct {
	called bool
	err    error
}

func (svc *fakeNasAutoMountFacade) Reload(ctx context.Context) error {
	svc.called = true
	return svc.err
}

func TestNasReloadDiskDelegatesToAutomountFacade(t *testing.T) {
	original := newNasAutoMountService
	defer func() { newNasAutoMountService = original }()

	facade := &fakeNasAutoMountFacade{}
	newNasAutoMountService = func() nasAutoMountFacade {
		return facade
	}

	NasReloadDisk()

	if !facade.called {
		t.Fatal("Reload was not called")
	}
}

func TestNasReloadDiskSwallowsAutomountErrors(t *testing.T) {
	original := newNasAutoMountService
	defer func() { newNasAutoMountService = original }()

	facade := &fakeNasAutoMountFacade{err: errors.New("reload failed")}
	newNasAutoMountService = func() nasAutoMountFacade {
		return facade
	}

	NasReloadDisk()

	if !facade.called {
		t.Fatal("Reload was not called")
	}
}
