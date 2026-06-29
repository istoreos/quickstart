package setup

import (
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
)

func PrepareQuickSetupRequest(req *models.WirelessQuickSetupRequest) error {
	if err := ValidateQuickSetupRequest(req); err != nil {
		return err
	}
	NormalizeQuickSetupRequest(req)
	return nil
}

func ValidateQuickSetupRequest(req *models.WirelessQuickSetupRequest) error {
	if req == nil ||
		req.Wifi2g == nil ||
		req.Wifi5g == nil ||
		req.Wifi2g.Ssid == "" ||
		req.Wifi2g.Key == "" ||
		req.Wifi5g.Ssid == "" ||
		req.Wifi5g.Key == "" ||
		req.Wifi2g.IfaceName == "" ||
		req.Wifi5g.IfaceName == "" {
		return errors.New("Invalid params")
	}

	if req.Wifi2g.Key != "" && (len(req.Wifi2g.Key) < 8 || len(req.Wifi2g.Key) > 20) {
		return errors.New("Invalid 2g password")
	}

	if req.Wifi5g.Key != "" && (len(req.Wifi5g.Key) < 8 || len(req.Wifi5g.Key) > 20) {
		return errors.New("Invalid 5g password")
	}

	return nil
}

func NormalizeQuickSetupRequest(req *models.WirelessQuickSetupRequest) {
	req.Wifi2g.Channel = 0
	req.Wifi2g.Encryption = "psk-mixed"
	req.Wifi2g.Htmode = "auto"
	req.Wifi2g.Hwmode = "11b/g/n/ax"
	req.Wifi2g.Hidden = false

	req.Wifi5g.Channel = 0
	req.Wifi5g.Encryption = "psk-mixed"
	req.Wifi5g.Htmode = "80"
	req.Wifi5g.Hwmode = "11a/n/ac/ax"
	req.Wifi5g.Hidden = false
}
