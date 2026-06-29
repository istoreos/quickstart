package guideddns

import "testing"

func TestBuildGuideDDNSServiceNameSupportsKnownProviders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "ali", want: "aliyun.com"},
		{input: "oray", want: "oray.com"},
		{input: "dnspod", want: "dnspod.cn"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := BuildGuideDDNSServiceName(tt.input)
			if err != nil {
				t.Fatalf("unexpected service-name error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBuildGuideDDNSServiceNameRejectsInvalidProvider(t *testing.T) {
	t.Parallel()

	got, err := BuildGuideDDNSServiceName("foo")
	if err == nil || err.Error() != "serviceName参数错误foo" {
		t.Fatalf("unexpected invalid service-name result: got=%q err=%v", got, err)
	}
	if got != "" {
		t.Fatalf("expected empty service name on error, got %q", got)
	}
}

func TestBuildGuideDDNSApplyCommandsBuildsIPv4NetworkCommands(t *testing.T) {
	t.Parallel()

	got := BuildGuideDDNSApplyCommands(GuideDDNSApplyCommandInput{
		ConfigName:  "myddns_ipv4",
		UseIPv6:     "0",
		ServiceName: "aliyun.com",
		Domain:      "demo.example.com",
		UserName:    "user",
		Password:    "pass",
		Interface:   "wan",
		HasPublic:   true,
	})

	want := []string{
		"uci set ddns.myddns_ipv4=service",
		"uci set ddns.myddns_ipv4.enabled='1'",
		"uci set ddns.myddns_ipv4.use_ipv6=0",
		"uci set ddns.myddns_ipv4.service_name=aliyun.com",
		"uci set ddns.myddns_ipv4.lookup_host=demo.example.com",
		"uci set ddns.myddns_ipv4.domain=demo.example.com",
		"uci set ddns.myddns_ipv4.username=user",
		"uci set ddns.myddns_ipv4.password=pass",
		"uci set ddns.myddns_ipv4.interface=wan",
		"uci set ddns.myddns_ipv4.use_syslog=2",
		"uci set ddns.myddns_ipv4.check_unit=minutes",
		"uci set ddns.myddns_ipv4.force_unit=minutes",
		"uci set ddns.myddns_ipv4.retry_unit=seconds",
		"uci set ddns.myddns_ipv4.ip_source=network",
		"uci set ddns.myddns_ipv4.ip_network=wan",
		"uci commit ddns",
	}

	requireGuideDDNSCommands(t, got, want)
}

func TestBuildGuideDDNSApplyCommandsBuildsIPv6WebCommands(t *testing.T) {
	t.Parallel()

	got := BuildGuideDDNSApplyCommands(GuideDDNSApplyCommandInput{
		ConfigName:  "myddns_ipv6",
		UseIPv6:     "1",
		ServiceName: "dnspod.cn",
		Domain:      "ipv6.example.com",
		UserName:    "user6",
		Password:    "pass6",
		Interface:   "wan6",
		HasPublic:   false,
		IPURL:       "6.ipw.cn",
	})

	want := []string{
		"uci set ddns.myddns_ipv6=service",
		"uci set ddns.myddns_ipv6.enabled='1'",
		"uci set ddns.myddns_ipv6.use_ipv6=1",
		"uci set ddns.myddns_ipv6.service_name=dnspod.cn",
		"uci set ddns.myddns_ipv6.lookup_host=ipv6.example.com",
		"uci set ddns.myddns_ipv6.domain=ipv6.example.com",
		"uci set ddns.myddns_ipv6.username=user6",
		"uci set ddns.myddns_ipv6.password=pass6",
		"uci set ddns.myddns_ipv6.interface=wan6",
		"uci set ddns.myddns_ipv6.use_syslog=2",
		"uci set ddns.myddns_ipv6.check_unit=minutes",
		"uci set ddns.myddns_ipv6.force_unit=minutes",
		"uci set ddns.myddns_ipv6.retry_unit=seconds",
		"uci set ddns.myddns_ipv6.ip_source=web",
		"uci set ddns.myddns_ipv6.ip_url=6.ipw.cn",
		"uci del ddns.myddns_ipv6.ip_network",
		"uci commit ddns",
	}

	requireGuideDDNSCommands(t, got, want)
}

func TestResolveGuideDDNSRuntimeSelectsIPv4PublicNetwork(t *testing.T) {
	t.Parallel()

	got, err := ResolveGuideDDNSRuntime("ipv4", GuideDDNSRuntimeSnapshot{
		IPv4: &GuideDDNSRuntimeInterfaceSnapshot{
			InterfaceName: "wan",
			IP:            "1.2.3.4",
			Public:        true,
		},
		IPv6: &GuideDDNSRuntimeInterfaceSnapshot{
			InterfaceName: "wan6",
			IP:            "2001:db8::1",
			Public:        false,
		},
	})
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}

	requireGuideDDNSRuntime(t, got, GuideDDNSRuntimeResolution{
		ConfigName: "myddns_ipv4",
		UseIPv6:    "0",
		Interface:  "wan",
		HasPublic:  true,
		IPURL:      "",
	})
}

func TestResolveGuideDDNSRuntimeSelectsIPv4WebFallbackWithoutPublicNetwork(t *testing.T) {
	t.Parallel()

	got, err := ResolveGuideDDNSRuntime("ipv4", GuideDDNSRuntimeSnapshot{
		IPv4: &GuideDDNSRuntimeInterfaceSnapshot{
			InterfaceName: "wan",
			IP:            "10.0.0.2",
			Public:        false,
		},
	})
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}

	requireGuideDDNSRuntime(t, got, GuideDDNSRuntimeResolution{
		ConfigName: "myddns_ipv4",
		UseIPv6:    "0",
		Interface:  "wan",
		HasPublic:  false,
		IPURL:      "4.ipw.cn",
	})
}

