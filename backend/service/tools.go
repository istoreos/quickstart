package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/modules/raid/inventory"
	"github.com/linkease/quick-start/istore-backend/utils"
)

func Raid1Tool(DevicePaths []string) error {
	err := createRaid(DevicePaths)
	if err != nil {
		return err
	}
	return partitionMd(context.TODO())
}
func RaidFixAndMount() error {
	ctx := context.Background()
	_, err := RaidAutoFix(ctx)
	if err != nil {
		return err
	}
	err = mountFixedRaid(ctx)
	if err != nil {
		return err
	}
	return nil
}

func createRaid(DevicePaths []string) error {
	ctx := context.Background()
	level := "1"
	if len(DevicePaths) < 2 {
		return errors.New("没有足够的成员设备")
	}
	// check if already mount
	disks, err := LoadDevices(ctx)
	if err != nil {
		return err
	}
	for _, disk := range disks {
		for _, dPath := range DevicePaths {
			if "/dev/"+disk.Name == dPath {
				UnMount(disk.MountPoint)
			}
		}

	}

	diskPartPaths := []string{}
	//清除磁盘分区，建立全盘分区
	for _, v := range DevicePaths {
		match := matchStringOnce(v, `/dev/(\w+)`)
		disk, err := get_disk_info(match[1])
		if err != nil {
			return err
		}
		for _, part := range disk.Childrens {
			if strings.HasPrefix(part.MountPoint, "Raid Member:") {
				//return errors.New("already raid member of" + part.MountPoint)
				return nil
			}
			if len(part.MountPoint) > 0 && part.MountPoint != "" {
				err = Unmount(part.MountPoint)
				if err != nil {
					return err
				}
			}
		}
		err = Erase(disk.Path)
		if err != nil {
			return err
		}
		err = makeRaidPart(disk.Path)
		if err != nil {
			return err
		}
		diskPartPaths = append(diskPartPaths, inventory.DiskToPart(disk.Path, "1"))
	}
	time.Sleep(time.Second)

	deviceCount := len(diskPartPaths)
	devicePathStr := strings.Join(diskPartPaths, " ")

	idx := findFreeMd(0)
	if idx == -1 {
		return errors.New("生成raid路径失败")
	}
	rname := fmt.Sprintf("/dev/md%v", idx)

	cmdStr := fmt.Sprintf("mdadm -C %v --run --quiet --assume-clean --homehost=any -n %v -l %v %v", rname, deviceCount, level, devicePathStr)
	cmdList := []string{
		cmdStr,
	}

	err = utils.BatchRun(ctx, cmdList, 0)
	if err != nil {
		utils.BatchOutputCmd(ctx, fmt.Sprintf("rm -f %v", rname), 0)
		return errors.New("raid创建失败,请重试" + " " + cmdStr)
	}

	//添加mdadm配置,系统启动自动组建raid
	err = gen_mdadm_config()
	if err != nil {
		return err
	}

	return nil
}

func partitionMd(ctx context.Context) error {
	rname := "/dev/md1"
	time.Sleep(time.Second * 2)
	//check if already mount
	disks, err := LoadDevices(ctx)
	if err != nil {
		return err
	}
	fmt.Println("rname: ", rname)
	var mountpoint string
	for _, disk := range disks {
		if "/dev/"+disk.Name == rname {
			mountpoint = disk.MountPoint
		}
	}
	fmt.Println("mountpoint: ", mountpoint)
	moPoint := "/mnt/data_md1"
	if mountpoint == moPoint {
		// Already mount
		return nil
	}
	if mountpoint != "" {
		UnMount(mountpoint)
	}
	err = Ext4Partition(rname)
	if err != nil {
		return err
	}
	// mount
	_ = mkDlDir(ctx, moPoint)
	err = Mount(rname, moPoint)
	if err != nil {
		return err
	}
	//get uuid again, because ext4parted may change uuid
	disks, err = LoadDevices(ctx)
	if err != nil {
		return err
	}
	fmt.Println("rname: ", rname)
	uuid := ""
	for _, disk := range disks {
		if "/dev/"+disk.Name == rname {
			uuid = disk.UUID
		}
	}

	if mounts, ok := uci.GetSections("fstab", "mount"); ok {

		index := len(mounts)
		target := fmt.Sprintf("@mount[%v]", index)

		ucicmdList := []string{
			"add fstab mount ",
			fmt.Sprintf("set fstab.%v=%v", target, "mount"),
			fmt.Sprintf("set fstab.%v.target=%v", target, moPoint),
			fmt.Sprintf("set fstab.%v.enabled='1'", target),
			fmt.Sprintf("set fstab.%v.uuid=%v", target, uuid),
			"commit fstab",
		}
		reloadCmd := "/etc/init.d/fstab reload"
		err = utils.UCIBatchRun(ctx, ucicmdList, reloadCmd, 0)
		if err != nil {
			return err
		}

	}
	return nil
}

func mountFixedRaid(ctx context.Context) error {
	rname := "/dev/md1"
	time.Sleep(time.Second * 2)
	//check if already mount
	disks, err := LoadDevices(ctx)
	if err != nil {
		return err
	}
	fmt.Println("rname: ", rname)
	var mountpoint string
	for _, disk := range disks {
		if "/dev/"+disk.Name == rname {
			mountpoint = disk.MountPoint
		}
	}
	fmt.Println("mountpoint: ", mountpoint)
	moPoint := "/mnt/data_md1"
	if mountpoint == moPoint {
		// Already mount
		return nil
	}

	// mount
	_ = mkDlDir(ctx, moPoint)
	err = Mount(rname, moPoint)
	if err != nil {
		return err
	}
	//get uuid again, because ext4parted may change uuid
	disks, err = LoadDevices(ctx)
	if err != nil {
		return err
	}
	fmt.Println("rname: ", rname)
	uuid := ""
	for _, disk := range disks {
		if "/dev/"+disk.Name == rname {
			uuid = disk.UUID
		}
	}

	if mounts, ok := uci.GetSections("fstab", "mount"); ok {

		index := len(mounts)
		target := fmt.Sprintf("@mount[%v]", index)

		ucicmdList := []string{
			"add fstab mount ",
			fmt.Sprintf("set fstab.%v=%v", target, "mount"),
			fmt.Sprintf("set fstab.%v.target=%v", target, moPoint),
			fmt.Sprintf("set fstab.%v.enabled='1'", target),
			fmt.Sprintf("set fstab.%v.uuid=%v", target, uuid),
			"commit fstab",
		}
		reloadCmd := "/etc/init.d/fstab reload"
		err = utils.UCIBatchRun(ctx, ucicmdList, reloadCmd, 0)
		if err != nil {
			return err
		}

	}
	return nil
}

func LoadDevices(ctx context.Context) ([]BlockDevice, error) {
	ret, err := utils.BatchOutputCmd(ctx, "lsblk -fs --json", 0)
	if err != nil {
		return nil, err
	}

	var bd BlockDevices
	err = json.Unmarshal(ret, &bd)
	if err != nil {
		return nil, err
	}

	return bd.BlockDevices, nil
}

type BlockDevices struct {
	BlockDevices []BlockDevice `json:"blockdevices"`
}

type BlockDevice struct {
	Name       string         `json:"name"`
	DiskSize   int64          `json:"size"`
	DiskType   string         `json:"type"`
	FsType     string         `json:"fstype"`
	FsAvail    string         `json:"fsavail"`
	FsUse      string         `json:"fsuse%"`
	UUID       string         `json:"uuid"`
	MountPoint string         `json:"mountpoint"`
	Children   []*BlockDevice `json:"children"`
}
