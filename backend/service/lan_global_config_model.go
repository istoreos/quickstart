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

func toSpeedLimitModel(ctx context.Context, state SpeedLimitStatus) *models.LANCtrlSpeedLimitModule {
	capabilityCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()
	bandixInstalled, bandixAvailable, bandixReason := false, false, "network_unsupported"
	if selectedLAN(ctx) == defaultLAN {
		bandixInstalled, bandixAvailable, bandixReason = bandixProviderAvailability(capabilityCtx)
	}
	nativeInstalled, nativeAvailable, nativeReason := nativeProviderCapability(capabilityCtx)
	result := &models.LANCtrlSpeedLimitModule{
		Installed:     state.Installed,
		Enabled:       state.Enabled,
		UploadSpeed:   state.UploadSpeed,
		DownloadSpeed: state.DownloadSpeed,
		Provider:      scopedRateLimitProvider(ctx),
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
	if selectedLAN(ctx) != defaultLAN {
		result.Providers = result.Providers[:1]
	}
	return result
}
