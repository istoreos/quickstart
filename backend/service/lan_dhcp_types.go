package service

import "github.com/linkease/quick-start/istore-backend/models"

type DhcpGatewayInput struct {
	DhcpEnabled bool
	DhcpGateway string
}

type DhcpTagConfigInput struct {
	Action     string
	TagName    string
	TagTitle   string
	DhcpOption []string
}

type DhcpTagRecord struct {
	TagName     string
	TagTitle    string
	AutoCreated bool
	Gateway     string
	DhcpOption  []string
}

type LanStatusSnapshot struct {
	LanAddr          string
	Nexthop          string
	IsDefaultGateway bool
}

type LanDhcpState struct {
	DhcpOptions []string
	DhcpIgnore  bool
	Tags        []DhcpTagRecord
	FloatIP     *FloatIPSnapshot
}

type FloatIPSnapshot struct {
	Enabled bool
	SetIP   string
	CheckIP string
}

type DhcpTagPlan struct {
	Tags        []DhcpTagRecord
	DhcpGateway string
	DhcpEnabled bool
}

func toModelDhcpTags(tags []DhcpTagRecord) []*models.LANCtrlDhcpTagInfo {
	out := make([]*models.LANCtrlDhcpTagInfo, 0, len(tags))
	for _, tag := range tags {
		options := append([]string(nil), tag.DhcpOption...)
		out = append(out, &models.LANCtrlDhcpTagInfo{
			TagName:     tag.TagName,
			TagTitle:    tag.TagTitle,
			Gateway:     tag.Gateway,
			AutoCreated: tag.AutoCreated,
			DhcpOption:  options,
		})
	}
	return out
}
