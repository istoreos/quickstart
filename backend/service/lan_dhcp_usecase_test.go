package service

import (
	"context"
	"errors"
	"testing"
)

type fakeDhcpConfigStore struct {
	applyTagConfigFn     func(ctx context.Context, input DhcpTagConfigInput) error
	applyGatewayConfigFn func(ctx context.Context, input DhcpGatewayInput, lanStatus LanStatusSnapshot) error
}

func (store *fakeDhcpConfigStore) LoadLanState(ctx context.Context) (*LanDhcpState, error) {
	_ = ctx
	return nil, nil
}

func (store *fakeDhcpConfigStore) ApplyTagConfig(ctx context.Context, input DhcpTagConfigInput) error {
	if store.applyTagConfigFn != nil {
		return store.applyTagConfigFn(ctx, input)
	}
	return nil
}

func (store *fakeDhcpConfigStore) ApplyGatewayConfig(ctx context.Context, input DhcpGatewayInput, lanStatus LanStatusSnapshot) error {
	if store.applyGatewayConfigFn != nil {
		return store.applyGatewayConfigFn(ctx, input, lanStatus)
	}
	return nil
}

type fakeLanStatusReader struct {
	readLanStatusFn func(ctx context.Context) (LanStatusSnapshot, error)
}

func (reader *fakeLanStatusReader) ReadLanStatus(ctx context.Context) (LanStatusSnapshot, error) {
	if reader.readLanStatusFn != nil {
		return reader.readLanStatusFn(ctx)
	}
	return LanStatusSnapshot{}, nil
}

func TestSetDhcpGatewayRejectsInvalidIP(t *testing.T) {
	t.Parallel()

	storeCalled := false
	svc := NewLanDhcpService(
		&fakeDhcpConfigStore{
			applyGatewayConfigFn: func(ctx context.Context, input DhcpGatewayInput, lanStatus LanStatusSnapshot) error {
				_, _, _ = ctx, input, lanStatus
				storeCalled = true
				return nil
			},
		},
		&fakeLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{LanAddr: "192.168.1.1"}, nil
			},
		},
	)

	err := svc.SetDhcpGateway(context.Background(), DhcpGatewayInput{
		DhcpEnabled: true,
		DhcpGateway: "not-an-ip",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "dhcp gateway is not a valid IP address" {
		t.Fatalf("expected invalid gateway error, got %q", err.Error())
	}
	if storeCalled {
		t.Fatal("expected store not to be called")
	}
}

func TestSetDhcpGatewayForwardsExplicitGateway(t *testing.T) {
	t.Parallel()

	var gotInput DhcpGatewayInput
	var gotStatus LanStatusSnapshot
	svc := NewLanDhcpService(
		&fakeDhcpConfigStore{
			applyGatewayConfigFn: func(ctx context.Context, input DhcpGatewayInput, lanStatus LanStatusSnapshot) error {
				_ = ctx
				gotInput = input
				gotStatus = lanStatus
				return nil
			},
		},
		&fakeLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{LanAddr: "192.168.1.1", Nexthop: "192.168.1.254"}, nil
			},
		},
	)

	err := svc.SetDhcpGateway(context.Background(), DhcpGatewayInput{
		DhcpEnabled: true,
		DhcpGateway: "192.168.1.9",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotInput.DhcpGateway != "192.168.1.9" {
		t.Fatalf("expected explicit gateway to be forwarded, got %q", gotInput.DhcpGateway)
	}
	if !gotInput.DhcpEnabled {
		t.Fatal("expected DhcpEnabled to be forwarded")
	}
	if gotStatus.LanAddr != "192.168.1.1" {
		t.Fatalf("expected lan status to be forwarded, got %#v", gotStatus)
	}
}

func TestSetDhcpGatewayDefaultsEmptyGatewayToLanAddr(t *testing.T) {
	t.Parallel()

	var gotInput DhcpGatewayInput
	svc := NewLanDhcpService(
		&fakeDhcpConfigStore{
			applyGatewayConfigFn: func(ctx context.Context, input DhcpGatewayInput, lanStatus LanStatusSnapshot) error {
				_, _ = ctx, lanStatus
				gotInput = input
				return nil
			},
		},
		&fakeLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{LanAddr: "10.0.0.1"}, nil
			},
		},
	)

	err := svc.SetDhcpGateway(context.Background(), DhcpGatewayInput{DhcpEnabled: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotInput.DhcpGateway != "10.0.0.1" {
		t.Fatalf("expected empty gateway to default to LanAddr, got %q", gotInput.DhcpGateway)
	}
}

func TestSetDhcpGatewayPropagatesLanStatusError(t *testing.T) {
	t.Parallel()

	storeCalled := false
	wantErr := errors.New("read lan status failed")
	svc := NewLanDhcpService(
		&fakeDhcpConfigStore{
			applyGatewayConfigFn: func(ctx context.Context, input DhcpGatewayInput, lanStatus LanStatusSnapshot) error {
				_, _, _ = ctx, input, lanStatus
				storeCalled = true
				return nil
			},
		},
		&fakeLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{}, wantErr
			},
		},
	)

	err := svc.SetDhcpGateway(context.Background(), DhcpGatewayInput{DhcpEnabled: true})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected lan status error to propagate, got %v", err)
	}
	if storeCalled {
		t.Fatal("expected store not to be called")
	}
}

