package service

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
	networkbasics "github.com/linkease/quick-start/istore-backend/modules/guidecore/networkbasics"
	guidesetup "github.com/linkease/quick-start/istore-backend/modules/guidecore/setup"
	dockertransfer "github.com/linkease/quick-start/istore-backend/modules/guidestorage/dockertransfer"
	"github.com/linkease/quick-start/istore-backend/utils"
)

const (
	NetworkErrorDnsNotSetting = -1001
	NetworkErrorWanNotExists  = -1011
)

const (
	NetworkErrorMessageWanNotExists = "WAN not exists"
)

var readGuideSetupShadowFile = ioutil.ReadFile

var runGuideNetworkBasicsUCICommands = func(ctx context.Context, cmdList []string) error {
	return utils.BatchRun(ctx, cmdList, 0)
}

var runGuideDockerTransferPathOutput = func(ctx context.Context, cmd string) ([]byte, error) {
	return utils.BatchOutputCmd(ctx, cmd, 0)
}

func LanSetting(ip string, netmask string, ipOnly bool) error {
	if net.ParseIP(ip) == nil {
		return errors.New("ip不合法")
	}
	mask, err := SubNetMaskToLen(netmask)
	if err != nil {
		return err
	}
	if mask == 0 {
		return errors.New("mask不合法")
	}
	cmdList := []string{
		"uci delete network.lan.ipaddr",
		"uci delete network.lan.netmask",
		fmt.Sprintf("uci set network.lan.ipaddr='%v'", ip),
		fmt.Sprintf("uci set network.lan.netmask='%v'", netmask),
	}
	modifiedUci := []string{"network"}

	if !ipOnly {
		uci.LoadConfig("network", true)
		// check if lan was dhcp client
		if value, ok := uci.GetLast("network", "lan", "proto"); ok && value == "dhcp" {
			cmdList = append(cmdList, "uci set network.lan.proto=static")
			cmdList = append(cmdList, "uci set dhcp.lan.ignore=1")
			modifiedUci = append(modifiedUci, "dhcp")
		}
	}
	err = utils.BatchRun(context.Background(), cmdList, 0)
	if err != nil {
		return err
	}
	return utils.UciCommitAndApply(context.Background(), modifiedUci)
}

func uciGetNetDNSClient(net string) (string, []string) {
	uci.LoadConfig("network", true)
	dnsProto := "auto"
	dnsIPs := []string{}
	if value, ok := uci.GetLast("network", net, "peerdns"); ok && value == "0" {
		dnsProto = "manual"
	}
	if dnsProto == "manual" {
		if values, ok := uci.Get("network", net, "dns"); ok {
			dnsIPs = values
		}
	}
	return dnsProto, dnsIPs
}

func uciGetFirewallZoneByName(name string) string {
	uci.LoadConfig("firewall", true)
	secs, _ := uci.GetSections("firewall", "zone")
	for _, v := range secs {
		name0, _ := uci.GetLast("firewall", v, "name")
		if name == name0 {
			return v
		}
	}
	return ""
}

func uciSetLanMasq(ctx context.Context, enable bool) bool {
	lanzone := uciGetFirewallZoneByName("lan")
	if lanzone != "" {
		masq := ""
		if enable {
			masq = "1"
		}
		cmdList := []string{
			fmt.Sprintf("uci -q set 'firewall.%v.masq=%v'", lanzone, masq),
		}
		utils.BatchRun(ctx, cmdList, 0)
		return true
	}
	return false
}

func uciGetIPAndMask(net string) (string, string) {
	uci.LoadConfig("network", true)
	if values, ok := uci.Get("network", net, "ipaddr"); ok {
		if len(values) > 0 {
			ipComs := strings.Split(values[0], "/")
			if len(ipComs) == 2 {
				mask, _ := strconv.ParseInt(ipComs[1], 10, 8)
				return ipComs[0], LenToSubNetMask(int(mask))
			} else {
				mask, _ := uci.Get("network", net, "netmask")
				return values[0], mask[0]
			}
		}
	}
	return "", ""
}

