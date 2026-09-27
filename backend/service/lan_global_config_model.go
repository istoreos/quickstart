package service

import (
	"context"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

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
	capabilityCtx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	bandixInstalled, bandixAvailable, bandixReason := bandixProviderAvailability(capabilityCtx)
	nativeInstalled, nativeAvailable, nativeReason := nativeProviderCapability(capabilityCtx)
	return &models.LANCtrlSpeedLimitModule{
		Installed:     state.Installed,
		Enabled:       state.Enabled,
		UploadSpeed:   state.UploadSpeed,
		DownloadSpeed: state.DownloadSpeed,
		Provider:      effectiveRateLimitProvider(defaultRateLimitProviderPath),
		Providers: []*models.RateLimitProviderOption{
			{ID: nativePolicyProviderName, Available: nativeAvailable, Installed: nativeInstalled, Reason: nativeReason},
			{ID: "eqos", Available: state.Installed, Installed: state.Installed, Reason: func() string {
				if state.Installed {
					return ""
				}
				return "dependency_not_installed"
			}()},
			{ID: "bandix", Available: bandixAvailable, Installed: bandixInstalled, Reason: bandixReason},
		},
	}
}