func TestResolveGuideDDNSRuntimeSelectsIPv6PublicNetwork(t *testing.T) {
	t.Parallel()

	got, err := ResolveGuideDDNSRuntime("ipv6", GuideDDNSRuntimeSnapshot{
		IPv4: &GuideDDNSRuntimeInterfaceSnapshot{
			InterfaceName: "wan",
			IP:            "1.2.3.4",
			Public:        true,
		},
		IPv6: &GuideDDNSRuntimeInterfaceSnapshot{
			InterfaceName: "wan6",
			IP:            "240e::1",
			Public:        true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}

	requireGuideDDNSRuntime(t, got, GuideDDNSRuntimeResolution{
		ConfigName: "myddns_ipv6",
		UseIPv6:    "1",
		Interface:  "wan6",
		HasPublic:  true,
		IPURL:      "",
	})
}

func TestResolveGuideDDNSRuntimeSelectsIPv6WebFallbackWithoutPublicNetwork(t *testing.T) {
	t.Parallel()

	got, err := ResolveGuideDDNSRuntime("ipv6", GuideDDNSRuntimeSnapshot{
		IPv6: &GuideDDNSRuntimeInterfaceSnapshot{
			InterfaceName: "wan6",
			IP:            "fd00::2",
			Public:        false,
		},
	})
	if err != nil {
		t.Fatalf("unexpected runtime error: %v", err)
	}

	requireGuideDDNSRuntime(t, got, GuideDDNSRuntimeResolution{
		ConfigName: "myddns_ipv6",
		UseIPv6:    "1",
		Interface:  "wan6",
		HasPublic:  false,
		IPURL:      "6.ipw.cn",
	})
}

func TestResolveGuideDDNSRuntimeRejectsMissingOrInvalidIPVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ipVersion string
		snapshot  GuideDDNSRuntimeSnapshot
		wantErr   string
	}{
		{
			name:      "missing ipv4 snapshot",
			ipVersion: "ipv4",
			snapshot:  GuideDDNSRuntimeSnapshot{},
			wantErr:   "IPVersion参数错误ipv4",
		},
		{
			name:      "missing ipv6 snapshot",
			ipVersion: "ipv6",
			snapshot:  GuideDDNSRuntimeSnapshot{},
			wantErr:   "IPVersion参数错误ipv6",
		},
		{
			name:      "invalid ip version",
			ipVersion: "auto",
			snapshot: GuideDDNSRuntimeSnapshot{
				IPv4: &GuideDDNSRuntimeInterfaceSnapshot{
					InterfaceName: "wan",
					Public:        true,
				},
			},
			wantErr: "IPVersion参数错误auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ResolveGuideDDNSRuntime(tt.ipVersion, tt.snapshot)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("unexpected runtime error: got=%#v err=%v", got, err)
			}
			if got != (GuideDDNSRuntimeResolution{}) {
				t.Fatalf("expected empty runtime resolution on error, got %#v", got)
			}
		})
	}
}

func requireGuideDDNSCommands(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d commands, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected cmd[%d]: want %q, got %q", i, want[i], got[i])
		}
	}
}

func requireGuideDDNSRuntime(t *testing.T, got, want GuideDDNSRuntimeResolution) {
	t.Helper()

	if got != want {
		t.Fatalf("unexpected runtime resolution: want %#v, got %#v", want, got)
	}
}
