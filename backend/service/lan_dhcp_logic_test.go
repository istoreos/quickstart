package service

import "testing"

func TestBuildAutoDhcpPlan(t *testing.T) {
	t.Run("normal LAN", func(t *testing.T) {
		plan := BuildAutoDhcpPlan(
			LanStatusSnapshot{
				LanAddr:          "192.168.100.1",
				Nexthop:          "192.168.100.254",
				IsDefaultGateway: false,
			},
			&LanDhcpState{},
		)

		assertDhcpTagPlan(t, plan, "", true, []DhcpTagRecord{
			{
				TagName:     "t_auto_c0a86401",
				TagTitle:    "default",
				AutoCreated: true,
				Gateway:     "192.168.100.1",
				DhcpOption:  []string{"3,192.168.100.1", "6,192.168.100.1"},
			},
		})
	})

	t.Run("gateway is myself", func(t *testing.T) {
		plan := BuildAutoDhcpPlan(
			LanStatusSnapshot{
				LanAddr:          "192.168.100.1",
				Nexthop:          "192.168.100.254",
				IsDefaultGateway: true,
			},
			&LanDhcpState{},
		)

		assertDhcpTagPlan(t, plan, "", true, []DhcpTagRecord{
			{
				TagName:     "t_auto_c0a86401",
				TagTitle:    "default",
				AutoCreated: true,
				Gateway:     "192.168.100.1",
				DhcpOption:  []string{"3,192.168.100.1", "6,192.168.100.1"},
			},
			{
				TagName:     "t_auto_c0a864fe",
				TagTitle:    "parent",
				AutoCreated: true,
				Gateway:     "192.168.100.254",
				DhcpOption:  []string{"3,192.168.100.254", "6,192.168.100.254"},
			},
		})
	})

	t.Run("gateway is parent", func(t *testing.T) {
		plan := BuildAutoDhcpPlan(
			LanStatusSnapshot{
				LanAddr:          "192.168.100.1",
				Nexthop:          "192.168.100.254",
				IsDefaultGateway: true,
			},
			&LanDhcpState{
				DhcpOptions: []string{"6,192.168.100.254", "3,192.168.100.254"},
			},
		)

		assertDhcpTagPlan(t, plan, "192.168.100.254", true, []DhcpTagRecord{
			{
				TagName:     "t_auto_c0a864fe",
				TagTitle:    "default",
				AutoCreated: true,
				Gateway:     "192.168.100.254",
				DhcpOption:  []string{"3,192.168.100.254", "6,192.168.100.254"},
			},
			{
				TagName:     "t_auto_c0a86401",
				TagTitle:    "myself",
				AutoCreated: true,
				Gateway:     "192.168.100.1",
				DhcpOption:  []string{"3,192.168.100.1", "6,192.168.100.1"},
			},
		})
	})

	t.Run("dhcp disabled keeps default-tag fallback", func(t *testing.T) {
		plan := BuildAutoDhcpPlan(
			LanStatusSnapshot{
				LanAddr:          "192.168.100.1",
				Nexthop:          "192.168.100.254",
				IsDefaultGateway: true,
			},
			&LanDhcpState{
				DhcpIgnore:  true,
				DhcpOptions: []string{"3,192.168.100.254", "6,192.168.100.254"},
			},
		)

		assertDhcpTagPlan(t, plan, "", false, []DhcpTagRecord{
			{
				TagName:     "t_auto_c0a86401",
				TagTitle:    "default",
				AutoCreated: true,
				Gateway:     "192.168.100.1",
				DhcpOption:  []string{"3,192.168.100.1", "6,192.168.100.1"},
			},
		})
	})

	t.Run("malformed or missing dhcp option falls back safely", func(t *testing.T) {
		cases := []struct {
			name    string
			options []string
		}{
			{
				name:    "malformed option",
				options: []string{"3", "6,192.168.100.254", "3,192.168.100.1,extra"},
			},
			{
				name:    "missing option",
				options: nil,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				plan := BuildAutoDhcpPlan(
					LanStatusSnapshot{
						LanAddr:          "192.168.100.1",
						Nexthop:          "192.168.100.254",
						IsDefaultGateway: true,
					},
					&LanDhcpState{
						DhcpOptions: tc.options,
					},
				)

				assertDhcpTagPlan(t, plan, "", true, []DhcpTagRecord{
					{
						TagName:     "t_auto_c0a86401",
						TagTitle:    "default",
						AutoCreated: true,
						Gateway:     "192.168.100.1",
						DhcpOption:  []string{"3,192.168.100.1", "6,192.168.100.1"},
					},
					{
						TagName:     "t_auto_c0a864fe",
						TagTitle:    "parent",
						AutoCreated: true,
						Gateway:     "192.168.100.254",
						DhcpOption:  []string{"3,192.168.100.254", "6,192.168.100.254"},
					},
				})
			})
		}
	})

	t.Run("gateway option equal to lan address is treated as default gateway", func(t *testing.T) {
		plan := BuildAutoDhcpPlan(
			LanStatusSnapshot{
				LanAddr:          "192.168.100.1",
				Nexthop:          "192.168.100.254",
				IsDefaultGateway: true,
			},
			&LanDhcpState{
				DhcpOptions: []string{"3,192.168.100.1", "6,192.168.100.1"},
			},
		)

		assertDhcpTagPlan(t, plan, "", true, []DhcpTagRecord{
			{
				TagName:     "t_auto_c0a86401",
				TagTitle:    "default",
				AutoCreated: true,
				Gateway:     "192.168.100.1",
				DhcpOption:  []string{"3,192.168.100.1", "6,192.168.100.1"},
			},
			{
				TagName:     "t_auto_c0a864fe",
				TagTitle:    "parent",
				AutoCreated: true,
				Gateway:     "192.168.100.254",
				DhcpOption:  []string{"3,192.168.100.254", "6,192.168.100.254"},
			},
		})
	})
}

