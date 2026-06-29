package diskinventory

import "testing"

func TestParseLSBLKDisksMapsDiskAndChildren(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"blockdevices": [
			{
				"name": "sda",
				"path": "/dev/sda",
				"pttype": "dos",
				"tran": "usb",
				"type": "disk",
				"size": 2000000000,
				"vendor": " ATA ",
				"model": " SSD ",
				"serial": " 123 ",
				"children": [
					{
						"name": "sda1",
						"path": "/dev/sda1",
						"uuid": "uuid-1",
						"ro": true,
						"size": 1000,
						"fstype": "ext4",
						"fssize": 1000,
						"fsused": "250",
						"fsuse%": "25%",
						"mountpoint": "/mnt/data",
						"label": "data"
					},
					{
						"name": "sda2",
						"path": "/dev/sda2",
						"size": 2000,
						"fssize": null,
						"fsused": null
					}
				]
			}
		]
	}`)

	disks, err := ParseLSBLKDisks(raw)
	if err != nil {
		t.Fatalf("ParseLSBLKDisks returned error: %v", err)
	}
	if len(disks) != 1 {
		t.Fatalf("expected one disk, got %d", len(disks))
	}

	root := disks[0].Root
	if root.Name != "sda" || root.Path != "/dev/sda" || root.PType != "MBR" || root.TranName != "usb" || root.Type != "disk" {
		t.Fatalf("unexpected root fields: %#v", root)
	}
	if root.DisplayName != "ATA SSD 123" {
		t.Fatalf("DisplayName = %q, want ATA SSD 123", root.DisplayName)
	}
	if root.SizeIntStr != "2000000000" || root.SizeStr != "1.9 GiB" {
		t.Fatalf("unexpected root size fields: %#v", root)
	}

	children := disks[0].Children
	if len(children) != 2 {
		t.Fatalf("expected two children, got %d", len(children))
	}
	if children[0].Name != "sda1" || children[0].Mountpoint != "/mnt/data" || children[0].UUID != "uuid-1" || !children[0].ReadOnly {
		t.Fatalf("unexpected first child identity fields: %#v", children[0])
	}
	if children[0].SizeInt != 1000 || children[0].FSType != "ext4" || children[0].Fssize != 1000 || children[0].Fsused != 250 || children[0].FsusedPercent != "25%" {
		t.Fatalf("unexpected first child size fields: %#v", children[0])
	}
	if children[0].Path != "/dev/sda1" || children[0].Label != "data" || children[0].Type != "disk" {
		t.Fatalf("unexpected first child path/type fields: %#v", children[0])
	}
	if children[1].Fssize != 0 || children[1].Fsused != 0 {
		t.Fatalf("nil fssize/fsused should map to zero, got %#v", children[1])
	}
}

func TestParseLSBLKDisksSkipsSwapAndBuildsLoopChild(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"blockdevices": [
			{
				"name": "zram0",
				"mountpoint": "[SWAP]",
				"fstype": "swap",
				"size": 1048576
			},
			{
				"name": "loop0",
				"path": "/dev/loop0",
				"mountpoint": "/rom",
				"fstype": "squashfs",
				"size": 67108864,
				"fssize": 67108864,
				"fsused": 33554432,
				"type": "loop",
				"serial": "ignored"
			}
		]
	}`)

	disks, err := ParseLSBLKDisks(raw)
	if err != nil {
		t.Fatalf("ParseLSBLKDisks returned error: %v", err)
	}
	if len(disks) != 1 {
		t.Fatalf("expected only loop disk after swap skip, got %d", len(disks))
	}

	disk := disks[0]
	if disk.Root.Name != "loop0" || disk.Root.PType != "LOOP" || disk.Root.DisplayName != "" {
		t.Fatalf("unexpected loop root: %#v", disk.Root)
	}
	if len(disk.Children) != 1 {
		t.Fatalf("expected loop disk to produce one child, got %d", len(disk.Children))
	}
	child := disk.Children[0]
	if child.Name != "loop0" || child.Mountpoint != "/rom" || child.FSType != "squashfs" || child.SizeInt != 67108864 {
		t.Fatalf("unexpected loop child: %#v", child)
	}
	if child.Fssize != 67108864 || child.Fsused != 33554432 || child.Type != "loop" {
		t.Fatalf("unexpected loop child usage/type: %#v", child)
	}
}

func TestParseLSBLKDisksReturnsJSONErrors(t *testing.T) {
	t.Parallel()

	if _, err := ParseLSBLKDisks([]byte(`{`)); err == nil {
		t.Fatal("expected JSON error")
	}
}
