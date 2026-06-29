package partedinfo

import (
	"context"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

const partedSDA = `BYT;
/dev/sda:2097152s:scsi:512:512:gpt:ATA Test Disk:;
1:2048s:102399s:100352s:ext4:rootfs:boot;
2:102400s:204799s:102400s:ext4:data:raid;
3:204800s:409599s:204800s:free;
`

type fakeStore struct {
	rootPaths        []string
	dockerDevicePath string
	partedOutput     string
	mountPoints      map[string]string
	uuids            map[string]string
	usage            map[string]partitionUsage
	markerCalls      []markerCall
}

type partitionUsage struct {
	usedKB string
	usage  string
}

type markerCall struct {
	diskName         string
	partitionName    string
	rootPaths        []string
	dockerDevicePath string
}

func (store *fakeStore) RootPaths(ctx context.Context) []string {
	return store.rootPaths
}

func (store *fakeStore) DockerDevicePath(ctx context.Context) string {
	return store.dockerDevicePath
}

func (store *fakeStore) Parted(ctx context.Context, device string) string {
	return store.partedOutput
}

func (store *fakeStore) MountPoint(ctx context.Context, partitionName string) string {
	return store.mountPoints[partitionName]
}

func (store *fakeStore) UUID(ctx context.Context, partitionPath string) string {
	return store.uuids[partitionPath]
}

func (store *fakeStore) PartitionUsage(ctx context.Context, partitionName string) (string, string) {
	usage := store.usage[partitionName]
	return usage.usedKB, usage.usage
}

func (store *fakeStore) MarkMountedPartition(ctx context.Context, disk *models.NasDiskInfo, partition *models.PartitionInfo, rootPaths []string, dockerDevicePath string) {
	store.markerCalls = append(store.markerCalls, markerCall{
		diskName:         disk.Name,
		partitionName:    partition.Name,
		rootPaths:        append([]string(nil), rootPaths...),
		dockerDevicePath: dockerDevicePath,
	})
}

func TestReadEnrichesMountedPartitionsAndSkipsRom(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		rootPaths:        []string{"/dev/root"},
		dockerDevicePath: "/dev/sda1",
		partedOutput:     partedSDA,
		mountPoints: map[string]string{
			"sda1": "/mnt/data",
			"sda2": "/rom",
		},
		uuids: map[string]string{
			"/dev/sda1": "uuid-1",
			"/dev/sda2": "uuid-2",
		},
		usage: map[string]partitionUsage{
			"sda1": {usedKB: "100", usage: "10"},
			"sda2": {usedKB: "200", usage: "20"},
		},
	}

	disk, err := NewService(store).Read(context.Background(), "sda", false)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if disk.Name != "sda" || disk.Path != "/dev/sda" {
		t.Fatalf("unexpected disk identity: %#v", disk)
	}
	if len(disk.Childrens) != 1 {
		t.Fatalf("expected /rom partition to be skipped, got %#v", disk.Childrens)
	}

	part := disk.Childrens[0]
	if part.Name != "sda1" || part.Path != "/dev/sda1" || part.MountPoint != "/mnt/data" || part.UUID != "uuid-1" {
		t.Fatalf("unexpected enriched partition: %#v", part)
	}
	if part.Usage != 10 || part.Used == "" {
		t.Fatalf("expected mounted partition usage to be populated: %#v", part)
	}
	wantCalls := []markerCall{{
		diskName:         "sda",
		partitionName:    "sda1",
		rootPaths:        []string{"/dev/root"},
		dockerDevicePath: "/dev/sda1",
	}, {
		diskName:         "sda",
		partitionName:    "sda2",
		rootPaths:        []string{"/dev/root"},
		dockerDevicePath: "/dev/sda1",
	}}
	if !reflect.DeepEqual(store.markerCalls, wantCalls) {
		t.Fatalf("unexpected marker calls:\nwant=%#v\ngot=%#v", wantCalls, store.markerCalls)
	}
}

func TestReadIncludesFreeSpaceWhenRequested(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		partedOutput: partedSDA,
		mountPoints:  map[string]string{},
		uuids:        map[string]string{},
		usage:        map[string]partitionUsage{},
	}

	disk, err := NewService(store).Read(context.Background(), "sda", true)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if len(disk.Childrens) != 3 {
		t.Fatalf("expected free space to be included, got %#v", disk.Childrens)
	}
	if disk.Childrens[2].Filesystem != "Free Space" {
		t.Fatalf("expected free space partition, got %#v", disk.Childrens[2])
	}
	if len(store.markerCalls) != 0 {
		t.Fatalf("did not expect marker calls for unmounted partitions, got %#v", store.markerCalls)
	}
}
