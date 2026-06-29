package service

import (
	"path/filepath"
	"testing"
)

func TestLoadManufDataAllowsMissingFile(t *testing.T) {
	d = nil

	missingFile := filepath.Join(t.TempDir(), "missing-manuf")
	err := loadManufData(missingFile)
	if err != nil {
		t.Fatalf("expected missing manuf file to be ignored, got %v", err)
	}
	if d == nil {
		t.Fatal("expected manuf map to be initialized")
	}
	if len(d) != 0 {
		t.Fatalf("expected empty manuf map, got %d entries", len(d))
	}
}
