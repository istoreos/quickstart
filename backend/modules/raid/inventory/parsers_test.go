package inventory

import "testing"

func TestParsePartedInfoMapsDelimitedFieldsToKeys(t *testing.T) {
	t.Parallel()

	got := ParsePartedInfo(
		[]string{"path", "size", "type", "logic_sec"},
		"/dev/sda:2097152s:scsi:512:512:gpt:ATA Disk:;",
	)

	if got["path"] != "/dev/sda" || got["size"] != "2097152s" || got["type"] != "scsi" || got["logic_sec"] != "512" {
		t.Fatalf("unexpected parsed fields: %#v", got)
	}
	if _, ok := got["extra"]; ok {
		t.Fatalf("did not expect unknown keys: %#v", got)
	}
}

func TestDiskToPartHandlesNamesEndingInDigits(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"/dev/sda":     "/dev/sda1",
		"/dev/vda":     "/dev/vda1",
		"/dev/mmcblk0": "/dev/mmcblk0p1",
		"/dev/nvme0n1": "/dev/nvme0n1p1",
		"/dev/md0":     "/dev/md0p1",
	}
	for device, expected := range cases {
		if got := DiskToPart(device, "1"); got != expected {
			t.Fatalf("DiskToPart(%q) = %q, want %q", device, got, expected)
		}
	}
}

func TestParsePartitionUsageExtractsDFColumns(t *testing.T) {
	t.Parallel()

	used, free, usage := ParsePartitionUsage("Filesystem 1K-blocks Used Available Use% Mounted on\n/dev/sda1 100000 25000 75000 25% /mnt/data\n")
	if used != "25000" || free != "75000" || usage != "25" {
		t.Fatalf("unexpected df usage: used=%q free=%q usage=%q", used, free, usage)
	}

	used, free, usage = ParsePartitionUsage("not df output")
	if used != "" || free != "" || usage != "" {
		t.Fatalf("expected empty values for unmatched df output, got used=%q free=%q usage=%q", used, free, usage)
	}
}

func TestParseMountPointExtractsMountedPath(t *testing.T) {
	t.Parallel()

	mountOutput := "/dev/sda1 on /mnt/data type ext4 (rw,relatime)\n/dev/sdb1 on /mnt/backup type ext4 (rw)\n"
	if got := ParseMountPoint(mountOutput, "sdb1"); got != "/mnt/backup" {
		t.Fatalf("unexpected mount point: %q", got)
	}
	if got := ParseMountPoint(mountOutput, "sdc1"); got != "" {
		t.Fatalf("expected empty mount point, got %q", got)
	}
}

func TestParseRaidMemberPreservesLegacyMDStatMatching(t *testing.T) {
	t.Parallel()

	mdstat := "md0: active raid1 sda1[0] sdb1[1]\nmd1: active raid1 nvme0n1p1[0]\n"
	if got := ParseRaidMember(mdstat, "sdb1"); got != "Raid Member: md0" {
		t.Fatalf("unexpected raid member: %q", got)
	}
	if got := ParseRaidMember(mdstat, "nvme0n1p1"); got != "Raid Member: md1" {
		t.Fatalf("unexpected raid member: %q", got)
	}
	if got := ParseRaidMember(mdstat, "sdc1"); got != "" {
		t.Fatalf("expected non-member to be empty, got %q", got)
	}
}

func TestParseMDDetailExtractsKeyValueLinesForMDDevices(t *testing.T) {
	t.Parallel()

	detail := ParseMDDetail("/dev/md0", "    State : clean\nActive Devices : 2\nBad Line\n")
	if detail["State"] != "clean" || detail["Active Devices"] != "2" {
		t.Fatalf("unexpected md detail: %#v", detail)
	}

	if detail := ParseMDDetail("/dev/sda", "State : clean\n"); len(detail) != 0 {
		t.Fatalf("expected non-md path to be ignored, got %#v", detail)
	}
}
