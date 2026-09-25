package service

import (
	"context"
	"errors"
	"testing"

	"github.com/istoreos/quickstart/backend/models"
)

type fakeLanDeviceMigrationStore struct {
	snapshot            lanDeviceMigrationSnapshot
	receipt             []byte
	receiptExists       bool
	commitCalls         int
	restoreCalls        int
	commitErr           error
	verifyErr           error
	restoreErr          error
	failReadAfterCommit bool
}

func (store *fakeLanDeviceMigrationStore) Read(context.Context) (lanDeviceMigrationSnapshot, error) {
	if store.commitCalls > 0 && store.failReadAfterCommit {
		return lanDeviceMigrationSnapshot{}, store.verifyErr
	}
	return store.snapshot, nil
}
func (store *fakeLanDeviceMigrationStore) SnapshotReceipt(context.Context) ([]byte, bool, error) {
	return append([]byte(nil), store.receipt...), store.receiptExists, nil
}
func (store *fakeLanDeviceMigrationStore) Commit(_ context.Context, receipt lanDeviceMigrationReceipt) error {
	store.commitCalls++
	if store.commitErr != nil {
		return store.commitErr
	}
	store.snapshot.AppliedVersion = receipt.SourceVersion
	return nil
}
func (store *fakeLanDeviceMigrationStore) RestoreReceipt(context.Context, []byte, bool) error {
	store.restoreCalls++
	return store.restoreErr
}

func migrationFixtureItems() []*models.LanDeviceMigrationItem {
	return []*models.LanDeviceMigrationItem{
		{ID: "access:1", Kind: "access", Source: "firewall", Target: "access:device-1", Disposition: "adopt", Summary: "暂停联网"},
		{ID: "floatip:1", Kind: "floating_gateway", Source: "floatip", Target: "floating_gateway:default", Disposition: "adopt", Summary: "浮动网关"},
		{ID: "hostname:1", Kind: "hostname", Source: "dhcp", Target: "network:device-1", Disposition: "normalize", Summary: "living-room-tv"},
		{ID: "legacy-name:1", Kind: "alias", Source: "dhcp", Target: "profile:device-1", Disposition: "normalize", Summary: "客厅电视"},
		{ID: "route:bypass", Kind: "route", Source: "dhcp_tag", Target: "route:device-1", Disposition: "normalize", Summary: "旁路由"},
		{ID: "route:floating", Kind: "route", Source: "dhcp_tag", Target: "route:device-2", Disposition: "normalize", Summary: "浮动网关路线"},
		{ID: "speed:1", Kind: "speed", Source: "eqos", Target: "speed:device-1", Disposition: "adopt", Summary: "50 / 10 Mbit/s"},
		{ID: "static:1", Kind: "static", Source: "dhcp", Target: "network:device-2", Disposition: "adopt", Summary: "192.168.9.88"},
		{ID: "static:orphan", Kind: "static", Source: "dhcp", Target: "network:orphan:static:orphan", Disposition: "adopt", Summary: "离线规则"},
	}
}

func TestLanDeviceMigrationPlanIsReadOnlyAndComplete(t *testing.T) {
	store := &fakeLanDeviceMigrationStore{snapshot: lanDeviceMigrationSnapshot{
		Items: migrationFixtureItems(), SourceVersions: map[string]string{"dhcp": "one", "eqos": "two"},
	}}
	module := NewLanDeviceMigrationModule(store)
	response, err := module.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	plan := response.Result.Plan
	if !plan.CanApply || plan.Mode != "adopt_in_place" || len(plan.Items) != 9 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if store.commitCalls != 0 || store.restoreCalls != 0 {
		t.Fatal("dry-run wrote migration state")
	}
	foundUnicode := false
	for _, item := range plan.Items {
		if item.Summary == "客厅电视" {
			foundUnicode = true
		}
	}
	if !foundUnicode {
		t.Fatal("Unicode legacy alias was lost from the report")
	}
}

