package service

import "github.com/linkease/quick-start/istore-backend/modules/lancontrol/floatgateway"

type FloatGatewayWriteInput = floatgateway.Input

type FloatGatewayStateSnapshot = floatgateway.StateSnapshot

type FloatGatewayDhcpTagSnapshot = floatgateway.DhcpTagSnapshot

type FloatGatewayDhcpHostSnapshot = floatgateway.DhcpHostSnapshot

type FloatGatewayDhcpCleanupPlan = floatgateway.DhcpCleanupPlan

type FloatGatewayWriteExecutionPlan struct {
	FloatCommands []string
	CleanupPlan   FloatGatewayDhcpCleanupPlan
}
