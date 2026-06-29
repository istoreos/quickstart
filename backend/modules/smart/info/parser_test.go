package info

import (
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestParseSmartctlInfoParsesSATAOutput(t *testing.T) {
	base := models.SmartInfo{
		Name:    "sda",
		Path:    "/dev/sda",
		Model:   "Demo SATA",
		SizeStr: "1 TB",
	}
	out := `=== START OF INFORMATION SECTION ===
Serial Number:    SATA123
Rotation Rate:    7200 rpm
SATA Version is:  SATA 3.3
Power mode is:    ACTIVE or IDLE
=== START OF READ SMART DATA SECTION ===
SMART overall-health self-assessment test result: PASSED
194 Temperature_Celsius POSRCK 064 044 000 000 36
`

	got := ParseSmartctlInfo(base, out)
	want := &models.SmartInfo{
		Name:     "sda",
		Path:     "/dev/sda",
		Model:    "Demo SATA",
		SizeStr:  "1 TB",
		Serial:   "SATA123",
		RotaRate: "7200 rpm",
		SataVer:  "SATA 3.3",
		Status:   "ACTIVE",
		Health:   "PASSED",
		Temp:     "36°C",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected SATA info\nwant: %#v\n got: %#v", want, got)
	}
}

func TestParseSmartctlInfoParsesNVMeOutput(t *testing.T) {
	base := models.SmartInfo{Name: "nvme0n1", Path: "/dev/nvme0n1", Model: "Demo NVMe", SizeStr: "2 TB"}
	out := `=== START OF INFORMATION SECTION ===
Serial Number:    NVME123
NVMe Version:     1.4
Power mode is:    ACTIVE
=== START OF SMART DATA SECTION ===
SMART overall-health self-assessment test result: PASSED
Temperature:                        42 Celsius
`

	got := ParseSmartctlInfo(base, out)
	want := &models.SmartInfo{
		Name:    "nvme0n1",
		Path:    "/dev/nvme0n1",
		Model:   "Demo NVMe",
		SizeStr: "2 TB",
		Serial:  "NVME123",
		NvmeVer: "1.4",
		Status:  "ACTIVE",
		Health:  "PASSED",
		Temp:    "42°C",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected NVMe info\nwant: %#v\n got: %#v", want, got)
	}
}

func TestParseSmartctlInfoPreservesBaseWhenOutputIsEmpty(t *testing.T) {
	base := models.SmartInfo{Name: "sda", Path: "/dev/sda", Model: "Demo", SizeStr: "1 TB"}

	got := ParseSmartctlInfo(base, "")
	if !reflect.DeepEqual(got, &base) {
		t.Fatalf("expected base info to be preserved, got %#v", got)
	}
}
