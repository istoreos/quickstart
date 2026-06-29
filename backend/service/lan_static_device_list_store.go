package service

import (
	"context"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
)

type LanStaticDeviceDhcpTagReader interface {
	ReadDhcpTags(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error)
}

type StaticAssignmentListReader interface {
	ReadStaticAssignments(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) ([]*models.LANStaticAssigned, error)
}

type defaultLanStaticDeviceDhcpTagReader struct {
	store DhcpConfigStore
}

type defaultStaticAssignmentListReader struct{}

var _ LanStaticDeviceDhcpTagReader = (*defaultLanStaticDeviceDhcpTagReader)(nil)
var _ StaticAssignmentListReader = (*defaultStaticAssignmentListReader)(nil)
var lanStaticDeviceListLoadConfig = uci.LoadConfig
var lanStaticDeviceListGetSections = uci.GetSections
var lanStaticDeviceListGetLast = uci.GetLast

func NewDefaultLanStaticDeviceDhcpTagReader(store DhcpConfigStore) LanStaticDeviceDhcpTagReader {
	if store == nil {
		store = NewDefaultDhcpConfigStore()
	}
	return &defaultLanStaticDeviceDhcpTagReader{store: store}
}

func NewDefaultStaticAssignmentListReader() StaticAssignmentListReader {
	return &defaultStaticAssignmentListReader{}
}

func (reader *defaultLanStaticDeviceDhcpTagReader) ReadDhcpTags(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error) {
	state, err := reader.store.LoadLanState(ctx)
	if err != nil {
		return nil, err
	}
	return buildGlobalDhcpTags(lanStatus, state), nil
}

func buildStaticAssignmentTagMap(tagList []*models.LANCtrlDhcpTagInfo) map[string]*models.LANCtrlDhcpTagInfo {
	tagMap := make(map[string]*models.LANCtrlDhcpTagInfo, len(tagList))
	for _, tag := range tagList {
		if tag == nil || tag.TagName == "" {
			continue
		}
		tagMap[tag.TagName] = tag
	}
	return tagMap
}

func buildStaticAssignmentItem(
	mac string,
	ip string,
	ipOk bool,
	name string,
	tag string,
	tagOk bool,
	tagTitle string,
	tagTitleOk bool,
	tagMap map[string]*models.LANCtrlDhcpTagInfo,
) (*models.LANStaticAssigned, bool) {
	if strings.TrimSpace(mac) == "" {
		return nil, false
	}

	item := &models.LANStaticAssigned{
		Hostname:    name,
		AssignedIP:  ip,
		AssignedMac: strings.ToUpper(strings.TrimSpace(mac)),
	}
	if ipOk && ip != "" {
		item.BindIP = true
	}

	if tagOk && tag != "" {
		if option, ok := tagMap[tag]; ok {
			item.DhcpGateway = option.Gateway
			item.TagName = option.TagName
			item.TagTitle = option.TagTitle
		}
	} else if tag == "" && tagTitleOk && tagTitle != "" {
		item.TagTitle = tagTitle
		item.TagName = ""
	}

	return item, true
}

func (reader *defaultStaticAssignmentListReader) ReadStaticAssignments(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) ([]*models.LANStaticAssigned, error) {
	_ = ctx
	lanStaticDeviceListLoadConfig("dhcp", true)

	hostSections, ok := lanStaticDeviceListGetSections("dhcp", "host")
	if !ok {
		return []*models.LANStaticAssigned{}, nil
	}

	tagMap := buildStaticAssignmentTagMap(tagList)
	items := make([]*models.LANStaticAssigned, 0, len(hostSections))
	for _, sectionName := range hostSections {
		mac, macOk := lanStaticDeviceListGetLast("dhcp", sectionName, "mac")
		ip, ipOk := lanStaticDeviceListGetLast("dhcp", sectionName, "ip")
		tag, tagOk := lanStaticDeviceListGetLast("dhcp", sectionName, "tag")
		tagTitle, tagTitleOk := lanStaticDeviceListGetLast("dhcp", sectionName, "tag_title")
		name, _ := lanStaticDeviceListGetLast("dhcp", sectionName, "name")

		item, ok := buildStaticAssignmentItem(mac, ip, ipOk, name, tag, tagOk, tagTitle, tagTitleOk, tagMap)
		if !ok || !macOk {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}