func isLanDHCPServerEnabled() bool {
	uci.LoadConfig("dhcp", true)
	if value, ok := uci.GetLast("dhcp", "lan", "ignore"); !ok || value != "1" {
		return true
	}
	return false
}

func enabledLanDHCPServer(ctx context.Context) {
	uci.LoadConfig("dhcp", true)
	var dhcpStartStr string
	if value, ok := uci.GetLast("dhcp", "lan", "start"); ok {
		dhcpStartStr = value
	}

	var dhcplimitStr string
	if value, ok := uci.GetLast("dhcp", "lan", "limit"); ok {
		dhcplimitStr = value
	}

	var leasetime string
	if value, ok := uci.GetLast("dhcp", "lan", "leasetime"); ok {
		leasetime = value
	}

	cmdList := []string{
		// dhcpv6 server
		"uci set dhcp.lan.dhcpv6=server",
		"uci set dhcp.lan.ra=server",
		"uci set dhcp.lan.ra_slaac=1",
		"uci del dhcp.lan.ra_flags",
		"uci add_list dhcp.lan.ra_flags=managed-config",
		"uci add_list dhcp.lan.ra_flags=other-config",

		// dhcpv4 server
		"uci del dhcp.lan.ignore",
		"uci set dhcp.lan.dhcpv4=server",
	}
	if dhcpStartStr == "" || dhcplimitStr == "" {
		cmdList = append(cmdList, "uci set dhcp.lan.start=100")
		cmdList = append(cmdList, "uci set dhcp.lan.limit=150")
	}
	if leasetime == "" {
		cmdList = append(cmdList, "uci set dhcp.lan.leasetime=12h")
	}
	utils.BatchRun(ctx, cmdList, 0)
}

func IsWanPresent() bool {
	uci.LoadConfig("network", true)
	// check if has wan
	value, ok := uci.GetLast("network", "wan", "proto")
	return ok && value != ""
}

func WanAccessAllow(ctx context.Context) {
	utils.BatchRun(ctx, []string{
		"/etc/init.d/wan_drop stop >/dev/null 2>&1",
		"/etc/init.d/wan_drop disable >/dev/null 2>&1",
		"WAN_ZONE=`uci show firewall | grep -E '^firewall\\.@zone\\[[0-9]+\\]\\.name=' | grep wan | head -n1 | head -c -12`",
		"if [ \"`uci get ${WAN_ZONE}.input`\" != \"ACCEPT\" ]; then",
		"	uci -q batch <<-EOF >/dev/null",
		"		set ${WAN_ZONE}.input=ACCEPT",
		"		commit firewall",
		"EOF",
		"	/etc/init.d/firewall reload >/dev/null",
		"fi",
	}, 0)
}

func checkNeedSetupFromShadow() (bool, error) {
	b0, err0 := readGuideSetupShadowFile("/rom/etc/shadow")
	if err0 != nil {
		return false, err0
	}
	b1, err1 := readGuideSetupShadowFile("/etc/shadow")
	if err1 != nil {
		return false, err1
	}
	return guidesetup.NeedSetupFromShadow(b0, b1), nil
}

func mkDlDir(ctx context.Context, path string) error {
	cmdList := []string{
		fmt.Sprintf("if [ ! -d '%v' ]; then mkdir -p '%v'; fi", path, path),
		fmt.Sprintf("chmod 777 '%v'", path),
	}

	return utils.BatchRun(ctx, cmdList, 0)
}

