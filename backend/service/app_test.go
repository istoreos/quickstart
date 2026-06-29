package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadApplistFromPath(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.json")
	validJSON := `{
		"name": "demo-app",
		"title": "Demo App",
		"arch": ["all"],
		"depends": ["base"],
		"tags": ["demo"],
		"version": "1.0.0"
	}`
	if err := os.WriteFile(validPath, []byte(validJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "invalid.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	apps, err := readApplistFromPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) == 0 {
		t.Fatal("expected at least one parsed app")
	}
	if len(apps) != 1 {
		t.Fatalf("expected invalid JSON file to be ignored, got %d apps", len(apps))
	}
	if apps[0].Name != "demo-app" {
		t.Fatalf("expected parsed app name demo-app, got %q", apps[0].Name)
	}
	if apps[0].Time == 0 {
		t.Fatal("expected file mod time to be set on parsed app")
	}
}
