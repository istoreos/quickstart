package service

import "github.com/linkease/quick-start/istore-backend/models"

func toFloatGatewayModel(state FloatIPStatus) *models.LANCtrlFloatGatewayModule {
	return &models.LANCtrlFloatGatewayModule{
		Installed:       state.Installed,
		Enabled:         state.Enabled,
		Role:            state.Role,
		SetIP:           state.SetIP,
		CheckIP:         state.CheckIP,
		CheckURL:        state.CheckURL,
		CheckURLTimeout: state.CheckURLTimeout,
	}
}

func toSpeedLimitModel(state SpeedLimitStatus) *models.LANCtrlSpeedLimitModule {
	return &models.LANCtrlSpeedLimitModule{
		Installed:     state.Installed,
		Enabled:       state.Enabled,
		UploadSpeed:   state.UploadSpeed,
		DownloadSpeed: state.DownloadSpeed,
	}
}