func uciGet(ctx context.Context, location string) (string, error) {
	value, err := utils.BatchOutputCmd(ctx, "uci get '"+location+"'", 0)
	if err != nil {
		return "", err
	}
	return strings.Replace(string(value), "\n", "", -1), nil
}
func getAria2Status(ctx context.Context) (*models.GuideDownloadAria2Info, error) {

	model := models.GuideDownloadAria2Info{}

	//使用go的uci无法获取到arai2的配置，必须还是用命令获取
	cmdStr := "[ -e /etc/init.d/aria2 ] && echo true || echo false"
	aria2Installed, _ := utils.BatchOutput(ctx, []string{cmdStr}, 0)
	aria2InstalledStr := strings.Replace(string(aria2Installed), "\n", "", -1)

	if aria2InstalledStr == "false" {
		model.Status = "not installed"
		return &model, nil
	} else {
		if CheckAppIsRunning("aria2c") {
			model.Status = "running"
		} else {
			model.Status = "stopped"
		}
	}

	aria2ConfigPath, err := uciGet(ctx, "aria2.main.config_dir")
	if err != nil {
		l.Debugln("获取aria2 config dir 失败")
	}
	model.ConfigPath = aria2ConfigPath

	aria2DownloadPath, err := uciGet(ctx, "aria2.main.dir")
	if err != nil {
		l.Debugln("获取aria2 dir 失败")
	}
	model.DownloadPath = aria2DownloadPath

	aria2RpcSecret, err := uciGet(ctx, "aria2.main.rpc_secret")
	if err != nil {
		l.Debugln("获取aria2 rpc secret 失败")
	}
	model.RPCToken = aria2RpcSecret
	model.WebPath = "/ariang"

	aria2RpcPort, err := uciGet(ctx, "aria2.main.rpc_listen_port")
	if err != nil {
		l.Debugln("获取aria2 rpc port 失败")
	}
	if len(aria2RpcPort) > 0 {
		aria2RpcPortInt, _ := strconv.ParseUint(aria2RpcPort, 10, 32)
		model.RPCPort = uint32(aria2RpcPortInt)
	} else {
		model.RPCPort = 6800
	}

	return &model, nil
}

func getQbittorrentStatus(ctx context.Context) (*models.GuideDownloadQbittorrentInfo, error) {

	model := models.GuideDownloadQbittorrentInfo{}

	//使用go的uci无法获取到arai2的配置，必须还是用命令获取
	aria2Installed, _ := utils.BatchOutputCmd(ctx, "[ -e /etc/init.d/qbittorrent ] && echo true || echo false", 0)
	aria2InstalledStr := strings.Replace(string(aria2Installed), "\n", "", -1)

	if aria2InstalledStr == "false" {
		model.Status = "not installed"
		return &model, nil
	} else {
		if CheckAppIsRunning("qbittorrent") {
			model.Status = "running"
		} else {
			model.Status = "stopped"
		}
	}

	aria2ConfigPath, err := uciGet(ctx, "qbittorrent.main.profile")
	if err != nil {
		l.Debugln("qbit config dir 失败")
	}
	model.ConfigPath = aria2ConfigPath

	aria2DownloadPath, err := uciGet(ctx, "qbittorrent.main.SavePath")
	if err != nil {
		l.Debugln("qbit save path 为空")
	}
	model.DownloadPath = aria2DownloadPath

	port, err := uciGet(ctx, "qbittorrent.main.Port")
	if err != nil {
		l.Debugln("qbit port 为空")
	} else {
		model.WebPath = ":" + port
	}

	return &model, nil
}

func getTransmissionStatus(ctx context.Context) (*models.GuideDownloadTransmissionInfo, error) {

	model := models.GuideDownloadTransmissionInfo{}

	//使用go的uci无法获取到arai2的配置，必须还是用命令获取
	aria2Installed, _ := utils.BatchOutputCmd(ctx, "[ -e /etc/init.d/transmission ] && echo true || echo false", 0)
	aria2InstalledStr := strings.Replace(string(aria2Installed), "\n", "", -1)

	if aria2InstalledStr == "false" {
		model.Status = "not installed"
		return &model, nil
	} else {
		if CheckAppIsRunning("transmission") {
			model.Status = "running"
		} else {
			model.Status = "stopped"
		}
	}

	aria2ConfigPath, err := uciGet(ctx, "transmission.@transmission[0].config_dir")
	if err != nil {
		l.Debugln("transmission config dir 失败")
	}
	model.ConfigPath = aria2ConfigPath

	aria2DownloadPath, err := uciGet(ctx, "transmission.@transmission[0].download_dir")
	if err != nil {
		l.Debugln("transmission save path 为空")
	} else {
		model.DownloadPath = aria2DownloadPath
	}

	port, err := uciGet(ctx, "transmission.@transmission[0].rpc_port")
	if err != nil {
		l.Debugln("transmission port 为空")
	} else {
		model.WebPath = ":" + port
	}

	return &model, nil
}

