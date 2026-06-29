package inventory

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Store interface {
	ReadMDStat(ctx context.Context) (string, error)
	ReadDisk(ctx context.Context, name string) (*models.NasDiskInfo, error)
	ReadMDDetail(ctx context.Context, path string) (map[string]string, error)
	ReadDetailText(ctx context.Context, path string) (string, error)
	ReadAllDisks(ctx context.Context) ([]*models.NasDiskInfo, error)
	ReadRaidMember(path string) string
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) List(ctx context.Context) ([]*models.NasDiskInfo, error) {
	mdstat, err := svc.store.ReadMDStat(ctx)
	if err != nil {
		return nil, errors.New("读取raid配置文件失败")
	}

	disks := make([]*models.NasDiskInfo, 0)
	lines := strings.SplitAfter(mdstat, "\n")
	for _, line := range lines {
		match := matchStringOnce(line, `^(md\d+) : (.+)`)
		if match == nil {
			continue
		}

		mdpath := match[1]
		memberTokens := matchAllWithStr(match[2], `\S+`)
		members := make([]string, 0, len(memberTokens))
		for _, token := range memberTokens {
			member := string(token[0])
			memberMatch := matchStringOnce(member, `^(\S+)\[\d+\]`)
			if memberMatch != nil {
				member = "/dev/" + memberMatch[1]
			}
			members = append(members, member)
		}
		if len(members) == 0 {
			continue
		}

		active := members[0]
		level := "-"
		if active == "active" && len(members) > 1 {
			level = members[1]
		}

		deviceInfo, err := svc.store.ReadDisk(ctx, mdpath)
		if deviceInfo == nil || err != nil {
			return nil, err
		}
		deviceInfo = cloneDiskInfo(deviceInfo)
		deviceInfo.Level = level
		deviceInfo.Active = active
		deviceInfo.Name = mdpath
		deviceInfo.Path = "/dev/" + mdpath
		if len(members) > 2 {
			deviceInfo.Members = members[2:]
		} else {
			deviceInfo.Members = nil
		}

		detail, err := svc.store.ReadMDDetail(ctx, deviceInfo.Path)
		if err != nil {
			return nil, err
		}
		deviceInfo.Status = detail["State"]
		if matchStatus := matchStringOnce(mdstat, `\n`+mdpath+` :.*?\n.*?\n\s+\[[=>\.]+?\]\s+(.*)`); matchStatus != nil {
			deviceInfo.RebuildStatus = matchStatus[1]
		}
		for _, partition := range deviceInfo.Childrens {
			if partition == nil {
				continue
			}
			partition.SecStart = 0
			partition.SecEnd = 0
		}

		disks = append(disks, deviceInfo)
	}
	return disks, nil
}

func (svc *Service) Detail(ctx context.Context, path string) (string, error) {
	detail, err := svc.store.ReadDetailText(ctx, path)
	if err != nil {
		return "", errors.New("获取raid详情失败")
	}
	return detail, nil
}

func (svc *Service) CreateList(ctx context.Context) ([]*models.RaidMemberInfo, error) {
	disks, err := svc.store.ReadAllDisks(ctx)
	if err != nil {
		return nil, err
	}

	members := make([]*models.RaidMemberInfo, 0)
	for _, disk := range disks {
		if disk == nil || disk.IsSystemRoot {
			continue
		}
		if len(disk.Childrens) > 0 {
			isRaid := false
			isMounted := false
			for _, part := range disk.Childrens {
				if part == nil {
					continue
				}
				if len(svc.store.ReadRaidMember(part.Path)) > 0 {
					isRaid = true
					break
				}
				if part.MountPoint != "" {
					isMounted = true
					break
				}
			}
			if isRaid || isMounted {
				continue
			}
		}
		members = append(members, &models.RaidMemberInfo{
			Name:    disk.Name,
			Path:    disk.Path,
			Model:   disk.VenderModel,
			SizeStr: disk.Size,
		})
	}
	return members, nil
}

func matchAllWithStr(str string, pattern string) [][][]byte {
	reg := regexp.MustCompile(pattern)
	return reg.FindAllSubmatch([]byte(str), -1)
}

func matchStringOnce(str string, pattern string) []string {
	reg := regexp.MustCompile(pattern)
	return reg.FindStringSubmatch(str)
}

func cloneDiskInfo(disk *models.NasDiskInfo) *models.NasDiskInfo {
	if disk == nil {
		return nil
	}
	clone := *disk
	if disk.Members != nil {
		clone.Members = append([]string(nil), disk.Members...)
	}
	if disk.Childrens != nil {
		clone.Childrens = make([]*models.PartitionInfo, len(disk.Childrens))
		for idx, partition := range disk.Childrens {
			if partition == nil {
				continue
			}
			partitionClone := *partition
			clone.Childrens[idx] = &partitionClone
		}
	}
	return &clone
}
