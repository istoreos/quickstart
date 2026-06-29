package service

type SpeedLimitWriteInput struct {
	Action        string
	IP            string
	MAC           string
	NetworkAccess bool
	UploadSpeed   int64
	DownloadSpeed int64
	Comment       string
}

type SpeedLimitModuleInput struct {
	Enabled       bool
	UploadSpeed   int64
	DownloadSpeed int64
}

type SpeedLimitRuleMatch struct {
	Config      string
	SectionName string
	MatchIP     string
	MatchMAC    string
}

type SpeedLimitWritePlan struct {
	Input          SpeedLimitWriteInput
	DeleteSections []SpeedLimitRuleMatch
	AddSpeedLimit  bool
	AddBlockRule   bool
}
