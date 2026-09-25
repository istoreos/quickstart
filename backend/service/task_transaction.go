package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const (
	defaultTaskTransactionPath  = "/etc/quickstart/task-transactions-v1.json"
	defaultTaskTransactionLimit = 512
)

type taskTransactionRecord struct {
	Task           string `json:"task"`
	IdempotencyKey string `json:"idempotencyKey"`
	Fingerprint    string `json:"fingerprint"`
	Stage          string `json:"stage"`
	Status         string `json:"status"`
	RecoveryAction string `json:"recoveryAction,omitempty"`
	UpdatedAt      string `json:"updatedAt"`
}

type taskTransactionDocument struct {
	SchemaVersion int                              `json:"schemaVersion"`
	Order         []string                         `json:"order"`
	Records       map[string]taskTransactionRecord `json:"records"`
}

type taskTransactionBegin struct {
	Transaction *models.TaskTransaction
	Replay      bool
	Conflict    bool
}

type TaskTransactionJournal struct {
	mu          sync.Mutex
	path        string
	limit       int
	persist     func(string, []byte) error
	memory      *taskTransactionDocument
	recoveryRun bool
	now         func() time.Time
}

var taskTransactionFileMu sync.Mutex

func NewDefaultTaskTransactionJournal() *TaskTransactionJournal {
	return &TaskTransactionJournal{path: defaultTaskTransactionPath, limit: defaultTaskTransactionLimit, persist: persistClassificationOverrides, now: time.Now}
}

func newMemoryTaskTransactionJournal() *TaskTransactionJournal {
	document := emptyTaskTransactionDocument()
	return &TaskTransactionJournal{limit: defaultTaskTransactionLimit, memory: &document, now: time.Now}
}

func (journal *TaskTransactionJournal) Begin(_ context.Context, task, key, fingerprint string) (taskTransactionBegin, error) {
	if key == "" {
		return taskTransactionBegin{Transaction: &models.TaskTransaction{Task: task, Stage: "plan", Status: "in_progress"}}, nil
	}
	if len(key) > 128 || len(fingerprint) > 16384 {
		return taskTransactionBegin{}, errors.New("invalid idempotency record")
	}
	fingerprint = hashTaskTransactionFingerprint(fingerprint)
	journal.mu.Lock()
	defer journal.mu.Unlock()
	unlock := journal.lockFile()
	defer unlock()
	document, err := journal.read()
	if err != nil {
		return taskTransactionBegin{}, err
	}
	if err := journal.recoverInterrupted(&document); err != nil {
		return taskTransactionBegin{}, err
	}
	recordKey := task + "\x00" + key
	if record, ok := document.Records[recordKey]; ok {
		transaction := publicTaskTransaction(record)
		transaction.Replayed = true
		return taskTransactionBegin{Transaction: transaction, Replay: record.Fingerprint == fingerprint, Conflict: record.Fingerprint != fingerprint}, nil
	}
	record := taskTransactionRecord{Task: task, IdempotencyKey: key, Fingerprint: fingerprint, Stage: "plan", Status: "in_progress", UpdatedAt: journal.now().UTC().Format(time.RFC3339Nano)}
	document.Records[recordKey] = record
	document.Order = append(document.Order, recordKey)
	for len(document.Order) > journal.limit {
		delete(document.Records, document.Order[0])
		document.Order = document.Order[1:]
	}
	if err := journal.write(document); err != nil {
		return taskTransactionBegin{}, err
	}
	return taskTransactionBegin{Transaction: publicTaskTransaction(record)}, nil
}

func hashTaskTransactionFingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (journal *TaskTransactionJournal) Advance(_ context.Context, transaction *models.TaskTransaction, stage, status, recoveryAction string) error {
	if transaction == nil {
		return nil
	}
	transaction.Stage, transaction.Status, transaction.RecoveryAction = stage, status, recoveryAction
	if transaction.IdempotencyKey == "" {
		return nil
	}
	journal.mu.Lock()
	defer journal.mu.Unlock()
	unlock := journal.lockFile()
	defer unlock()
	document, err := journal.read()
	if err != nil {
		return err
	}
	recordKey := transaction.Task + "\x00" + transaction.IdempotencyKey
	record, ok := document.Records[recordKey]
	if !ok {
		return errors.New("transaction record missing")
	}
	record.Stage, record.Status, record.RecoveryAction = stage, status, recoveryAction
	record.UpdatedAt = journal.now().UTC().Format(time.RFC3339Nano)
	document.Records[recordKey] = record
	return journal.write(document)
}

func (journal *TaskTransactionJournal) recoverInterrupted(document *taskTransactionDocument) error {
	if journal.recoveryRun {
		return nil
	}
	journal.recoveryRun = true
	changed := false
	for key, record := range document.Records {
		if record.Status != "in_progress" {
			continue
		}
		record.Status = "recovery_required"
		record.Stage = "recover"
		record.RecoveryAction = "restore_task_snapshot"
		document.Records[key] = record
		changed = true
	}
	if changed {
		return journal.write(*document)
	}
	return nil
}

func (journal *TaskTransactionJournal) read() (taskTransactionDocument, error) {
	if journal.memory != nil {
		return cloneTaskTransactionDocument(*journal.memory), nil
	}
	raw, err := os.ReadFile(journal.path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyTaskTransactionDocument(), nil
	}
	if err != nil {
		return taskTransactionDocument{}, err
	}
	var document taskTransactionDocument
	if json.Unmarshal(raw, &document) != nil || document.SchemaVersion != 1 || document.Records == nil || len(document.Records) > journal.limit || len(document.Order) > journal.limit {
		return taskTransactionDocument{}, errors.New("invalid task transaction journal")
	}
	return document, nil
}

func (journal *TaskTransactionJournal) write(document taskTransactionDocument) error {
	if journal.memory != nil {
		copy := cloneTaskTransactionDocument(document)
		journal.memory = &copy
		return nil
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return err
	}
	return journal.persist(journal.path, raw)
}

func (journal *TaskTransactionJournal) lockFile() func() {
	if journal.memory != nil {
		return func() {}
	}
	taskTransactionFileMu.Lock()
	return taskTransactionFileMu.Unlock
}

func emptyTaskTransactionDocument() taskTransactionDocument {
	return taskTransactionDocument{SchemaVersion: 1, Order: []string{}, Records: map[string]taskTransactionRecord{}}
}

func cloneTaskTransactionDocument(source taskTransactionDocument) taskTransactionDocument {
	copy := emptyTaskTransactionDocument()
	copy.Order = append(copy.Order, source.Order...)
	for key, record := range source.Records {
		copy.Records[key] = record
	}
	return copy
}

func publicTaskTransaction(record taskTransactionRecord) *models.TaskTransaction {
	return &models.TaskTransaction{Task: record.Task, IdempotencyKey: record.IdempotencyKey, Stage: record.Stage, Status: record.Status, RecoveryAction: record.RecoveryAction}
}

func rejectedTaskTransaction(task, key string) *models.TaskTransaction {
	return &models.TaskTransaction{Task: task, IdempotencyKey: key, Stage: "validate", Status: "rejected"}
}
