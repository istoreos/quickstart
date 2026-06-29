package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/nas/diskstatus"
	"github.com/linkease/quick-start/istore-backend/modules/nas/partedinfo"
	"github.com/linkease/quick-start/istore-backend/modules/raid/inventory"
	"github.com/linkease/quick-start/istore-backend/modules/raid/writecommands"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type partedInfoStore struct{}

func get_disk_info(device string) (*models.NasDiskInfo, error) {
	// if !canAccessPath("/dev/" + device) {
	// 	return nil, errors.New("设备不存在" + device)
	// }
	disk, err := get_parted_info(device, false)
	// if disk.TranName == "md" {

	// }
	return disk, err
}

func get_disk_info_include_free(device string) (*models.NasDiskInfo, error) {
	// if !canAccessPath("/dev/" + device) {
	// 	return nil, errors.New("设备不存在" + device)
	// }
	disk, err := get_parted_info(device, true)
	// if disk.TranName == "md" {

	// }
	return disk, err
}

func get_boot_parts(ctx context.Context) []string {
	rootPaths := []string{}
	for _, sp := range []string{"/overlay", "/rom", "/boot", "/"} {
		rootPath, err := utils.BatchOutputCmd(ctx, "findmnt -T "+sp+"  -o SOURCE|sed -n 2p", 0)
		if err != nil {
			continue
		}
		rootPathStr := strings.Replace(string(rootPath), "\n", "", -1)
		rootPaths = append(rootPaths, rootPathStr)
	}
	return rootPaths
}

func get_docker_part(ctx context.Context) string {
	dockerDevicePathStr := "NOTFOUND"
	dockerRootPath, err := utils.BatchOutputCmd(ctx, "uci get dockerd.globals.data_root", 0)
	if err == nil && len(dockerRootPath) > 0 {
		dockerRootStr := strings.Replace(string(dockerRootPath), "\n", "", -1)
		cmdStr := fmt.Sprintf("findmnt -T %v -v -o SOURCE|sed -n 2p", dockerRootStr)
		dockerDevicePath, err := utils.BatchOutputCmd(ctx, cmdStr, 0)
		if err == nil && len(dockerDevicePath) > 0 {
			dockerDevicePathStr = strings.Replace(string(dockerDevicePath), "\n", "", -1)
		}
		if strings.HasPrefix(dockerDevicePathStr, "overlayfs:") {
			dockerRootStr = strings.TrimPrefix(dockerDevicePathStr, "overlayfs:")
			cmdStr := fmt.Sprintf("findmnt -T %v -v -o SOURCE|sed -n 2p", dockerRootStr)
			dockerDevicePath, err := utils.BatchOutputCmd(ctx, cmdStr, 0)
			if err == nil && len(dockerDevicePath) > 0 {
				dockerDevicePathStr = strings.Replace(string(dockerDevicePath), "\n", "", -1)
			}
		}
	}
	return dockerDevicePathStr
}

func fill_part_status(ctx context.Context, disk *models.NasDiskInfo, partition *models.PartitionInfo, rootPaths []string, dockerDevicePathStr string) {
	//isReadOnly,isSystemRoot,isDockerRoot
	cmdList := []string{
		fmt.Sprintf("touch '%v/.readonly_test'", partition.MountPoint),
	}
	testReadonly := utils.BatchRun(ctx, cmdList, 0)
	if testReadonly != nil {
		partition.IsReadOnly = true
	} else {
		cmdStr := fmt.Sprintf("rm -f '%v/.readonly_test'", partition.MountPoint)
		utils.BatchOutputCmd(ctx, cmdStr, 0)
		partition.IsReadOnly = false
	}

	diskstatus.MarkSystemAndDocker(disk, partition, rootPaths, dockerDevicePathStr)
}

// https://github.com/lisaac/luci-app-diskman/blob/6ba3005ebdf1faabc7e0c4889c95caa3c153cafb/applications/luci-app-diskman/luasrc/model/diskman.lua#L147
func get_parted_info(device string, includeFree bool) (*models.NasDiskInfo, error) {
	return partedinfo.NewService(partedInfoStore{}).Read(context.Background(), device, includeFree)
}

