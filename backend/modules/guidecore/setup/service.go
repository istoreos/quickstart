package setup

import (
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
)

type NeedSetupInput struct {
	ShowGuide         bool
	SetupMarked       bool
	PasswordCheckOK   bool
	PasswordUnchanged bool
	HasWireless       bool
}

func NeedSetupFromShadow(romShadow []byte, currentShadow []byte) bool {
	return rootShadowLine(romShadow) == rootShadowLine(currentShadow)
}

func BuildNeedSetupInfo(input NeedSetupInput) *models.GuideNeedSetupInfo {
	need := input.ShowGuide &&
		!input.SetupMarked &&
		input.PasswordCheckOK &&
		input.PasswordUnchanged

	return &models.GuideNeedSetupInfo{
		Need: need,
		Wifi: input.HasWireless,
	}
}

func rootShadowLine(input []byte) string {
	for _, line := range strings.Split(string(input), "\n") {
		if strings.Contains(line, "root:") {
			return line
		}
	}
	return ""
}
