package writecommands

import (
	"errors"
	"fmt"
	"strings"
)

func NormalizeLevel(level string) string {
	switch level {
	case "linear":
		return "linear"
	case "raid0":
		return "0"
	case "raid1":
		return "1"
	case "raid5":
		return "5"
	case "raid6":
		return "6"
	case "raid10":
		return "10"
	default:
		return ""
	}
}

func ValidateMemberCount(level string, count int) error {
	if level == "5" || level == "6" {
		if count < 3 {
			return errors.New("没有足够的成员设备")
		}
		return nil
	}
	if level == "10" {
		if count < 4 {
			return errors.New("没有足够的成员设备")
		}
		return nil
	}
	if count < 2 {
		return errors.New("没有足够的成员设备")
	}
	return nil
}

func BuildCreateCommand(device string, level string, memberPaths []string) string {
	return fmt.Sprintf(
		"mdadm -C %v --run --quiet --assume-clean --homehost=any -n %v -l %v %v",
		device,
		len(memberPaths),
		level,
		strings.Join(memberPaths, " "),
	)
}

func BuildDeleteCommands(path string, members []string) []string {
	memberPaths := strings.Join(members, " ")
	return []string{
		fmt.Sprintf("mdadm --stop %v", path),
		fmt.Sprintf("mdadm --remove %v", path),
		fmt.Sprintf("mdadm --zero-superblock %v", memberPaths),
	}
}

func BuildAddCommand(path string, memberPath string) string {
	return fmt.Sprintf("mdadm -a %v %v", path, memberPath)
}

func BuildGrowCommand(path string, count uint64) string {
	return fmt.Sprintf("mdadm -G %v -n %v", path, count)
}

func BuildRemoveCommands(path string, memberPath string) []string {
	return []string{
		fmt.Sprintf("mdadm --manage %v --fail %v", path, memberPath),
		fmt.Sprintf("mdadm --manage %v --remove %v", path, memberPath),
	}
}

func BuildRecoverCommands(path string, memberPath string) []string {
	return []string{BuildAddCommand(path, memberPath)}
}

func BuildRaidPartitionCommand(device string) (string, error) {
	if device == "" {
		return "", errors.New("param missing")
	}
	return fmt.Sprintf("parted -a opt %v mkpart primary ext4 16M 100%% set 1 raid on", device), nil
}

func BuildAutoFixUUIDCommand() string {
	return `block info | grep -Fw 'TYPE="linux_raid_member"' | sed -n 's/^.* UUID="\([a-f0-9-]\+\)".*$/\1/igp' | sort -u | sed 's/\([a-f0-9]\{8\}\)-\([a-f0-9]\{4\}\)-\([a-f0-9]\{4\}\)-\([a-f0-9]\{4\}\)-\([a-f0-9]\{4\}\)\([a-f0-9]\{8\}\)/\1:\2\3:\4\5:\6/ig'`
}

func BuildDeleteFirstMdadmArrayCommand() []string {
	return []string{"uci del mdadm.@array[0]"}
}

func BuildMdadmArrayCommands(device string, uuid string) []string {
	return []string{
		"uci add mdadm array",
		fmt.Sprintf("uci set mdadm.@array[-1].device=%v", device),
		fmt.Sprintf("uci set mdadm.@array[-1].uuid=%v", uuid),
	}
}

func BuildCommitMdadmCommand() []string {
	return []string{"uci commit mdadm"}
}

func BuildRestartMdadmCommand() []string {
	return []string{"/etc/init.d/mdadm restart"}
}

func BuildEnableMdadmCommand() []string {
	return []string{"/etc/init.d/mdadm enable"}
}
