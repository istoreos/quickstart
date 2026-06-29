package diskstatus

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/nas/diskinventory"
)

type fakeInventoryReader struct {
	disks []*diskinventory.DiskInfo
	err   error
}

func (reader fakeInventoryReader) List(ctx context.Context) ([]*diskinventory.DiskInfo, error) {
	return reader.disks, reader.err
}

type fakePartitionMarker struct {
	systemPath string
	calls      int
}

func (marker *fakePartitionMarker) Mark(ctx context.Context, disk *models.NasDiskInfo, partition *models.PartitionInfo) {
	marker.calls++
	if partition.Path == marker.systemPath {
		partition.IsSystemRoot = true
		disk.IsSystemRoot = true
	}
}

type fakeRAIDReader struct {
	members map[string]string
}

func (reader fakeRAIDReader) RAIDMember(ctx context.Context, diskName string) string {
	return reader.members[diskName]
}

type fakeSMARTReader struct {
	config      *models.SmartConfigResponseResult
	health      map[string]string
	healthCalls []string
}

func (reader *fakeSMARTReader) Config(ctx context.Context) (*models.SmartConfigResponseResult, error) {
	return reader.config, nil
}

func (reader *fakeSMARTReader) Health(ctx context.Context, diskName string) (string, error) {
	reader.healthCalls = append(reader.healthCalls, diskName)
	return reader.health[diskName], nil
}

func TestBuildDiskInfoMapsDiskRootAndExternalFlag(t *testing.T) {
	t.Parallel()

	disk, ok := BuildDiskInfo(&diskinventory.DiskInfo{
		Root: diskinventory.DiskInfoRoot{
			Name:        "sda",
			Path:        "/dev/sda",
			SizeStr:     "1.9 GiB",
			SizeIntStr:  "2000000000",
			DisplayName: "ATA SSD",
			PType:       "MBR",
			Type:        "disk",
			TranName:    "usb",
		},
	})

	if !ok {
		t.Fatal("expected disk root to be included")
	}
	if disk.Name != "sda" || disk.Path != "/dev/sda" || disk.Size != "1.9 GiB" || disk.SizeInt != "2000000000" {
		t.Fatalf("unexpected disk identity fields: %#v", disk)
	}
	if disk.VenderModel != "ATA SSD" || disk.PartLabelType != "MBR" || disk.TranName != "usb" || !disk.IsExternalDisk {
		t.Fatalf("unexpected disk metadata fields: %#v", disk)
	}
}

func TestBuildDiskInfoSkipsNonDiskRoots(t *testing.T) {
	t.Parallel()

	if disk, ok := BuildDiskInfo(&diskinventory.DiskInfo{Root: diskinventory.DiskInfoRoot{Type: "loop"}}); ok || disk != nil {
		t.Fatalf("expected non-disk root to be skipped, got disk=%#v ok=%v", disk, ok)
	}
}

func TestBuildPartitionInfoMapsUsageAndFilesystemDefaults(t *testing.T) {
	t.Parallel()

	point, usage := BuildPartitionInfo(&diskinventory.DiskInfoChildren{
		Name:       "sda1",
		Path:       "/dev/sda1",
		UUID:       "uuid-1",
		Mountpoint: "/mnt/data",
		SizeInt:    1000,
		Fsused:     250,
	})

	if point.Name != "sda1" || point.Path != "/dev/sda1" || point.UUID != "uuid-1" || point.MountPoint != "/mnt/data" {
		t.Fatalf("unexpected partition identity fields: %#v", point)
	}
	if point.Filesystem != "unknown" {
		t.Fatalf("Filesystem = %q, want unknown for mounted partition without fstype", point.Filesystem)
	}
	if point.Total != "1000 B" || point.SizeInt != "1000" || point.Used != "250 B" || point.Usage != 25 {
		t.Fatalf("unexpected partition usage fields: %#v", point)
	}
	if usage.Total != 1000 || usage.Used != 250 {
		t.Fatalf("unexpected usage summary: %#v", usage)
	}

	point, _ = BuildPartitionInfo(&diskinventory.DiskInfoChildren{})
	if point.Filesystem != "No FileSystem" {
		t.Fatalf("Filesystem = %q, want No FileSystem for unmounted partition without fstype", point.Filesystem)
	}
}