func TestLanDeviceMigrationReportsUnknownAndConflictWithoutDroppingItems(t *testing.T) {
	items := migrationFixtureItems()
	items = append(items,
		&models.LanDeviceMigrationItem{ID: "route:unknown", Kind: "route", Source: "dhcp_tag", Target: "route:unknown", Disposition: "unresolved", Summary: "未知标签"},
		&models.LanDeviceMigrationItem{ID: "static:duplicate", Kind: "static", Source: "dhcp", Target: "network:device-2", Disposition: "adopt", Summary: "duplicate"},
	)
	module := NewLanDeviceMigrationModule(&fakeLanDeviceMigrationStore{snapshot: lanDeviceMigrationSnapshot{Items: items, SourceVersions: map[string]string{"dhcp": "one"}}})
	response, err := module.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	plan := response.Result.Plan
	if plan.CanApply || plan.UnresolvedCount != 1 || plan.ConflictCount != 1 || len(plan.Items) != 11 {
		t.Fatalf("unexpected attention report: %#v", plan)
	}
}

func TestLanDeviceMigrationApplyIsVersionedAndIdempotent(t *testing.T) {
	store := &fakeLanDeviceMigrationStore{snapshot: lanDeviceMigrationSnapshot{Items: migrationFixtureItems(), SourceVersions: map[string]string{"dhcp": "one"}}}
	module := NewLanDeviceMigrationModule(store)
	preview, err := module.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	result, err := module.Apply(context.Background(), &models.LanDeviceMigrationRequest{ExpectedVersion: preview.Result.Plan.Version})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Result.Changed || result.Result.Error != nil || store.commitCalls != 1 {
		t.Fatalf("unexpected apply: %#v", result.Result)
	}
	repeated, err := module.Apply(context.Background(), &models.LanDeviceMigrationRequest{ExpectedVersion: preview.Result.Plan.Version})
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Result.Changed || repeated.Result.Error != nil || store.commitCalls != 1 {
		t.Fatalf("repeat was not idempotent: %#v", repeated.Result)
	}
	conflict, err := module.Apply(context.Background(), &models.LanDeviceMigrationRequest{ExpectedVersion: "stale"})
	if err != nil {
		t.Fatal(err)
	}
	if conflict.Result.Error == nil || conflict.Result.Error.Code != "conflict" {
		t.Fatalf("expected conflict: %#v", conflict.Result)
	}
}

func TestLanDeviceMigrationVerificationFailureRestoresReceipt(t *testing.T) {
	store := &fakeLanDeviceMigrationStore{snapshot: lanDeviceMigrationSnapshot{Items: migrationFixtureItems(), SourceVersions: map[string]string{"dhcp": "one"}}, failReadAfterCommit: true, verifyErr: errors.New("read failed")}
	module := NewLanDeviceMigrationModule(store)
	preview, _ := module.Plan(context.Background())
	result, err := module.Apply(context.Background(), &models.LanDeviceMigrationRequest{ExpectedVersion: preview.Result.Plan.Version})
	if err != nil {
		t.Fatal(err)
	}
	if result.Result.Error == nil || result.Result.Error.Code != "rolled_back" || store.restoreCalls != 1 {
		t.Fatalf("expected rollback: %#v", result.Result)
	}

	store = &fakeLanDeviceMigrationStore{snapshot: lanDeviceMigrationSnapshot{Items: migrationFixtureItems(), SourceVersions: map[string]string{"dhcp": "one"}}, failReadAfterCommit: true, verifyErr: errors.New("read failed"), restoreErr: errors.New("disk failed")}
	module = NewLanDeviceMigrationModule(store)
	preview, _ = module.Plan(context.Background())
	result, err = module.Apply(context.Background(), &models.LanDeviceMigrationRequest{ExpectedVersion: preview.Result.Plan.Version})
	if err != nil {
		t.Fatal(err)
	}
	if result.Result.Error == nil || result.Result.Error.Code != "recovery_required" {
		t.Fatalf("expected recovery_required: %#v", result.Result)
	}
}

func TestLanDeviceMigrationCommitFailureRestoresReceipt(t *testing.T) {
	store := &fakeLanDeviceMigrationStore{snapshot: lanDeviceMigrationSnapshot{Items: migrationFixtureItems(), SourceVersions: map[string]string{"dhcp": "one"}}, commitErr: errors.New("write failed")}
	module := NewLanDeviceMigrationModule(store)
	preview, _ := module.Plan(context.Background())
	result, err := module.Apply(context.Background(), &models.LanDeviceMigrationRequest{ExpectedVersion: preview.Result.Plan.Version})
	if err != nil {
		t.Fatal(err)
	}
	if result.Result.Error == nil || result.Result.Error.Code != "rolled_back" || store.restoreCalls != 1 {
		t.Fatalf("expected apply rollback: %#v", result.Result)
	}
}
