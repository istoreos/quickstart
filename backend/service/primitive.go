package service

import (
	"context"

	"github.com/bitly/go-simplejson"
)

func System(ctx context.Context) (*simplejson.Json, error) {
	o, err := UbusCall(ctx, "system info")
	if err != nil {
		return nil, err
	}
	return o, nil
}

func Dhcp(ctx context.Context) (interface{}, error) {
	o, err := UbusCall(ctx, "dhcp ipv4leases")
	if err != nil {
		return nil, err
	}
	return o, nil
}

func Board(ctx context.Context) (interface{}, error) {
	o, err := UbusCall(ctx, "system board")
	if err != nil {
		return nil, err
	}
	return o, nil
}

func NetworkBasic(ctx context.Context) (interface{}, error) {
	o, err := UbusCall(ctx, "uci get {\"config\":\"network\"}")
	if err != nil {
		return nil, err
	}
	return o, nil
}

func NetworkInterfaces(ctx context.Context) (interface{}, error) {
	o, err := UbusCall(ctx, "network.device status")
	if err != nil {
		return nil, err
	}
	return o, nil
}

func NetworkLan(ctx context.Context) (interface{}, error) {
	o, err := UbusCall(ctx, "network.interface.lan status")
	if err != nil {
		return nil, err
	}
	return o, nil
}

func NetworkWan(ctx context.Context) (interface{}, error) {
	o, err := UbusCall(ctx, "network.interface.wan status")
	if err != nil {
		return nil, err
	}
	return o, nil
}