func TestShouldIncludePartitionPreservesLegacyFilters(t *testing.T) {
	t.Parallel()

	regular, usage := BuildPartitionInfo(&diskinventory.DiskInfoChildren{SizeInt: 128 * 1024 * 1024})
	if !ShouldIncludePartition("data", regular, usage.Total) {
		t.Fatal("expected regular large partition to be included")
	}

	small, usage := BuildPartitionInfo(&diskinventory.DiskInfoChildren{SizeInt: 32 * 1024 * 1024})
	if ShouldIncludePartition("data", small, usage.Total) {
		t.Fatal("expected small non-system partition to be filtered")
	}

	systemSmall, usage := BuildPartitionInfo(&diskinventory.DiskInfoChildren{SizeInt: 32 * 1024 * 1024})
	systemSmall.IsSystemRoot = true
	if !ShouldIncludePartition("data", systemSmall, usage.Total) {
		t.Fatal("expected small system partition to be preserved")
	}

	rom, usage := BuildPartitionInfo(&diskinventory.DiskInfoChildren{Mountpoint: "/rom", SizeInt: 128 * 1024 * 1024})
	if ShouldIncludePartition("data", rom, usage.Total) {
		t.Fatal("expected /rom partition to be filtered")
	}

	kernel, usage := BuildPartitionInfo(&diskinventory.DiskInfoChildren{Label: "kernel", SizeInt: 128 * 1024 * 1024})
	if ShouldIncludePartition("kernel", kernel, usage.Total) {
		t.Fatal("expected kernel label partition to be filtered")
	}
}

func TestApplyDiskUsageAndBuildFreeSpacePartition(t *testing.T) {
	t.Parallel()

	disk, ok := BuildDiskInfo(&diskinventory.DiskInfo{Root: diskinventory.DiskInfoRoot{Type: "disk"}})
	if !ok {
		t.Fatal("expected disk")
	}
	ApplyDiskUsage(disk, 512, 1024)
	if disk.Used != "512 B" || disk.UsedInt != "512" || disk.Total != "1.0 KiB" || disk.Usage != 50 {
		t.Fatalf("unexpected disk usage fields: %#v", disk)
	}

	free := BuildFreeSpacePartition(3*1024*1024*1024, 1024*1024*1024)
	if free == nil {
		t.Fatal("expected free space partition when free bytes > 1GiB")
	}
	if free.Name != "Free Space" || free.Filesystem != "Free Space" || free.SizeInt != "2147483648" || free.Total != "2.0 GiB" {
		t.Fatalf("unexpected free space partition: %#v", free)
	}

	if free := BuildFreeSpacePartition(2*1024*1024*1024, 1024*1024*1024); free != nil {
		t.Fatalf("expected no free space partition at exactly 1GiB free, got %#v", free)
	}
}

func TestMarkSystemAndDockerMarksPartitionAndDiskRoots(t *testing.T) {
	t.Parallel()

	disk, ok := BuildDiskInfo(&diskinventory.DiskInfo{Root: diskinventory.DiskInfoRoot{Type: "disk"}})
	if !ok {
		t.Fatal("expected disk")
	}
	part := mustPartition(t, &diskinventory.DiskInfoChildren{Path: "/dev/sda2", Mountpoint: "/overlay", SizeInt: 128 * 1024 * 1024})

	MarkSystemAndDocker(disk, part, []string{"/dev/sda2"}, "/dev/sda2")

	if !part.IsSystemRoot || !disk.IsSystemRoot {
		t.Fatalf("expected system root flags, disk=%#v part=%#v", disk, part)
	}
	if !part.IsDockerRoot || !disk.IsDockerRoot {
		t.Fatalf("expected docker root flags, disk=%#v part=%#v", disk, part)
	}
}

