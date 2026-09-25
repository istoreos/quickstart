package service

import (
	"path/filepath"
	"testing"
)

func useManufData(t *testing.T, data map[int]map[uint64]string) {
	t.Helper()
	<-initDone
	original := d
	originalPrefixes := manufPrefixLengths
	d = data
	finalizeManufIndex()
	t.Cleanup(func() {
		d = original
		manufPrefixLengths = originalPrefixes
	})
}

func TestLoadManufDataAllowsMissingFile(t *testing.T) {
	<-initDone
	original := d
	originalPrefixes := manufPrefixLengths
	t.Cleanup(func() {
		d = original
		manufPrefixLengths = originalPrefixes
	})
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

func TestGomanufSearchUsesMostSpecificPrefix(t *testing.T) {
	useManufData(t, map[int]map[uint64]string{
		24: {0x001BC5000000: "Broad vendor"},
		36: {0x001BC5001000: "Specific vendor"},
	})

	for range 100 {
		if got := GomanufSearch("00:1b:c5:00:1a:bc"); got != "Specific vendor" {
			t.Fatalf("expected longest-prefix match, got %q", got)
		}
	}
}

func TestGomanufSearchRejectsNonGlobalAndInvalidMACs(t *testing.T) {
	useManufData(t, map[int]map[uint64]string{
		24: {
			0x020000000000: "Locally administered",
			0x010000000000: "Multicast",
		},
	})

	for _, mac := range []string{"02:00:00:00:00:01", "01:00:00:00:00:01", "not-a-mac", ""} {
		if got := GomanufSearch(mac); got != "" {
			t.Errorf("expected %q to be rejected, got %q", mac, got)
		}
	}
}

func TestParseUsesObservedPrefixLengthWhenMaskIsOmitted(t *testing.T) {
	useManufData(t, make(map[int]map[uint64]string))

	parse("00:1B", "Two-byte vendor")

	entries, ok := d[16]
	if !ok {
		t.Fatalf("expected an unmasked two-byte prefix to be stored as /16, got keys %#v", d)
	}
	if got := entries[0x001B00000000]; got != "Two-byte vendor" {
		t.Fatalf("expected parsed vendor, got %q", got)
	}
}
