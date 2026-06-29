package version

import (
	"context"
	"errors"
	"testing"
)

type fakeStore struct {
	board Board
	err   error
}

func (store fakeStore) ReadBoard(ctx context.Context) (Board, error) {
	return store.board, store.err
}

func TestGetBuildsVersionResult(t *testing.T) {
	t.Parallel()

	svc := NewService(fakeStore{
		board: Board{
			Kernel: "5.10.176",
			Model:  "LinkEase Box",
			Release: &Release{
				Description: "iStoreOS 22.03",
				Target:      "mediatek/filogic",
			},
		},
	}, "1.2.3")

	result, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.Model != "LinkEase Box" {
		t.Fatalf("Model = %q", result.Model)
	}
	if result.FirmwareVersion != "iStoreOS 22.03" {
		t.Fatalf("FirmwareVersion = %q", result.FirmwareVersion)
	}
	if result.KernelVersion != "5.10.176" {
		t.Fatalf("KernelVersion = %q", result.KernelVersion)
	}
	if result.Quickstart != "1.2.3" {
		t.Fatalf("Quickstart = %q", result.Quickstart)
	}
}

func TestGetNormalizesDefaultX86Model(t *testing.T) {
	t.Parallel()

	svc := NewService(fakeStore{
		board: Board{
			Kernel: "6.1",
			Model:  "Default string Default string",
			Release: &Release{
				Description: "iStoreOS x86",
				Target:      "x86/64",
			},
		},
	}, "dev")

	result, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.Model != "x86 Generic" {
		t.Fatalf("Model = %q, want x86 Generic", result.Model)
	}
}

func TestGetHandlesMissingRelease(t *testing.T) {
	t.Parallel()

	svc := NewService(fakeStore{
		board: Board{
			Kernel: "6.1",
			Model:  "Unknown",
		},
	}, "dev")

	result, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.FirmwareVersion != "" {
		t.Fatalf("FirmwareVersion = %q, want empty", result.FirmwareVersion)
	}
	if result.Model != "Unknown" {
		t.Fatalf("Model = %q", result.Model)
	}
}

func TestGetPropagatesStoreError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("ubus failed")
	svc := NewService(fakeStore{err: expectedErr}, "dev")

	if _, err := svc.Get(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("Get error = %v, want expectedErr", err)
	}
}