func (store partedInfoStore) RootPaths(ctx context.Context) []string {
	return get_boot_parts(ctx)
}

func (store partedInfoStore) DockerDevicePath(ctx context.Context) string {
	return get_docker_part(ctx)
}

func (store partedInfoStore) Parted(ctx context.Context, device string) string {
	cmdStr := fmt.Sprintf("parted -s -m /dev/%v unit s print free", device)
	stdout, _, _ := utils.BatchOutErr(ctx, []string{cmdStr}, 0)
	return stdout
}

func (store partedInfoStore) MountPoint(ctx context.Context, partitionName string) string {
	return get_mount_point(partitionName)
}

func (store partedInfoStore) UUID(ctx context.Context, partitionPath string) string {
	cmdStr := fmt.Sprintf(`block info %s | grep -m1 '^%s:' | sed -nE 's/^.* UUID="([^"]+)".*$/\1/p'`, partitionPath, partitionPath)
	uuid, _, _ := utils.BatchOutErr(ctx, []string{cmdStr}, 0)
	if len(uuid) > 0 {
		return uuid
	}
	return ""
}

func (store partedInfoStore) PartitionUsage(ctx context.Context, partitionName string) (string, string) {
	used, _, usage := get_partition_usage(partitionName)
	return used, usage
}

func (store partedInfoStore) MarkMountedPartition(ctx context.Context, disk *models.NasDiskInfo, partition *models.PartitionInfo, rootPaths []string, dockerDevicePath string) {
	fill_part_status(ctx, disk, partition, rootPaths, dockerDevicePath)
}

func get_partition_usage(partition string) (string, string, string) {
	out, _ := utils.BatchOutputCmd(context.Background(), "df /dev/"+partition+" | grep -m1 '^/dev/'", 0)
	return inventory.ParsePartitionUsage(string(out))
}

func get_mount_point(partition string) string {
	out, _ := utils.BatchOutputCmd(context.Background(), "mount", 0)
	mountPoint := inventory.ParseMountPoint(string(out), partition)
	if mountPoint != "" {
		return mountPoint
	}

	return is_raid_member(partition)
}

func is_raid_member(partition string) string {
	if canAccessPath("/proc/mdstat") {
		mdstat, _ := utils.BatchOutput(context.Background(), []string{"grep md /proc/mdstat | sed 's/[][]//g'"}, 0)
		return inventory.ParseRaidMember(string(mdstat), partition)
	}
	return ""
}

// parivate
func canAccessPath(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

func findFreeMd(min int) int {
	for i := min; i < 127; i++ {
		path := fmt.Sprintf("/dev/md%v", i)
		if !canAccessPath(path) {
			return i
		}
	}
	return -1
}

func mddetail(path string) map[string]string {
	match, _ := regexp.MatchString(`^/dev/md\d+$`, path)
	if !match {
		return map[string]string{}
	}
	mdadm, _, _ := utils.BatchOutErr(context.Background(), []string{fmt.Sprintf("mdadm -D %v", path)}, 0)
	return inventory.ParseMDDetail(path, mdadm)
}

func matchStringOnce(str string, pattern string) []string {
	reg := regexp.MustCompile(pattern)
	match := reg.FindStringSubmatch(str)
	return match
}

// https://github.com/lisaac/luci-app-diskman/blob/6ba3005ebdf1faabc7e0c4889c95caa3c153cafb/applications/luci-app-diskman/luasrc/model/diskman.lua#L525
func gen_mdadm_config() error {
	return newRaidMdadmConfigService().Generate(context.Background())
}

func makeRaidPart(device string) error {
	cmdStr, err := writecommands.BuildRaidPartitionCommand(device)
	if err != nil {
		return err
	}
	_, err = utils.BatchOutputCmd(context.Background(), cmdStr, 0)
	if err != nil {
		return errors.New("make partition failed " + cmdStr)
	}
	return nil
}
