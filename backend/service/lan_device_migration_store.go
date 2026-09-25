package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/istoreos/quickstart/backend/models"
)

const defaultLanDeviceMigrationReceiptPath = "/etc/quickstart/lan-device-model-v1.json"

type systemLanDeviceMigrationStore struct {
	rules       *NetworkRulesModule
	receiptPath string
}

func (store *systemLanDeviceMigrationStore) Read(ctx context.Context) (lanDeviceMigrationSnapshot, error) {
	snapshot := lanDeviceMigrationSnapshot{Items: []*models.LanDeviceMigrationItem{}, SourceVersions: map[string]string{}}
	if store.rules == nil {
		return snapshot, errors.New("network rules module is unavailable")
	}
	response, err := store.rules.List(ctx)
	if err != nil {
		return snapshot, err
	}
	if response == nil || response.Result == nil {
		return snapshot, errors.New("network rules are unavailable")
	}
	snapshot.SourceVersions["network_rules"] = response.Result.Version
	for _, rule := range response.Result.Rules {
		if rule == nil {
			continue
		}
		disposition := "adopt"
		reason := ""
		if rule.Status == "unsupported" {
			disposition, reason = "unresolved", "legacy route target cannot be represented safely"
		}
		target := fmt.Sprintf("%s:%s", rule.Kind, rule.DeviceID)
		if rule.DeviceID == "" {
			target = fmt.Sprintf("%s:orphan:%s", rule.Kind, rule.ID)
		}
		snapshot.Items = append(snapshot.Items, &models.LanDeviceMigrationItem{
			ID: rule.ID, Kind: rule.Kind, Source: "network_rules", Target: target,
			Disposition: disposition, Summary: rule.Summary, Reason: reason,
		})
	}
	for _, source := range []string{"/etc/config/dhcp", "/etc/config/eqos", "/etc/config/firewall", "/etc/config/floatip", defaultDeviceProfilePath} {
		snapshot.SourceVersions[source] = hashOptionalMigrationFile(source)
	}
	receipt, exists, err := readMigrationReceipt(store.receiptPath)
	if err != nil {
		return snapshot, err
	}
	if exists && receipt.ModelVersion == lanDeviceModelVersion && receipt.Mode == "adopt_in_place" {
		snapshot.AppliedVersion = receipt.SourceVersion
	}
	return snapshot, nil
}

func (store *systemLanDeviceMigrationStore) SnapshotReceipt(context.Context) ([]byte, bool, error) {
	data, err := os.ReadFile(store.receiptPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return data, err == nil, err
}

func (store *systemLanDeviceMigrationStore) Commit(_ context.Context, receipt lanDeviceMigrationReceipt) error {
	data, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	return persistMigrationReceipt(store.receiptPath, data)
}

func (store *systemLanDeviceMigrationStore) RestoreReceipt(_ context.Context, data []byte, existed bool) error {
	if !existed {
		if err := os.Remove(store.receiptPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return persistMigrationReceipt(store.receiptPath, data)
}

func readMigrationReceipt(path string) (lanDeviceMigrationReceipt, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return lanDeviceMigrationReceipt{}, false, nil
	}
	if err != nil {
		return lanDeviceMigrationReceipt{}, false, err
	}
	var receipt lanDeviceMigrationReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return lanDeviceMigrationReceipt{}, false, err
	}
	return receipt, true, nil
}

func hashOptionalMigrationFile(path string) string {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "missing"
	}
	if err != nil {
		return "error:" + err.Error()
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func persistMigrationReceipt(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".lan-device-model-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
