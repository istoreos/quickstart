package portlist

import "github.com/linkease/quick-start/istore-backend/models"

type MembershipSnapshot struct {
	InterfaceName string
	Device        string
}

func mergeMembership(ports []*models.NetworkPortInfo, memberships []MembershipSnapshot) []*models.NetworkPortInfo {
	for _, membership := range memberships {
		if len(membership.Device) == 0 {
			continue
		}
		for _, port := range ports {
			if port.Name == membership.Device || port.Master == membership.Device {
				port.InterfaceNames = append(port.InterfaceNames, membership.InterfaceName)
			}
		}
	}
	return ports
}

func buildResult(ports []*models.NetworkPortInfo) *models.NetworkPortListResponse {
	return &models.NetworkPortListResponse{
		Result: &models.NetworkPortListResponseResult{
			Ports: ports,
		},
	}
}
