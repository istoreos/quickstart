package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
	"github.com/istoreos/quickstart/backend/modules/nas/diskcommands"
	"github.com/istoreos/quickstart/backend/modules/nas/diskinventory"
	"github.com/istoreos/quickstart/backend/modules/nas/diskstatus"
	"github.com/istoreos/quickstart/backend/utils"
)

type nasDiskCommandRunner struct{}

func (runner nasDiskCommandRunner) Run(ctx context.Context, commands []string) error {
	return utils.BatchRun(ctx, commands, 0)
}

func (runner nasDiskCommandRunner) OutErr(ctx context.Context, commands []string) (string, string, error) {
	return utils.BatchOutErr(ctx, commands, 0)
}

func (runner nasDiskCommandRunner) Output(ctx context.Context, command string) ([]byte, error) {
	return utils.BatchOutputCmd(ctx, command, 0)
}

func newNasDiskCommandService() *diskcommands.Service {
	return diskcommands.NewService(nasDiskCommandRunner{})
}

func Mount(devicePath string, mountPoint string) error {
	return newNasDiskCommandService().Mount(context.Background(), devicePath, mountPoint)
}

func UnMount(devicePath string) error {
	return newNasDiskCommandService().UnMount(context.Background(), devicePath)
}

func AddFstab(uuid string, path string, skipExisted bool) (string, error) {
	l.Debugln("AddFstab", "uuid", uuid, path)
	return newNasDiskCommandService().AddFstab(context.Background(), uuid, path, skipExisted, loadFstabMounts())
}

func loadFstabMounts() []diskcommands.FstabMount {
	uci.LoadConfig("fstab", true)
	sections, _ := uci.GetSections("fstab", "mount")
	mounts := make([]diskcommands.FstabMount, 0, len(sections))
	for _, section := range sections {
		uuid, _ := uci.GetLast("fstab", section, "uuid")
		target, _ := uci.GetLast("fstab", section, "target")
		mounts = append(mounts, diskcommands.FstabMount{
			Name:   section,
			UUID:   uuid,
			Target: target,
		})
	}
	return mounts
}

func Unmount(mountPoint string) error {
	return newNasDiskCommandService().Unmount(context.Background(), mountPoint)
}

func Erase(device string) error {
	return newNasDiskCommandService().Erase(context.Background(), device)
}

func MakePart(device string) error {
	return newNasDiskCommandService().MakePart(context.Background(), device)
}

func Ext4Partition(path string) error {
	return newNasDiskCommandService().Ext4Partition(context.Background(), path)
}

func NasDiskPartitionMount(ctx context.Context, r *http.Request) (*models.NasDiskPartitionMountResponse, error) {
	req := models.NasDiskPartitionMountRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	partition, err := newNasDiskLifecycleService().MountPartition(ctx, NasDiskPartitionMountInput{
		UUID:       req.UUID,
		Path:       req.Path,
		MountPoint: strings.TrimSpace(req.MountPoint),
	})
	if err != nil {
		return nil, err
	}
	resp := models.NasDiskPartitionMountResponse{}
	resp.Result = partition
	return &resp, nil
}

func commitFstabAndBlockMount() error {
	l.Debugln("commitFstabAndBlockMount")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return newNasDiskCommandService().CommitFstabAndBlockMount(ctx)
}

func commitFstab() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return newNasDiskCommandService().CommitFstab(ctx)
}

// 获取所有磁盘，包括raid磁盘，去除raid成员磁盘
func getAllDisks(ctx context.Context) ([]*models.NasDiskInfo, error) {
	models := []*models.NasDiskInfo{}
	disks, err := getDisksStatus(ctx)
	if err != nil {
		return nil, err
	}
	models = append(models, disks.Disks...)

	raids, err := RaidGetList(ctx)
	if err != nil {
		return nil, err
	}
	models = append(models, raids.Result.Disks...)
	return models, nil
}

func NasDiskPartitionFormatByDevicePath(ctx context.Context, devicePath string) (*models.NasDiskPartitionFormatResponse, error) {
	partition, err := newNasDiskLifecycleService().FormatByDevicePath(ctx, NasDiskFormatByDevicePathInput{
		DevicePath: devicePath,
	})
	if err != nil {
		return nil, err
	}
	resp := models.NasDiskPartitionFormatResponse{}
	resp.Result = partition
	return &resp, nil
}

func NasDiskPartitionFormat(ctx context.Context, r *http.Request) (*models.NasDiskPartitionFormatResponse, error) {
	req := models.NasDiskPartitionFormatRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return NasDiskPartitionFormatByDevicePath(ctx, req.Path)
}

