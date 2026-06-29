package service

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	simplejson "github.com/bitly/go-simplejson"
)

func UbusCall(ctx context.Context, arg string) (*simplejson.Json, error) {
	args := strings.Split(arg, " ")
	args = append([]string{"-S", "call"}, args...)
	ret, err := exec.CommandContext(ctx, "ubus", args...).Output()
	if err != nil {
		return nil, err
	}
	o := &simplejson.Json{}
	err = o.UnmarshalJSON(ret)
	return o, err
}

func UbusCallWithObject(ctx context.Context, arg string, obj interface{}) error {
	args := strings.Split(arg, " ")
	args = append([]string{"-S", "call"}, args...)
	ret, err := exec.CommandContext(ctx, "ubus", args...).Output()
	if err != nil {
		return err
	}
	return json.Unmarshal(ret, obj)
}
