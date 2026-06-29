package service

import "testing"

func TestBuildBlockedDeviceRuleMapsLegacyFields(t *testing.T) {
	item, ok := buildBlockedDeviceRule(" aa:bb:cc:dd:ee:ff ", "BL_console", "REJECT")
	if !ok {
		t.Fatal("expected BL_ + REJECT rule to be accepted")
	}
	if item.Mac != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("Mac = %q, want %q", item.Mac, "AA:BB:CC:DD:EE:FF")
	}
	if !item.Enabled {
		t.Fatal("expected blocked item to be enabled")
	}
	if item.NetworkAccess {
		t.Fatal("expected blocked item to disable network access")
	}
}

func TestBuildBlockedDeviceRuleRejectsNonBlockingRules(t *testing.T) {
	if _, ok := buildBlockedDeviceRule("", "BL_console", "REJECT"); ok {
		t.Fatal("expected missing mac to be rejected")
	}
	if _, ok := buildBlockedDeviceRule("   ", "BL_console", "REJECT"); ok {
		t.Fatal("expected whitespace-only mac to be rejected")
	}
	if _, ok := buildBlockedDeviceRule("aa:bb:cc:dd:ee:ff", "ALLOW_console", "REJECT"); ok {
		t.Fatal("expected non BL_ name to be rejected")
	}
	if _, ok := buildBlockedDeviceRule("aa:bb:cc:dd:ee:ff", "BL_console", "ACCEPT"); ok {
		t.Fatal("expected non REJECT target to be rejected")
	}
}

func TestBuildSpeedLimitRuleParsesLegacyFields(t *testing.T) {
	item := buildSpeedLimitRule("192.168.100.8", "2048", "4096", "kid tablet")
	if item.IP != "192.168.100.8" {
		t.Fatalf("IP = %q, want %q", item.IP, "192.168.100.8")
	}
	if item.UploadSpeed != 2048 {
		t.Fatalf("UploadSpeed = %d, want %d", item.UploadSpeed, 2048)
	}
	if item.DownloadSpeed != 4096 {
		t.Fatalf("DownloadSpeed = %d, want %d", item.DownloadSpeed, 4096)
	}
	if item.Comment != "kid tablet" {
		t.Fatalf("Comment = %q, want %q", item.Comment, "kid tablet")
	}
	if !item.NetworkAccess {
		t.Fatal("expected speedlimit rule to default network access true")
	}
}

func TestBuildSpeedLimitRuleUsesZeroWhenRateParsingFails(t *testing.T) {
	item := buildSpeedLimitRule("192.168.100.9", "oops", "bad", "fallback")
	if item.UploadSpeed != 0 {
		t.Fatalf("UploadSpeed = %d, want 0", item.UploadSpeed)
	}
	if item.DownloadSpeed != 0 {
		t.Fatalf("DownloadSpeed = %d, want 0", item.DownloadSpeed)
	}
}

func TestPreloadLanSpeedLimitRuleConfigsUsesRuleStoreLoadHook(t *testing.T) {
	original := lanSpeedLimitRuleLoadConfig
	t.Cleanup(func() {
		lanSpeedLimitRuleLoadConfig = original
	})

	var calls []string
	lanSpeedLimitRuleLoadConfig = func(config string, overwrite bool) error {
		if !overwrite {
			t.Fatalf("expected overwrite=true for %s", config)
		}
		calls = append(calls, config)
		return nil
	}

	preloadLanSpeedLimitRuleConfigs()

	if len(calls) != 2 || calls[0] != "eqos" || calls[1] != "firewall" {
		t.Fatalf("unexpected preload calls: %+v", calls)
	}
}
