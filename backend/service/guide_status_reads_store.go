package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	simplejson "github.com/bitly/go-simplejson"
	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type GuideStatusReadsReader interface {
	ReadDdnstoConfig(ctx context.Context) (*GuideDdnstoStatusSnapshot, error)
	ReadDDNSStatus(ctx context.Context) (*GuideDDNSStatusSnapshot, error)
	ReadDownloadPartitions(ctx context.Context) ([]string, error)
}

var readGuideStatusReadsBatchOutErr = func(ctx context.Context, cmds []string, timeout int) (string, string, error) {
	return utils.BatchOutErr(ctx, cmds, timeout)
}

var readGuideStatusReadsUbusCall = func(ctx context.Context, arg string) (*simplejson.Json, error) {
	return UbusCall(ctx, arg)
}

var readGuideStatusReadsCheckAppIsInstalled = CheckAppIsInstalled

var readGuideStatusReadsAllDisks = getAllDisks

var readGuideStatusReadsUciGetLast = func(config string, section string, option string) (string, bool) {
	return uci.GetLast(config, section, option)
}

type defaultGuideStatusReadsReader struct{}

func newDefaultGuideStatusReadsReader() *defaultGuideStatusReadsReader {
	return &defaultGuideStatusReadsReader{}
}

func (reader *defaultGuideStatusReadsReader) ReadDdnstoConfig(ctx context.Context) (*GuideDdnstoStatusSnapshot, error) {
	deviceID, _, _ := readGuideStatusReadsBatchOutErr(ctx, []string{
		"/usr/sbin/ddnsto -w | awk '{print $2}'",
	}, 0)
	return &GuideDdnstoStatusSnapshot{DeviceID: deviceID}, nil
}

func (reader *defaultGuideStatusReadsReader) ReadDDNSStatus(ctx context.Context) (*GuideDDNSStatusSnapshot, error) {
	snapshot := &GuideDDNSStatusSnapshot{}

	json, err := readGuideStatusReadsUbusCall(ctx, "luci.ddns get_services_status")
	if err == nil {
		ipv4NextUpdate := json.Get("myddns_ipv4").Get("next_update").MustString()
		ipv6NextUpdate := json.Get("myddns_ipv6").Get("next_update").MustString()
		uci.LoadConfig("ddns", true)
		if ipv4NextUpdate == "Stopped" || ipv4NextUpdate == "Disabled" {
			snapshot.IPV4Domain = ipv4NextUpdate
		} else {
			arr, _ := readGuideStatusReadsUciGetLast("ddns", "myddns_ipv4", "lookup_host")
			snapshot.IPV4Domain = arr
		}
		if ipv6NextUpdate == "Stopped" || ipv6NextUpdate == "Disabled" {
			snapshot.IPV6Domain = ipv6NextUpdate
		} else {
			arr, _ := readGuideStatusReadsUciGetLast("ddns", "myddns_ipv6", "lookup_host")
			snapshot.IPV6Domain = arr
		}
	}

	isInstall, err := readGuideStatusReadsCheckAppIsInstalled("ddnsto")
	if err != nil {
		return nil, err
	}
	if isInstall {
		out, _, _ := readGuideStatusReadsBatchOutErr(ctx, []string{
			"uci get ddnsto.@ddnsto[0].address",
		}, 0)
		if len(out) == 0 {
			deviceID, _, _ := readGuideStatusReadsBatchOutErr(ctx, []string{
				"/usr/sbin/ddnsto -w | awk '{print $2}'",
			}, 0)
			out = fmt.Sprintf("https://istore-%v-%v.kooldns.cn:443", deviceID, rand.Intn(100))
			_, _, err = readGuideStatusReadsBatchOutErr(ctx, []string{
				fmt.Sprintf("uci set ddnsto.@ddnsto[0].address=%v", out),
				"uci commit ddnsto",
			}, 0)
			if err != nil {
				return nil, errors.New("设置ddnsto地址失败" + out)
			}
		}
		snapshot.DdnstoDomain = out
	}

	return snapshot, nil
}

func (reader *defaultGuideStatusReadsReader) ReadDownloadPartitions(ctx context.Context) ([]string, error) {
	disks, err := readGuideStatusReadsAllDisks(ctx)
	if err != nil {
		return nil, err
	}
	partitions := make([]string, 0)
	for _, device := range disks {
		for _, part := range device.Childrens {
			if !part.IsSystemRoot && !part.IsReadOnly && part.MountPoint != "" && part.Filesystem != "squashfs" && part.Filesystem != "swap" {
				partitions = append(partitions, part.MountPoint+"/download")
			}
		}
	}
	return partitions, nil
}