func NasSanboxDisks(ctx context.Context) (*models.NasSandboxDisksResponse, error) {
	disks, err := newNasSandboxService().ListDisks(ctx)
	if err != nil {
		return nil, err
	}
	model := models.NasSandboxDisksResponseResult{}
	model.Disks = disks
	resp := models.NasSandboxDisksResponse{Result: &model}
	return &resp, nil
}

func NasSanboxStatus(ctx context.Context) (*models.NasSandboxStatusResponse, error) {
	model := models.NasSandboxStatusResponseResult{}
	status, err := newNasSandboxService().Status(ctx)
	if err != nil {
		return nil, err
	}
	model.Status = status
	resp := models.NasSandboxStatusResponse{Result: &model}
	return &resp, nil
}

func NasSanboxSubmit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	if err := newNasSandboxService().Commit(ctx); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func NasSanboxReset(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	if err := newNasSandboxService().Reset(ctx); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func NasSanboxExit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	if err := newNasSandboxService().Exit(ctx); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func NasSanboxPartitionFormat(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.NasSandboxRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	if err := newNasSandboxService().FormatPartition(ctx, req.Path); err != nil {
		return nil, err
	}
	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

// 磁盘插入后自动挂载
func NasReloadDisk() {
	if err := newNasAutoMountService().Reload(context.Background()); err != nil {
		l.Warnln("reload disk failed", err)
	}
}

func genMountPoint(name string) string {
	out, _, err := utils.BatchOutErr(context.Background(), []string{fmt.Sprintf("sh /usr/libexec/blockphy.sh %v", name)}, 0)
	if err != nil {
		return "data_" + name
	}
	return out
}

func NasDiskMountPoint(ctx context.Context, req models.NasDiskMountPointRequest) (*models.NasDiskMountPointResponse, error) {
	mountPoint, err := newNasDiskLifecycleService().GenerateMountPoint(ctx, req.Path)
	if err != nil {
		return nil, err
	}
	model := models.NasDiskMountPointResponseResult{Mountpoint: mountPoint}
	resp := models.NasDiskMountPointResponse{}
	resp.Result = &model
	return &resp, nil
}

func NasDiskInit(ctx context.Context, r *http.Request) (*models.NasDiskInitDiskResponse, error) {
	req := models.NasDiskInitDiskRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	model, err := newNasDiskLifecycleService().InitDisk(ctx, NasDiskInitInput{
		Name: req.Name,
		Path: req.Path,
	})
	if err != nil {
		return nil, err
	}
	resp := models.NasDiskInitDiskResponse{}
	resp.Result = model
	return &resp, nil
}

func NasDiskInitRest(ctx context.Context, r *http.Request) (*models.NasDiskInitDiskResponse, error) {
	req := models.NasDiskInitDiskRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, errors.New("获取参数失败")
	}
	model, err := newNasDiskLifecycleService().InitDiskRest(ctx, NasDiskInitRestInput{
		Name: req.Name,
		Path: req.Path,
	})
	if err != nil {
		return nil, err
	}
	resp := models.NasDiskInitDiskResponse{}
	resp.Result = model
	return &resp, nil
}

func getDisksStatus(ctx context.Context) (*models.NasDiskStatusResponseResult, error) {
	svc := diskstatus.NewService(
		nasDiskStatusInventoryReader{},
		nasDiskStatusPartitionMarker{
			rootPaths:        get_boot_parts(ctx),
			dockerDevicePath: get_docker_part(ctx),
		},
		nasDiskStatusRAIDReader{},
		nasDiskStatusSMARTReader{},
	)
	disks, err := svc.List(ctx)
	if err != nil {
		return nil, err
	}
	return &models.NasDiskStatusResponseResult{Disks: disks}, nil
}
func NasDiskStatus(ctx context.Context) (*models.NasDiskStatusResponse, error) {

	model, err := getDisksStatus(ctx)
	resp := models.NasDiskStatusResponse{}
	resp.Result = model
	return &resp, err
}

func NasServiceSambaStatus() []*models.NasServiceSambaInfo {
	uci.LoadConfig("samba4", true)
	sambas := make([]*models.NasServiceSambaInfo, 0)
	if sambashares, ok := uci.GetSections("samba4", "sambashare"); ok {
		for _, share := range sambashares {
			smb := &models.NasServiceSambaInfo{}
			if value, ok := uci.GetLast("samba4", share, "name"); ok {
				smb.ShareName = value
			}
			if value, ok := uci.GetLast("samba4", share, "path"); ok {
				smb.Path = value
			}
			sambas = append(sambas, smb)
		}
	}
	return sambas
}

func NasServiceStatus(ctx context.Context) (*models.NasServiceResponse, error) {
	model, err := newNasServiceStatusServiceFacade().Read(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.NasServiceResponse{}
	resp.Result = model
	return &resp, nil
}

func NasServiceLinkeaseEnable(ctx context.Context, r *http.Request) (*models.NasLinkeaseEnableResponse, error) {
	model, err := newNasLinkeaseEnableServiceFacade().Enable(ctx)
	if err != nil {
		return nil, err
	}
	resp := models.NasLinkeaseEnableResponse{}
	resp.Result = model
	return &resp, nil
}

type DiskInfoForX86Install = diskinventory.X86Install
type DiskInfoForX86Root = diskinventory.X86InstallRoot
type DiskInfoForX86Dev = diskinventory.X86InstallDev

type nasX86InstallStore struct{}

func (store nasX86InstallStore) ReadRootEnd(ctx context.Context, partitionName string) (int64, error) {
	return readRootEnd(partitionName)
}

func (store nasX86InstallStore) ReadFallbackBootRoot(ctx context.Context) (*diskinventory.X86InstallRoot, error) {
	return getBootDisk(ctx)
}

func getBootDisk(ctx context.Context) (*DiskInfoForX86Root, error) {
	// grep -Fm1 ' - squashfs /dev/ventoy2 ' /proc/self/mountinfo | cut -d' ' -f3
	// Pttype = `echo ok | parted ---pretend-input-tty -m /dev/dm-0 unit s print 2>/dev/null | grep '^/dev/dm-0:' | cut -d: -f 6`
	// end = `echo ok | parted ---pretend-input-tty -m /dev/dm-0 unit s print 2>/dev/null | grep '^3:' | cut -d: -f 2 | cut -ds -f1`
	ret, err := utils.BatchOutput(ctx, []string{
		`
if [ -e /rom/note ]; then
	rootpart="$(grep -Fm1 ' / / ' /proc/self/mountinfo | grep -F ' - squashfs ' | cut -d' ' -f3)"
else
	rootpart="$(grep -Fm1 ' / /rom ' /proc/self/mountinfo | grep -F ' - squashfs ' | cut -d' ' -f3)"
fi
[ -z "$rootpart" ] && exit 1
major=${rootpart%%:*}
minor=${rootpart##*:}
minor="$(( $minor & 0xfffc ))"
devpath="$(readlink "/sys/dev/block/$major:$minor")"
[ -z "$devpath" ] && exit 1
rootdisk="${devpath##*/}"
echo "$rootdisk"
		`,
		"exit 0",
	}, 0)
	if err != nil {
		return nil, err
	}
	rootdisk := strings.Trim(string(ret), "\n")
	if len(rootdisk) == 0 {
		return nil, errors.New("root not found")
	}
	var diskinfo *models.NasDiskInfo
	diskinfo, err = get_disk_info(rootdisk)
	if err != nil {
		return nil, err
	}
	end := int64(0)
	for _, v := range diskinfo.Childrens {
		if v.Number == 3 {
			end = int64(v.SecStart * 512)
			break
		}
	}
	if end == 0 {
		return nil, errors.New("root not found")
	}
	return &DiskInfoForX86Root{
		Name:     rootdisk,
		TranName: diskinfo.TranName,
		End:      end,
		PType:    diskinfo.PartLabelType,
	}, nil

}

func GetDiskInfoForX86Install() (*DiskInfoForX86Install, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancel()
	ret, err := utils.BatchOutputCmd(ctx, "lsblk --output NAME,MOUNTPOINT,TRAN,PTTYPE,SIZE,VENDOR,MODEL,SERIAL --json -b", 0)
	if err != nil {
		return nil, err
	}
	return diskinventory.NewX86InstallService(nasX86InstallStore{}).FromLSBLK(ctx, ret)
}

func readRootEnd(name string) (int64, error) {
	start, err := os.ReadFile(fmt.Sprintf("/sys/class/block/%s/start", name))
	if err != nil {
		return 0, err
	}
	startStr := strings.Trim(string(start), "\n")
	startInt, _ := strconv.ParseInt(startStr, 0, 64)
	if err != nil {
		return 0, err
	}
	return startInt * 512, nil
}

var readNasDiskInfoLSBLK = func(ctx context.Context) ([]byte, error) {
	return utils.BatchOutputCmd(ctx, "lsblk --output NAME,MOUNTPOINT,UUID,RO,SIZE,TYPE,FSTYPE,FSSIZE,FSUSED,FSUSE%,VENDOR,MODEL,PATH,PTTYPE,TRAN,SERIAL,LABEL --json -b", 0)
}

func getDiskInfo(ctx context.Context) ([]*diskinventory.DiskInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	ret, err := readNasDiskInfoLSBLK(ctx)
	if err != nil {
		return nil, err
	}
	return diskinventory.ParseLSBLKDisks(ret)
}

func checkMountPoint(mountPoint string) bool {
	if mountPoint == "" || mountPoint == "-" {
		return false
	}
	return true
}
