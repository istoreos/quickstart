package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/digineo/go-uci"
)

type FloatIPReader interface {
	ReadFloatIPStatus(context.Context) (FloatIPStatus, error)
}

type SpeedLimitReader interface {
	ReadSpeedLimitStatus(context.Context) (SpeedLimitStatus, error)
}

type defaultFloatIPReader struct{}

type defaultSpeedLimitReader struct{}

var _ FloatIPReader = (*defaultFloatIPReader)(nil)
var _ SpeedLimitReader = (*defaultSpeedLimitReader)(nil)

func NewDefaultFloatIPReader() FloatIPReader {
	return &defaultFloatIPReader{}
}

func NewDefaultSpeedLimitReader() SpeedLimitReader {
	return &defaultSpeedLimitReader{}
}

func (reader *defaultFloatIPReader) ReadFloatIPStatus(ctx context.Context) (FloatIPStatus, error) {
	_ = ctx
	uci.LoadConfig("floatip", true)

	role, _ := uci.GetLast("floatip", "main", "role")
	enabled, _ := uci.GetLast("floatip", "main", "enabled")
	installed, err := CheckAppIsInstalled("floatip")
	if err != nil {
		return FloatIPStatus{}, err
	}

	setIP, _ := uci.GetLast("floatip", "main", "set_ip")
	checkIP, _ := uci.GetLast("floatip", "main", "check_ip")
	checkURL, _ := uci.GetLast("floatip", "main", "check_url")
	checkURLTimeout, _ := uci.GetLast("floatip", "main", "check_url_timeout")

	return buildFloatIPStatus(installed, enabled, role, setIP, checkIP, checkURL, checkURLTimeout), nil
}

func (reader *defaultSpeedLimitReader) ReadSpeedLimitStatus(ctx context.Context) (SpeedLimitStatus, error) {
	_ = ctx
	uci.LoadConfig("eqos", true)

	enabled, _ := uci.GetLast("eqos", "@eqos[0]", "enabled")
	upload, _ := uci.GetLast("eqos", "@eqos[0]", "upload")
	download, _ := uci.GetLast("eqos", "@eqos[0]", "download")

	installed := canAccessPath(fmt.Sprintf("/usr/lib/opkg/info/%v.control", "luci-app-eqos"))
	return buildSpeedLimitStatus(installed, enabled, upload, download), nil
}

func buildFloatIPStatus(installed bool, enabled, role, setIP, checkIP, checkURL, checkURLTimeout string) FloatIPStatus {
	state := FloatIPStatus{
		Installed: installed,
		Enabled:   enabled == "1",
		Role:      role,
		SetIP:     setIP,
		CheckIP:   checkIP,
	}
	if state.Role == "main" {
		state.CheckURL = checkURL
		state.CheckURLTimeout, _ = strconv.ParseInt(checkURLTimeout, 10, 64)
	}

	return state
}

func buildSpeedLimitStatus(installed bool, enabled, upload, download string) SpeedLimitStatus {
	state := SpeedLimitStatus{
		Installed: installed,
		Enabled:   enabled == "1",
	}
	state.UploadSpeed, _ = strconv.ParseInt(upload, 10, 64)
	state.DownloadSpeed, _ = strconv.ParseInt(download, 10, 64)
	return state
}