func DockerTransferTool(path string) error {
	resp, err := newGuideDockerTransferFacade().Transfer(context.Background(), GuideDockerTransferInput{
		Path:         path,
		Force:        true,
		OverwriteDir: false,
	})
	if err != nil {
		return err
	}
	if resp != nil && resp.Result != nil && resp.Result.EmptyPathWarning {
		return errors.New("目标路径不为空")
	}
	return nil
}

func uciSetPppoeWithoutCommit(ctx context.Context, account string, password string) error {
	err := runGuideNetworkBasicsUCICommands(ctx, networkbasics.BuildPPPoECommands(account, password))
	if err != nil {
		return err
	}
	return nil
}

func uciSetInterfaceWithoutCommit(ctx context.Context, it string, proto string, netmask string, ip string, gateway string) error {
	batches, buildErr := networkbasics.BuildInterfaceCommandBatches(networkbasics.InterfaceInput{
		InterfaceName: it,
		Proto:         proto,
		Netmask:       netmask,
		IP:            ip,
		Gateway:       gateway,
	})
	for _, cmdList := range batches {
		if err := runGuideNetworkBasicsUCICommands(ctx, cmdList); err != nil {
			l.Warnln(err)
		}
	}

	return buildErr
}

func checkDockerPath(ctx context.Context, path string, orginPath string) error {

	if path == orginPath {
		return dockertransfer.ValidatePathSnapshot(dockertransfer.PathSnapshot{
			TargetPath:   path,
			OriginPath:   orginPath,
			TargetSource: "target",
		})
	}

	//目标路径不存在的时候，必须先创建文件夹，不然后面无法获取到目标文件系统信息
	cmdStr := fmt.Sprintf("[ -d '%v' ] || mkdir -p '%v'", path, path)
	_, err := runGuideDockerTransferPathOutput(ctx, cmdStr)
	if err != nil {
		return err
	}

	cmdStr = fmt.Sprintf("findmnt -T '%v' -o SOURCE,FSTYPE --json", path)
	ret, err := runGuideDockerTransferPathOutput(ctx, cmdStr)
	if err != nil {
		l.Debugln(cmdStr, err.Error())
		return err
	}
	findmntJson := &simplejson.Json{}
	err = findmntJson.UnmarshalJSON(ret)
	if err != nil {
		l.Debugln("UnmarshalJSON", err.Error())
		return err
	}

	fileSystems := findmntJson.Get("filesystems").MustArray()
	if len(fileSystems) < 1 {
		return errors.New("路径信息获取失败")
	}
	source := findmntJson.Get("filesystems").GetIndex(0).Get("source").MustString()
	fstype := findmntJson.Get("filesystems").GetIndex(0).Get("fstype").MustString()

	rootPath, err := runGuideDockerTransferPathOutput(ctx, "findmnt -T /overlay -o SOURCE|sed -n 2p")
	if err != nil {
		l.Debugln(err.Error())
		return err
	}
	rootPathStr := strings.Replace(string(rootPath), "\n", "", -1)

	cmdStr = fmt.Sprintf("findmnt -T '%v' -o SOURCE|sed -n 2p", orginPath)
	rootPath, err = runGuideDockerTransferPathOutput(ctx, cmdStr)
	if err != nil {
		l.Debugln(err.Error())
		return err
	}
	originRootPathStr := strings.Replace(string(rootPath), "\n", "", -1)
	if err := dockertransfer.ValidatePathSnapshot(dockertransfer.PathSnapshot{
		TargetPath:    path,
		OriginPath:    orginPath,
		TargetSource:  source,
		TargetFSType:  fstype,
		OverlaySource: rootPathStr,
		OriginSource:  originRootPathStr,
	}); err != nil {
		return err
	}
	l.Debugln(originRootPathStr, " ==? ", source)

	return nil
}

