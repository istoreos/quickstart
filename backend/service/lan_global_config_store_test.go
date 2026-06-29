package service

import "testing"

func TestBuildFloatIPStatusMainRoleIncludesCheckURLFields(t *testing.T) {
	state := buildFloatIPStatus(true, "1", "main", "192.168.1.2", "1.1.1.1", "https://example.com/check", "30")

	if !state.Installed {
		t.Fatalf("Installed = %v, want true", state.Installed)
	}
	if !state.Enabled {
		t.Fatalf("Enabled = %v, want true", state.Enabled)
	}
	if state.Role != "main" {
		t.Fatalf("Role = %q, want %q", state.Role, "main")
	}
	if state.SetIP != "192.168.1.2" {
		t.Fatalf("SetIP = %q, want %q", state.SetIP, "192.168.1.2")
	}
	if state.CheckIP != "1.1.1.1" {
		t.Fatalf("CheckIP = %q, want %q", state.CheckIP, "1.1.1.1")
	}
	if state.CheckURL != "https://example.com/check" {
		t.Fatalf("CheckURL = %q, want %q", state.CheckURL, "https://example.com/check")
	}
	if state.CheckURLTimeout != 30 {
		t.Fatalf("CheckURLTimeout = %d, want %d", state.CheckURLTimeout, 30)
	}
}

func TestBuildFloatIPStatusNonMainRoleSkipsCheckURLFields(t *testing.T) {
	state := buildFloatIPStatus(true, "1", "backup", "192.168.1.2", "1.1.1.1", "https://example.com/check", "30")

	if state.CheckURL != "" {
		t.Fatalf("CheckURL = %q, want empty", state.CheckURL)
	}
	if state.CheckURLTimeout != 0 {
		t.Fatalf("CheckURLTimeout = %d, want 0", state.CheckURLTimeout)
	}
}

func TestBuildSpeedLimitStatusParsesNumericFields(t *testing.T) {
	state := buildSpeedLimitStatus(true, "1", "2048", "8192")

	if !state.Installed {
		t.Fatalf("Installed = %v, want true", state.Installed)
	}
	if !state.Enabled {
		t.Fatalf("Enabled = %v, want true", state.Enabled)
	}
	if state.UploadSpeed != 2048 {
		t.Fatalf("UploadSpeed = %d, want %d", state.UploadSpeed, 2048)
	}
	if state.DownloadSpeed != 8192 {
		t.Fatalf("DownloadSpeed = %d, want %d", state.DownloadSpeed, 8192)
	}
}

func TestToFloatGatewayModel(t *testing.T) {
	state := FloatIPStatus{
		Installed:       true,
		Enabled:         true,
		Role:            "main",
		SetIP:           "192.168.1.2",
		CheckIP:         "1.1.1.1",
		CheckURL:        "https://example.com/check",
		CheckURLTimeout: 15,
	}

	model := toFloatGatewayModel(state)

	if model.Installed != state.Installed {
		t.Fatalf("Installed = %v, want %v", model.Installed, state.Installed)
	}
	if model.Enabled != state.Enabled {
		t.Fatalf("Enabled = %v, want %v", model.Enabled, state.Enabled)
	}
	if model.Role != state.Role {
		t.Fatalf("Role = %q, want %q", model.Role, state.Role)
	}
	if model.SetIP != state.SetIP {
		t.Fatalf("SetIP = %q, want %q", model.SetIP, state.SetIP)
	}
	if model.CheckIP != state.CheckIP {
		t.Fatalf("CheckIP = %q, want %q", model.CheckIP, state.CheckIP)
	}
	if model.CheckURL != state.CheckURL {
		t.Fatalf("CheckURL = %q, want %q", model.CheckURL, state.CheckURL)
	}
	if model.CheckURLTimeout != state.CheckURLTimeout {
		t.Fatalf("CheckURLTimeout = %d, want %d", model.CheckURLTimeout, state.CheckURLTimeout)
	}
}

func TestToSpeedLimitModel(t *testing.T) {
	state := SpeedLimitStatus{
		Installed:     true,
		Enabled:       true,
		UploadSpeed:   2048,
		DownloadSpeed: 8192,
	}

	model := toSpeedLimitModel(state)

	if model.Installed != state.Installed {
		t.Fatalf("Installed = %v, want %v", model.Installed, state.Installed)
	}
	if model.Enabled != state.Enabled {
		t.Fatalf("Enabled = %v, want %v", model.Enabled, state.Enabled)
	}
	if model.UploadSpeed != state.UploadSpeed {
		t.Fatalf("UploadSpeed = %d, want %d", model.UploadSpeed, state.UploadSpeed)
	}
	if model.DownloadSpeed != state.DownloadSpeed {
		t.Fatalf("DownloadSpeed = %d, want %d", model.DownloadSpeed, state.DownloadSpeed)
	}
}
