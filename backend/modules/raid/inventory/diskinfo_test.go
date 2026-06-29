package inventory

import "testing"

const partedSDA = `BYT;
/dev/sda:2097152s:scsi:512:512:gpt:ATA Test Disk:;
1:2048s:102399s:100352s:ext4:rootfs:boot;
2:102400s:204799s:102400s:ext4:data:raid;
1:204800s:409599s:204800s:free;
`

func TestBuildDiskInfoFromPartedBuildsDiskAndPartitions(t *testing.T) {
	t.Parallel()

	disk := BuildDiskInfoFromParted("sda", true, partedSDA)

	if disk.Path != "/dev/sda" || disk.Name != "sda" {
		t.Fatalf("unexpected disk identity: %#v", disk)
	}
	if disk.PartLabelType != "GPT" || disk.VenderModel != "ATA Test Disk" || disk.TranName != "scsi" {
		t.Fatalf("unexpected disk metadata: %#v", disk)
	}
	if disk.SizeInt != "1073741824" || disk.Size != "1.0 GiB" {
		t.Fatalf("unexpected disk size: SizeInt=%q Size=%q", disk.SizeInt, disk.Size)
	}

	if len(disk.Childrens) != 3 {
		t.Fatalf("len(children) = %d, want 3", len(disk.Childrens))
	}
	root := disk.Childrens[0]
	if root.Name != "sda1" || root.Number != 1 || root.Filesystem != "ext4" || root.IsRaidOn {
		t.Fatalf("unexpected root partition: %#v", root)
	}
	if root.SecStart != 2048 || root.SecEnd != 102399 || root.SizeInt != "51380224" || root.Total != "49.0 MiB" {
		t.Fatalf("unexpected root partition size/range: %#v", root)
	}

	data := disk.Childrens[1]
	if data.Name != "sda2" || !data.IsRaidOn {
		t.Fatalf("unexpected raid partition: %#v", data)
	}

	free := disk.Childrens[2]
	if free.Name != "" || free.Filesystem != "Free Space" || free.Number != 0 {
		t.Fatalf("unexpected free partition: %#v", free)
	}
}

func TestBuildDiskInfoFromPartedSkipsFreeSpaceWhenRequested(t *testing.T) {
	t.Parallel()

	disk := BuildDiskInfoFromParted("sda", false, partedSDA)

	if len(disk.Childrens) != 2 {
		t.Fatalf("len(children) = %d, want 2", len(disk.Childrens))
	}
	for _, part := range disk.Childrens {
		if part.Filesystem == "Free Space" {
			t.Fatalf("did not expect free space partition: %#v", part)
		}
	}
}

func TestBuildDiskInfoFromPartedHandlesLegacyPartitionTablesAndNames(t *testing.T) {
	t.Parallel()

	msdos := `BYT;
/dev/mmcblk0:409600s:mmc:512:512:msdos:MMC Card:;
1:2048s:409599s:407552s:ext4:data:;
`
	disk := BuildDiskInfoFromParted("mmcblk0", false, msdos)
	if disk.PartLabelType != "MBR" {
		t.Fatalf("PartLabelType = %q, want MBR", disk.PartLabelType)
	}
	if len(disk.Childrens) != 1 || disk.Childrens[0].Name != "mmcblk0p1" {
		t.Fatalf("unexpected mmc partition naming: %#v", disk.Childrens)
	}

	nvme := `BYT;
/dev/nvme0n1:409600s:nvme:512:512:gpt:NVMe Disk:;
1:2048s:409599s:407552s::data:;
`
	disk = BuildDiskInfoFromParted("nvme0n1", false, nvme)
	if len(disk.Childrens) != 1 || disk.Childrens[0].Name != "nvme0n1p1" || disk.Childrens[0].Filesystem != "No FileSystem" {
		t.Fatalf("unexpected nvme partition: %#v", disk.Childrens)
	}
}

func TestBuildDiskInfoFromPartedHandlesLoopPartitionName(t *testing.T) {
	t.Parallel()

	loop := `BYT;
/dev/loop0:102400s:file:512:512:loop:Loop Device:;
1:0s:102399s:102400s:squashfs:rootfs:;
`
	disk := BuildDiskInfoFromParted("loop0", false, loop)

	if disk.PartLabelType != "LOOP" {
		t.Fatalf("PartLabelType = %q, want LOOP", disk.PartLabelType)
	}
	if len(disk.Childrens) != 1 || disk.Childrens[0].Name != "loop0" || disk.Childrens[0].Number != 0 {
		t.Fatalf("unexpected loop partition: %#v", disk.Childrens)
	}
}