func transferDockerPath(ctx context.Context, targetPath string, force bool, overwriteDir bool, orginPath string) (*models.GuideDockerTransferResponseResult, error) {
	//要判断目标空间，是否还是原来的根分区，或者是无效的分区
	//目标文件夹是否存在
	cmdStr := fmt.Sprintf("[ -d '%v' ] && echo true || echo false", targetPath)
	pathResult, err := utils.BatchOutputCmd(ctx, cmdStr, 0)
	if err != nil {
		return nil, err
	}
	pathResultStr := strings.Replace(string(pathResult), "\n", "", -1)
	l.Debugln("文件夹是否存在", pathResultStr)

	if pathResultStr == "true" {
		//文件夹存在，如果里面还有多余的文件，也不能直接覆盖，提示用户
		cmdStr := fmt.Sprintf("ls -A '%v' | wc -l", targetPath)
		l.Debugln("文件夹判断", cmdStr)
		dirResult, err := utils.BatchOutputCmd(ctx, cmdStr, 0)
		if err != nil {
			return nil, err
		}
		dirResultStr := strings.Replace(string(dirResult), "\n", "", -1)
		l.Debugln("文件是否为空", dirResultStr != "0")

		if dirResultStr != "0" {
			if force {
				if overwriteDir {
					l.Debugln("覆盖迁移", overwriteDir)
					cmdList := []string{
						fmt.Sprintf("rm -r '%v'", targetPath),
					}
					l.Debugln("删除文件夹")
					err = utils.BatchRun(ctx, cmdList, 0)
					if err != nil {
						return nil, err
					}
					l.Debugln("创建文件夹")
					cmdList = []string{
						fmt.Sprintf("mkdir -p '%v'", targetPath),
					}
					err = utils.BatchRun(ctx, cmdList, 0)
					if err != nil {
						return nil, err
					}
					l.Debugln("复制文件", orginPath, targetPath)
					cmdList = []string{
						fmt.Sprintf("cp -a '%v/.' '%v/'", orginPath, targetPath),
					}
					err = utils.BatchRun(ctx, cmdList, 0)
					if err != nil {
						return nil, err
					}
					l.Debugln("文件迁移完成")
				} else {
					l.Debugln("仅迁移目录", overwriteDir)
					//仅迁移目录，不复制文件，只修改路径
				}

			} else {
				l.Debugln("文件夹不为空，弹框提示")
				return dockertransfer.BuildEmptyTargetDirectoryWarning(targetPath)
			}
		} else {
			l.Debugln("文件夹为空，复制文件")

			cmdList := []string{
				fmt.Sprintf("cp -a '%v/.' '%v/'", orginPath, targetPath),
			}
			err = utils.BatchRun(ctx, cmdList, 0)
			if err != nil {
				return nil, errors.New("复制迁移docker目录失败，请检查目标路径")
			}
		}

	} else {
		l.Debugln("文件夹不存在，复制文件")

		cmdList := []string{
			fmt.Sprintf("mkdir -p '%v'", targetPath),
			fmt.Sprintf("cp -a '%v/.' '%v/'", orginPath, targetPath),
		}
		err = utils.BatchRun(ctx, cmdList, 0)
		if err != nil {
			return nil, errors.New("复制迁移docker目录失败，请检查目标路径")
		}
	}
	return nil, nil
}

func uciSetDNSWithoutCommit(ctx context.Context, it string, dnsProto string, dnsIPs []string) error {
	batches := networkbasics.BuildDNSCommandBatches(it, dnsProto, dnsIPs)
	for i, cmdList := range batches {
		err := runGuideNetworkBasicsUCICommands(ctx, cmdList)
		if err != nil && i > 0 {
			return err
		}
	}
	return nil
}