func TestMarkSystemAndDockerPreservesLegacyRomLoopDockerDiskFlag(t *testing.T) {
	t.Parallel()

	disk, ok := BuildDiskInfo(&diskinventory.DiskInfo{Root: diskinventory.DiskInfoRoot{Type: "disk"}})
	if !ok {
		t.Fatal("expected disk")
	}
	part := mustPartition(t, &diskinventory.DiskInfoChildren{Path: "/dev/root", Mountpoint: "/rom", SizeInt: 128 * 1024 * 1024})

	MarkSystemAndDocker(disk, part, nil, "/dev/loop0")

	if part.IsDockerRoot {
		t.Fatalf("expected /rom overlay compatibility to leave partition docker flag false: %#v", part)
	}
	if !disk.IsDockerRoot {
		t.Fatalf("expected /rom overlay compatibility to mark disk docker root: %#v", disk)
	}
}

func TestShouldIncludeDiskPreservesLegacyDiskFilters(t *testing.T) {
	t.Parallel()

	regular := &models.NasDiskInfo{Name: "sda", SizeInt: "2000000000"}
	if !ShouldIncludeDisk(regular, 1, 2000000000, "") {
		t.Fatal("expected regular non-system disk over 1GB to be included")
	}

	systemLoopWithoutPartitions := &models.NasDiskInfo{IsSystemRoot: true, PartLabelType: "LOOP", SizeInt: "2000000000"}
	if ShouldIncludeDisk(systemLoopWithoutPartitions, 0, 2000000000, "") {
		t.Fatal("expected system LOOP disk without visible partitions to be filtered")
	}

	smallNonSystem := &models.NasDiskInfo{Name: "sdb", SizeInt: "999999999"}
	if ShouldIncludeDisk(smallNonSystem, 1, 999999999, "") {
		t.Fatal("expected non-system disk under 1GB to be filtered")
	}

	smallSystem := &models.NasDiskInfo{Name: "root", IsSystemRoot: true, SizeInt: "999999999"}
	if !ShouldIncludeDisk(smallSystem, 1, 999999999, "") {
		t.Fatal("expected system disk under 1GB to be included")
	}

	raidMember := &models.NasDiskInfo{Name: "sdc", SizeInt: "2000000000"}
	if ShouldIncludeDisk(raidMember, 1, 2000000000, "/dev/md0") {
		t.Fatal("expected RAID member disk to be filtered")
	}
}

func TestShouldCheckSMARTRequiresEnabledConfigAndMatchingDevice(t *testing.T) {
	t.Parallel()

	config := &models.SmartConfigResponseResult{
		Global: &models.SmartConfigGlobal{Enable: true},
		Devices: []*models.SmartConfigDevice{
			{DevicePath: "/dev/sda"},
			{DevicePath: "/dev/sdb"},
		},
	}

	if !ShouldCheckSMART("/dev/sda", config) {
		t.Fatal("expected matching disk to be checked when SMART is enabled")
	}
	if ShouldCheckSMART("/dev/sdc", config) {
		t.Fatal("expected unmatched disk not to be checked")
	}
	config.Global.Enable = false
	if ShouldCheckSMART("/dev/sda", config) {
		t.Fatal("expected disabled SMART config not to check disks")
	}
}

func TestApplySMARTHealthMarksWarningsForNonPassedHealth(t *testing.T) {
	t.Parallel()

	passed := &models.NasDiskInfo{}
	ApplySMARTHealth(passed, "PASSED")
	if passed.SmartWarning {
		t.Fatal("expected PASSED health not to mark warning")
	}

	failed := &models.NasDiskInfo{}
	ApplySMARTHealth(failed, "FAILED")
	if !failed.SmartWarning {
		t.Fatal("expected non-PASSED health to mark warning")
	}
}

