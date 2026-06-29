package service

import (
	"context"
	"net/http"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
	guidesetup "github.com/linkease/quick-start/istore-backend/modules/guidecore/setup"
	"github.com/linkease/quick-start/istore-backend/utils"
)

func (backend *ServiceBackend) GuideNeedSetup(ctx context.Context, r *http.Request) (*models.GuideNeedSetupResponse, error) {
	uci.LoadConfig("quickstart", true)
	input := guidesetup.NeedSetupInput{
		HasWireless: checkHasWireless(),
	}
	if val, ok := uci.GetLast("quickstart", "main", "show_guide"); ok && val == "1" {
		input.ShowGuide = true
		if val, ok := uci.GetLast("quickstart", "main", "setup"); ok && val == "1" {
			input.SetupMarked = true
		} else {
			needFromShadow, err := checkNeedSetupFromShadow()
			input.PasswordCheckOK = err == nil
			input.PasswordUnchanged = needFromShadow
		}
	}

	return &models.GuideNeedSetupResponse{
		Result: guidesetup.BuildNeedSetupInfo(input),
	}, nil
}

func (backend *ServiceBackend) GuideFinishSetup(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	uci.LoadConfig("quickstart", true)
	if val, ok := uci.GetLast("quickstart", "main", "show_guide"); ok && val == "1" {
		utils.BatchRun(ctx, []string{"uci -q del quickstart.main.show_guide", "uci commit quickstart"}, 0)
	}
	resp := models.SDKNormalResponse{}
	return &resp, nil
}
