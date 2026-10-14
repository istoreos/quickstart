package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskTransactionJournalPersistsReplayAndConflict(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "transactions.json")
	firstJournal := NewDefaultTaskTransactionJournal()
	firstJournal.path = path
	first, err := firstJournal.Begin(ctx, "network", "request-1", "fingerprint-a")
	if err != nil || first.Replay || first.Conflict {
		t.Fatalf("first = %#v, %v", first, err)
	}
	if err := firstJournal.Advance(ctx, first.Transaction, "verify", "committed", ""); err != nil {
		t.Fatal(err)
	}

	restarted := NewDefaultTaskTransactionJournal()
	restarted.path = path
	replay, err := restarted.Begin(ctx, "network", "request-1", "fingerprint-a")
	if err != nil || !replay.Replay || replay.Conflict || replay.Transaction.Status != "committed" || !replay.Transaction.Replayed {
		t.Fatalf("replay = %#v, %v", replay, err)
	}
	conflict, err := restarted.Begin(ctx, "network", "request-1", "fingerprint-b")
	if err != nil || !conflict.Conflict || conflict.Replay {
		t.Fatalf("conflict = %#v, %v", conflict, err)
	}
}

func TestTaskTransactionJournalMarksInterruptedWorkForRecovery(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "transactions.json")
	beforeCrash := NewDefaultTaskTransactionJournal()
	beforeCrash.path = path
	started, err := beforeCrash.Begin(ctx, "restrictions", "request-2", "fingerprint")
	if err != nil {
		t.Fatal(err)
	}
	if err := beforeCrash.Advance(ctx, started.Transaction, "apply", "in_progress", "restore_task_snapshot"); err != nil {
		t.Fatal(err)
	}

	afterRestart := NewDefaultTaskTransactionJournal()
	afterRestart.path = path
	recovered, err := afterRestart.Begin(ctx, "restrictions", "request-2", "fingerprint")
	if err != nil || !recovered.Replay || recovered.Transaction.Status != "recovery_required" || recovered.Transaction.Stage != "recover" || recovered.Transaction.RecoveryAction != "restore_task_snapshot" {
		t.Fatalf("recovered = %#v, %v", recovered, err)
	}
}

func TestTaskTransactionJournalHasBoundedPersistentOrder(t *testing.T) {
	ctx := context.Background()
	journal := NewDefaultTaskTransactionJournal()
	journal.path = filepath.Join(t.TempDir(), "transactions.json")
	journal.limit = 2
	for _, key := range []string{"one", "two", "three"} {
		begin, err := journal.Begin(ctx, "profile", key, key)
		if err != nil {
			t.Fatal(err)
		}
		if err := journal.Advance(ctx, begin.Transaction, "verify", "committed", ""); err != nil {
			t.Fatal(err)
		}
	}
	document, err := journal.read()
	if err != nil || len(document.Records) != 2 || len(document.Order) != 2 {
		t.Fatalf("document = %#v, %v", document, err)
	}
}

func TestTaskTransactionJournalStoresOnlyRequestFingerprint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transactions.json")
	journal := NewDefaultTaskTransactionJournal()
	journal.path = path
	secret := `{"deviceId":"mac:aa","alias":"家庭私密设备名"}`
	if _, err := journal.Begin(context.Background(), "profile", "privacy", secret); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "家庭私密设备名") || strings.Contains(string(raw), secret) {
		t.Fatalf("transaction journal leaked request content: %s", raw)
	}
}