func TestSetDhcpTagsRejectsAutoPrefix(t *testing.T) {
	t.Parallel()

	storeCalled := false
	svc := NewLanDhcpService(
		&fakeDhcpConfigStore{
			applyTagConfigFn: func(ctx context.Context, input DhcpTagConfigInput) error {
				_, _ = ctx, input
				storeCalled = true
				return nil
			},
		},
		&fakeLanStatusReader{},
	)

	err := svc.SetDhcpTags(context.Background(), DhcpTagConfigInput{
		Action:     "add",
		TagName:    "t_auto_reserved",
		TagTitle:   "reserved",
		DhcpOption: []string{"3,192.168.1.1"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "tag name error using t_auto_" {
		t.Fatalf("expected auto-prefix error, got %q", err.Error())
	}
	if storeCalled {
		t.Fatal("expected store not to be called")
	}
}

func TestSetDhcpTagsForwardsValidInput(t *testing.T) {
	t.Parallel()

	var gotInput DhcpTagConfigInput
	svc := NewLanDhcpService(
		&fakeDhcpConfigStore{
			applyTagConfigFn: func(ctx context.Context, input DhcpTagConfigInput) error {
				_ = ctx
				gotInput = input
				return nil
			},
		},
		&fakeLanStatusReader{},
	)

	input := DhcpTagConfigInput{
		Action:     "modify",
		TagName:    "guest",
		TagTitle:   "Guest",
		DhcpOption: []string{"3,192.168.1.1", "6,192.168.1.1"},
	}
	err := svc.SetDhcpTags(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotInput.Action != input.Action {
		t.Fatalf("expected action %q to pass through, got %q", input.Action, gotInput.Action)
	}
	if gotInput.TagName != input.TagName {
		t.Fatalf("expected tag name %q, got %q", input.TagName, gotInput.TagName)
	}
	if gotInput.TagTitle != input.TagTitle {
		t.Fatalf("expected tag title %q, got %q", input.TagTitle, gotInput.TagTitle)
	}
	if len(gotInput.DhcpOption) != len(input.DhcpOption) {
		t.Fatalf("expected %d dhcp options, got %d", len(input.DhcpOption), len(gotInput.DhcpOption))
	}
	for i := range input.DhcpOption {
		if gotInput.DhcpOption[i] != input.DhcpOption[i] {
			t.Fatalf("expected dhcp option %q at index %d, got %q", input.DhcpOption[i], i, gotInput.DhcpOption[i])
		}
	}
}

func TestSetDhcpTagsRejectsMissingParams(t *testing.T) {
	t.Parallel()

	storeCalled := false
	svc := NewLanDhcpService(
		&fakeDhcpConfigStore{
			applyTagConfigFn: func(ctx context.Context, input DhcpTagConfigInput) error {
				_, _ = ctx, input
				storeCalled = true
				return nil
			},
		},
		&fakeLanStatusReader{},
	)

	err := svc.SetDhcpTags(context.Background(), DhcpTagConfigInput{
		Action:   "add",
		TagName:  "guest",
		TagTitle: "",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "invalid params" {
		t.Fatalf("expected invalid params error, got %q", err.Error())
	}
	if storeCalled {
		t.Fatal("expected store not to be called")
	}
}

func TestToModelDhcpTagsCopiesValues(t *testing.T) {
	t.Run("nil and empty input", func(t *testing.T) {
		cases := []struct {
			name  string
			input []DhcpTagRecord
		}{
			{
				name:  "nil",
				input: nil,
			},
			{
				name:  "empty",
				input: []DhcpTagRecord{},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got := toModelDhcpTags(tc.input)
				if got == nil {
					t.Fatal("expected non-nil slice")
				}
				if len(got) != 0 {
					t.Fatalf("expected empty result, got %d items", len(got))
				}
			})
		}
	})

	t.Run("copies records including empty options", func(t *testing.T) {
		input := []DhcpTagRecord{
			{
				TagName:     "t_auto_192_168_1_1",
				TagTitle:    "default",
				Gateway:     "192.168.1.1",
				AutoCreated: true,
				DhcpOption:  []string{"3,192.168.1.1", "6,192.168.1.1"},
			},
			{
				TagName:     "custom",
				TagTitle:    "custom-title",
				Gateway:     "",
				AutoCreated: false,
				DhcpOption:  []string{},
			},
		}

		got := toModelDhcpTags(input)
		if len(got) != len(input) {
			t.Fatalf("expected %d tags, got %d", len(input), len(got))
		}

		for i := range input {
			if got[i] == nil {
				t.Fatalf("expected non-nil tag at index %d", i)
			}
			if got[i].TagName != input[i].TagName {
				t.Fatalf("expected tag name %q at index %d, got %q", input[i].TagName, i, got[i].TagName)
			}
			if got[i].TagTitle != input[i].TagTitle {
				t.Fatalf("expected tag title %q at index %d, got %q", input[i].TagTitle, i, got[i].TagTitle)
			}
			if got[i].Gateway != input[i].Gateway {
				t.Fatalf("expected gateway %q at index %d, got %q", input[i].Gateway, i, got[i].Gateway)
			}
			if got[i].AutoCreated != input[i].AutoCreated {
				t.Fatalf("expected auto created %t at index %d, got %t", input[i].AutoCreated, i, got[i].AutoCreated)
			}
			if len(got[i].DhcpOption) != len(input[i].DhcpOption) {
				t.Fatalf("expected %d dhcp options at index %d, got %d", len(input[i].DhcpOption), i, len(got[i].DhcpOption))
			}
			for j := range input[i].DhcpOption {
				if got[i].DhcpOption[j] != input[i].DhcpOption[j] {
					t.Fatalf("expected dhcp option %q at index [%d][%d], got %q", input[i].DhcpOption[j], i, j, got[i].DhcpOption[j])
				}
			}
		}
	})

	t.Run("copies option slices without aliasing input", func(t *testing.T) {
		input := []DhcpTagRecord{
			{
				TagName:     "t_auto_192_168_1_1",
				TagTitle:    "default",
				Gateway:     "192.168.1.1",
				AutoCreated: true,
				DhcpOption:  []string{"3,192.168.1.1", "6,192.168.1.1"},
			},
			{
				TagName:     "custom",
				TagTitle:    "custom-title",
				Gateway:     "192.168.1.2",
				AutoCreated: false,
				DhcpOption:  []string{"3,192.168.1.2", "6,192.168.1.2"},
			},
		}

		got := toModelDhcpTags(input)
		if len(got) != len(input) {
			t.Fatalf("expected %d tags, got %d", len(input), len(got))
		}

		input[0].DhcpOption[0] = "3,10.0.0.1"
		input[1].DhcpOption[1] = "6,10.0.0.2"
		if got[0].DhcpOption[0] != "3,192.168.1.1" {
			t.Fatalf("expected copied dhcp option to ignore input mutation, got %q", got[0].DhcpOption[0])
		}
		if got[1].DhcpOption[1] != "6,192.168.1.2" {
			t.Fatalf("expected copied dhcp option to ignore input mutation, got %q", got[1].DhcpOption[1])
		}

		got[0].DhcpOption[1] = "6,172.16.0.1"
		got[1].DhcpOption[0] = "3,172.16.0.2"
		if input[0].DhcpOption[1] != "6,192.168.1.1" {
			t.Fatalf("expected original input to ignore output mutation, got %q", input[0].DhcpOption[1])
		}
		if input[1].DhcpOption[0] != "3,192.168.1.2" {
			t.Fatalf("expected original input to ignore output mutation, got %q", input[1].DhcpOption[0])
		}
	})
}
