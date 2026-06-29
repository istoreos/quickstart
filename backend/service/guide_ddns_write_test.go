package service

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
)

var guideDDNSWriterTestMu sync.Mutex

func TestDefaultGuideDDNSWriterDelegatesDdnstoEnableAndAddressWrites(t *testing.T) {
	guideDDNSWriterTestMu.Lock()
	defer guideDDNSWriterTestMu.Unlock()

	originalBatchOutErr := writeGuideDDNSBatchOutErr
	defer func() { writeGuideDDNSBatchOutErr = originalBatchOutErr }()

	calls := make([][]string, 0)
	writeGuideDDNSBatchOutErr = func(ctx context.Context, cmds []string, timeout int) (string, string, error) {
		calls = append(calls, append([]string(nil), cmds...))
		return "", "", nil
	}

	writer := newDefaultGuideDDNSWriter()
	if _, err := writer.EnableDdnsto(ctxbgDDNS(), GuideDdnstoEnableInput{Token: "token-abc"}); err != nil {
		t.Fatalf("unexpected ddnsto enable error: %v", err)
	}
	if _, err := writer.UpdateDdnstoAddress(ctxbgDDNS(), GuideDdnstoAddressInput{Address: "https://demo.example.com"}); err != nil {
		t.Fatalf("unexpected ddnsto address error: %v", err)
	}

	expectedEnable := []string{
		"uci set ddnsto.@ddnsto[0].enabled=1",
		"uci set ddnsto.@ddnsto[0].token=token-abc",
		"uci commit ddnsto",
		"/etc/init.d/ddnsto restart",
	}
	expectedAddress := []string{
		"uci set ddnsto.@ddnsto[0].address=https://demo.example.com",
		"uci commit ddnsto",
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 batchOutErr calls, got %d", len(calls))
	}
	if !reflect.DeepEqual(calls[0], expectedEnable) {
		t.Fatalf("unexpected ddnsto enable commands: %#v", calls[0])
	}
	if !reflect.DeepEqual(calls[1], expectedAddress) {
		t.Fatalf("unexpected ddnsto address commands: %#v", calls[1])
	}
}

func TestDefaultGuideDDNSWriterDelegatesDDNSCommandsAndPropagatesErrors(t *testing.T) {
	guideDDNSWriterTestMu.Lock()
	defer guideDDNSWriterTestMu.Unlock()

	originalBatchOutErr := writeGuideDDNSBatchOutErr
	originalBatchRun := writeGuideDDNSBatchRun
	defer func() {
		writeGuideDDNSBatchOutErr = originalBatchOutErr
		writeGuideDDNSBatchRun = originalBatchRun
	}()

	outErr := errors.New("stderr failed")
	writeGuideDDNSBatchOutErr = func(ctx context.Context, cmds []string, timeout int) (string, string, error) {
		return "", "boom", outErr
	}
	writer := newDefaultGuideDDNSWriter()
	if _, err := writer.EnableDdnsto(ctxbgDDNS(), GuideDdnstoEnableInput{Token: "x"}); !errors.Is(err, outErr) {
		t.Fatalf("expected ddnsto enable error, got %v", err)
	}
	if _, err := writer.UpdateDdnstoAddress(ctxbgDDNS(), GuideDdnstoAddressInput{Address: "a"}); !errors.Is(err, outErr) {
		t.Fatalf("expected ddnsto address error, got %v", err)
	}

	var gotDDNS []string
	runErr := errors.New("run failed")
	writeGuideDDNSBatchRun = func(ctx context.Context, cmds []string, timeout int) error {
		gotDDNS = append([]string(nil), cmds...)
		return runErr
	}
	if err := writer.ApplyDDNSConfig(ctxbgDDNS(), []string{"uci set ddns.foo=bar", "uci commit ddns"}); !errors.Is(err, runErr) {
		t.Fatalf("expected ddns apply error, got %v", err)
	}
	if !reflect.DeepEqual(gotDDNS, []string{"uci set ddns.foo=bar", "uci commit ddns"}) {
		t.Fatalf("unexpected ddns commands: %#v", gotDDNS)
	}
}

func TestDefaultGuideDDNSWriterDelegatesDDNSStart(t *testing.T) {
	guideDDNSWriterTestMu.Lock()
	defer guideDDNSWriterTestMu.Unlock()

	originalBatchRun := writeGuideDDNSBatchRun
	defer func() { writeGuideDDNSBatchRun = originalBatchRun }()

	var got []string
	writeGuideDDNSBatchRun = func(ctx context.Context, cmds []string, timeout int) error {
		got = append([]string(nil), cmds...)
		return nil
	}

	writer := newDefaultGuideDDNSWriter()
	if err := writer.StartDDNSService(ctxbgDDNS(), "myddns_ipv4"); err != nil {
		t.Fatalf("unexpected ddns start error: %v", err)
	}
	expected := []string{"/usr/lib/ddns/dynamic_dns_lucihelper.sh -S myddns_ipv4 -- start"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected ddns start commands: %#v", got)
	}
}

func ctxbgDDNS() context.Context {
	return context.Background()
}
