package homeconfig

import (
	"encoding/json"
	"os"
	"testing"
)

func TestExternalLayoutMigrationAndMissingPlugin(t *testing.T) {
	raw, e := os.ReadFile("../../../docs/contracts/home-layout-v4.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixture struct {
		Legacy   json.RawMessage `json:"legacy"`
		Migrated json.RawMessage `json:"migrated"`
	}
	json.Unmarshal(raw, &fixture)
	migrated, e := Canonical("layout", fixture.Legacy)
	if e != nil {
		t.Fatal(e)
	}
	expected, e := Canonical("layout", fixture.Migrated)
	if e != nil || string(migrated) != string(expected) {
		t.Fatal("migration changed intent")
	}
	raw, e = os.ReadFile("../../../docs/contracts/fastnet-card.json")
	if e != nil {
		t.Fatal(e)
	}
	var f struct {
		Layout json.RawMessage `json:"layoutV4"`
	}
	json.Unmarshal(raw, &f)
	store := NewStore(t.TempDir())
	saved, e := store.Save("layout", "missing", f.Layout)
	if e != nil {
		t.Fatal(e)
	}
	read, e := NewStore(store.root).Read("layout")
	if e != nil || read.Version != saved.Version || string(read.Value) != string(saved.Value) {
		t.Fatal("missing plugin lost on read")
	}
	if _, e = store.Save("layout", "missing", fixture.Migrated); e != ErrConflict {
		t.Fatal("stale version overwrite", e)
	}
	var layout Layout
	json.Unmarshal(f.Layout, &layout)
	layout.WidgetPreferences["plugin.fastnet"] = Preference{Mode: "iframe"}
	bad, _ := json.Marshal(layout)
	if _, e = Canonical("layout", bad); e != ErrInvalid {
		t.Fatal("invalid external preference")
	}
}