func assertDhcpTagPlan(t *testing.T, plan DhcpTagPlan, wantGateway string, wantEnabled bool, wantTags []DhcpTagRecord) {
	t.Helper()

	if plan.DhcpGateway != wantGateway {
		t.Fatalf("expected gateway %q, got %q", wantGateway, plan.DhcpGateway)
	}
	if plan.DhcpEnabled != wantEnabled {
		t.Fatalf("expected DhcpEnabled=%t, got %t", wantEnabled, plan.DhcpEnabled)
	}
	if len(plan.Tags) != len(wantTags) {
		t.Fatalf("expected %d tags, got %d", len(wantTags), len(plan.Tags))
	}

	for i := range wantTags {
		got := plan.Tags[i]
		want := wantTags[i]
		if got.TagName != want.TagName {
			t.Fatalf("expected tag name %q at index %d, got %q", want.TagName, i, got.TagName)
		}
		if got.TagTitle != want.TagTitle {
			t.Fatalf("expected tag title %q at index %d, got %q", want.TagTitle, i, got.TagTitle)
		}
		if got.AutoCreated != want.AutoCreated {
			t.Fatalf("expected AutoCreated=%t at index %d, got %t", want.AutoCreated, i, got.AutoCreated)
		}
		if got.Gateway != want.Gateway {
			t.Fatalf("expected gateway %q at index %d, got %q", want.Gateway, i, got.Gateway)
		}
		if len(got.DhcpOption) != len(want.DhcpOption) {
			t.Fatalf("expected %d dhcp options at index %d, got %d", len(want.DhcpOption), i, len(got.DhcpOption))
		}
		for j := range want.DhcpOption {
			if got.DhcpOption[j] != want.DhcpOption[j] {
				t.Fatalf("expected dhcp option %q at index [%d][%d], got %q", want.DhcpOption[j], i, j, got.DhcpOption[j])
			}
		}
	}
}