func TestServiceBuildsDiskStatusWithMarkersAndSMART(t *testing.T) {
	t.Parallel()

	marker := &fakePartitionMarker{systemPath: "/dev/sda1"}
	smart := &fakeSMARTReader{
		config: &models.SmartConfigResponseResult{
			Global:  &models.SmartConfigGlobal{Enable: true},
			Devices: []*models.SmartConfigDevice{{DevicePath: "/dev/sda"}},
		},
		health: map[string]string{"sda": "FAILED"},
	}
	svc := NewService(
		fakeInventoryReader{disks: []*diskinventory.DiskInfo{
			{
				Root: diskinventory.DiskInfoRoot{
					Name:        "sda",
					Path:        "/dev/sda",
					Type:        "disk",
					SizeIntStr:  "3221225472",
					SizeStr:     "3.0 GiB",
					DisplayName: "ATA SSD",
				},
				Children: []*diskinventory.DiskInfoChildren{
					{
						Name:       "sda1",
						Path:       "/dev/sda1",
						Mountpoint: "/",
						FSType:     "ext4",
						SizeInt:    1024 * 1024 * 1024,
						Fsused:     512 * 1024 * 1024,
					},
				},
			},
		}},
		marker,
		fakeRAIDReader{},
		smart,
	)

	disks, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(disks) != 1 {
		t.Fatalf("expected one disk, got %d", len(disks))
	}
	disk := disks[0]
	if disk.Name != "sda" || !disk.IsSystemRoot || !disk.SmartWarning {
		t.Fatalf("unexpected disk flags: %#v", disk)
	}
	if disk.UsedInt != "536870912" || disk.Total != "1.0 GiB" || disk.Usage != 50 {
		t.Fatalf("unexpected disk usage: %#v", disk)
	}
	if len(disk.Childrens) != 2 {
		t.Fatalf("expected data partition plus free-space partition, got %#v", disk.Childrens)
	}
	if disk.Childrens[0].Path != "/dev/sda1" || !disk.Childrens[0].IsSystemRoot {
		t.Fatalf("unexpected first partition: %#v", disk.Childrens[0])
	}
	if disk.Childrens[1].Name != "Free Space" || disk.Childrens[1].SizeInt != "2147483648" {
		t.Fatalf("unexpected free-space partition: %#v", disk.Childrens[1])
	}
	if marker.calls != 1 {
		t.Fatalf("expected marker called once, got %d", marker.calls)
	}
	if len(smart.healthCalls) != 1 || smart.healthCalls[0] != "sda" {
		t.Fatalf("unexpected SMART health calls: %#v", smart.healthCalls)
	}
}

func TestServiceFiltersNonDiskSmallDiskAndRAIDMembers(t *testing.T) {
	t.Parallel()

	svc := NewService(
		fakeInventoryReader{disks: []*diskinventory.DiskInfo{
			{Root: diskinventory.DiskInfoRoot{Name: "loop0", Type: "loop", SizeIntStr: "2000000000"}},
			{Root: diskinventory.DiskInfoRoot{Name: "sdb", Type: "disk", SizeIntStr: "999999999"}},
			{Root: diskinventory.DiskInfoRoot{Name: "sdc", Type: "disk", SizeIntStr: "2000000000"}},
			{Root: diskinventory.DiskInfoRoot{Name: "sdd", Path: "/dev/sdd", Type: "disk", SizeIntStr: "2000000000"}},
		}},
		nil,
		fakeRAIDReader{members: map[string]string{"sdc": "/dev/md0"}},
		nil,
	)

	disks, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(disks) != 1 || disks[0].Name != "sdd" {
		t.Fatalf("expected only regular disk sdd, got %#v", disks)
	}
}

func TestServicePropagatesInventoryErrors(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("inventory failed")
	svc := NewService(fakeInventoryReader{err: expectedErr}, nil, nil, nil)

	if _, err := svc.List(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("expected inventory error, got %v", err)
	}
}

func mustPartition(t *testing.T, input *diskinventory.DiskInfoChildren) *models.PartitionInfo {
	t.Helper()
	part, _ := BuildPartitionInfo(input)
	return part
}
